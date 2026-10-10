// The MCP SDK is an OPTIONAL dependency. It MUST be loaded via the subpath
// entrypoints the published package actually ships. The package root
// (`@modelcontextprotocol/sdk`) resolves to `./dist/esm/index.js`, which no
// published version ships, so a root import always throws ERR_MODULE_NOT_FOUND
// and silently disables the bridge (spec 27).
type MCPClientCtor = typeof import("@modelcontextprotocol/sdk/client/index.js")["Client"];
type MCPStreamableHttpTransportCtor =
  typeof import("@modelcontextprotocol/sdk/client/streamableHttp.js")["StreamableHTTPClientTransport"];
type MCPStdioTransportCtor =
  typeof import("@modelcontextprotocol/sdk/client/stdio.js")["StdioClientTransport"];

let Client: MCPClientCtor | null = null;
let StreamableHTTPClientTransport: MCPStreamableHttpTransportCtor | null = null;
let StdioClientTransport: MCPStdioTransportCtor | null = null;
let SDK_LOAD_ERROR: string | null = null;

try {
  ({ Client } = await import("@modelcontextprotocol/sdk/client/index.js"));
  ({ StreamableHTTPClientTransport } = await import(
    "@modelcontextprotocol/sdk/client/streamableHttp.js"
  ));
  ({ StdioClientTransport } = await import("@modelcontextprotocol/sdk/client/stdio.js"));
} catch (err) {
  const code = (err as { code?: string })?.code;
  const message = String((err as Error)?.message ?? err).split("\n")[0];
  // Only a genuinely ABSENT optional dependency degrades the bridge quietly.
  // A failed resolution of a subpath that SHOULD exist inside an installed
  // package is a real defect (spec 27, R27.2): report it, never swallow it.
  const packageAbsent =
    code === "ERR_MODULE_NOT_FOUND" &&
    message.includes("Cannot find package '@modelcontextprotocol/sdk'");
  Client = null;
  StreamableHTTPClientTransport = null;
  StdioClientTransport = null;
  if (!packageAbsent) {
    SDK_LOAD_ERROR = `MCP SDK present but not loadable: ${code ?? "ERROR"} — ${message}`;
    console.error(`[pi-guardrails] ${SDK_LOAD_ERROR}`);
  }
}

/** True when the SDK's client entrypoints loaded and the bridge can operate. */
function sdkAvailable(): boolean {
  return Client !== null && StreamableHTTPClientTransport !== null && StdioClientTransport !== null;
}

/** Diagnosable cause when the SDK is installed but failed to load (spec 27 R27.2). */
export function getMcpSdkLoadError(): string | null {
  return SDK_LOAD_ERROR;
}

/**
 * The MCP Streamable HTTP endpoint served by radical-mcp-server.
 *
 * The server's router (`crates/server/src/router.rs`) mounts the primary
 * transport at `POST /mcp/stream`; there is no `/mcp/v1/...` prefix and the
 * legacy SSE routes are `/mcp/sse` + `/mcp/messages`. See spec 24.
 */
export const MCP_STREAM_PATH = "/mcp/stream";

/**
 * Resolve a caller-supplied MCP endpoint to the server's real transport route.
 *
 * - A bare base URL (or one ending in `/mcp`) resolves to `MCP_STREAM_PATH`.
 * - A legacy SSE path (e.g. `/mcp/v1/sse`, `/sse`) resolves to
 *   `MCP_STREAM_PATH`, because the server does not serve `/mcp/v1/sse`.
 * - Any other explicit path is preserved, so a caller can target a route the
 *   server does serve (e.g. an already-correct `/mcp/stream`).
 */
export function resolveMcpEndpoint(url: string): URL {
  const parsed = new URL(url);
  const path = parsed.pathname.replace(/\/+$/, "");
  if (path === "" || path === "/" || path === "/mcp" || path.endsWith("/sse")) {
    parsed.pathname = MCP_STREAM_PATH;
  }
  return parsed;
}

export class MCPClient {
  private client: InstanceType<MCPClientCtor> | null = null;
  private transport: unknown = null;
  private connected = false;
  private tools: string[] = [];
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private endpoint: string | null = null;

  async tryConnect(endpoint: string): Promise<boolean> {
    if (!sdkAvailable()) return false;

    this.endpoint = endpoint;

    try {
      // Determine transport type from endpoint
      // If endpoint looks like a URL (http/https), use the server's
      // Streamable HTTP transport (POST /mcp/stream).
      // If it's a file path or command, use stdio transport
      if (endpoint.startsWith("http://") || endpoint.startsWith("https://")) {
        return await this.connectHttp(endpoint);
      } else {
        return await this.connectStdio(endpoint);
      }
    } catch {
      this.connected = false;
      this.scheduleReconnect();
      return false;
    }
  }

  private async connectHttp(url: string): Promise<boolean> {
    if (!sdkAvailable()) return false;

    const Transport = StreamableHTTPClientTransport!;
    const endpoint = resolveMcpEndpoint(url);

    const headers: Record<string, string> = {};
    const apiKey = process.env.PI_GUARDRAILS_MCP_API_KEY;
    if (apiKey) {
      headers["Authorization"] = `Bearer ${apiKey}`;
    }

    this.transport = new Transport(endpoint, {
      requestInit: { headers },
    });

    return await this.performConnection();
  }

  private async connectStdio(command: string): Promise<boolean> {
    if (!sdkAvailable()) return false;

    const Transport = StdioClientTransport!;
    const parts = command.split(" ");
    this.transport = new Transport({
      command: parts[0],
      args: parts.slice(1),
      stderr: "pipe",
    });

    return await this.performConnection();
  }

  private async performConnection(): Promise<boolean> {
    if (!sdkAvailable() || !this.transport) return false;

    this.client = new Client!({ name: "pi-guardrails", version: "0.1.0" });

    await this.client.connect(this.transport as any);
    this.connected = true;
    this.reconnectAttempts = 0;

    // Discover available tools
    const result = await this.client.listTools();
    this.tools = result.tools.map((t: any) => t.name);

    return true;
  }

  async callTool(toolName: string, params: Record<string, unknown> = {}): Promise<any> {
    if (!this.connected || !this.client) {
      return {
        error: "MCP server not connected. Start it and run guardrail_init to retry.",
      };
    }

    try {
      const result = await this.client.callTool({ name: toolName, arguments: params });
      return result;
    } catch (err: any) {
      // Connection may have dropped
      this.connected = false;
      this.scheduleReconnect();
      return { error: `MCP call failed: ${err.message}` };
    }
  }

  isConnected(): boolean {
    return this.connected;
  }

  getTools(): string[] {
    return this.tools;
  }

  async close(): Promise<void> {
    try {
      if (this.transport && typeof (this.transport as any).close === "function") {
        await (this.transport as any).close();
      }
    } catch {
      // best-effort close
    }
    this.connected = false;
    this.client = null;
    this.transport = null;
  }

  private scheduleReconnect(): void {
    if (!this.endpoint || this.reconnectAttempts >= this.maxReconnectAttempts) return;

    this.reconnectAttempts++;
    const baseDelay = 1000;
    const maxDelay = 30000;
    const delay = Math.min(baseDelay * Math.pow(2, this.reconnectAttempts - 1), maxDelay);
    const jitter = Math.random() * delay * 0.1;

    setTimeout(() => {
      this.tryConnect(this.endpoint!).catch(() => {});
    }, delay + jitter);
  }
}
