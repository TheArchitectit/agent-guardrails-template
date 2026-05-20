let mcpSdk: any = null;
let sdkLoaded = false;

async function loadSDK(): Promise<any> {
  if (sdkLoaded) return mcpSdk;
  sdkLoaded = true;
  try {
    mcpSdk = await import("@modelcontextprotocol/sdk");
  } catch {
    mcpSdk = null;
  }
  return mcpSdk;
}

export class MCPClient {
  private client: any = null;
  private transport: unknown = null;
  private connected = false;
  private tools: string[] = [];
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private endpoint: string | null = null;

  async tryConnect(endpoint: string): Promise<boolean> {
    const sdk = await loadSDK();
    if (!sdk) return false;

    this.endpoint = endpoint;

    try {
      if (endpoint.startsWith("http://") || endpoint.startsWith("https://")) {
        return await this.connectSSE(endpoint);
      } else {
        return await this.connectStdio(endpoint);
      }
    } catch {
      this.connected = false;
      this.scheduleReconnect();
      return false;
    }
  }

  private async connectSSE(url: string): Promise<boolean> {
    const sdk = await loadSDK();
    if (!sdk) return false;

    const { SSEClientTransport } = sdk;
    const sseUrl = new URL(url.endsWith("/sse") ? url : `${url}/mcp/v1/sse`);

    const headers: Record<string, string> = {};
    const apiKey = process.env.PI_GUARDRAILS_MCP_API_KEY;
    if (apiKey) {
      headers["Authorization"] = `Bearer ${apiKey}`;
    }

    this.transport = new SSEClientTransport(sseUrl, {
      eventSourceInit: { headers },
      requestInit: { headers },
    });

    return await this.performConnection();
  }

  private async connectStdio(command: string): Promise<boolean> {
    const sdk = await loadSDK();
    if (!sdk) return false;

    const { StdioClientTransport } = sdk;
    const parts = command.split(" ");
    this.transport = new StdioClientTransport({
      command: parts[0],
      args: parts.slice(1),
      stderr: "pipe",
    });

    return await this.performConnection();
  }

  private async performConnection(): Promise<boolean> {
    const sdk = await loadSDK();
    if (!sdk || !this.transport) return false;

    const { Client } = sdk;
    this.client = new Client({ name: "pi-guardrails", version: "0.1.0" });

    await this.client.connect(this.transport as any);
    this.connected = true;
    this.reconnectAttempts = 0;

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
      if (this.endpoint) this.tryConnect(this.endpoint).catch(() => {});
    }, delay + jitter);
  }
}
