let MCP_SDK: typeof import("@modelcontextprotocol/sdk") | null = null;
try {
  MCP_SDK = await import("@modelcontextprotocol/sdk");
} catch {
  // @modelcontextprotocol/sdk is an optional dependency
  // MCP bridge is permanently unavailable when not installed
}

export class MCPClient {
  private client: InstanceType<typeof MCP_SDK!["Client"]> | null = null;
  private transport: unknown = null;
  private connected = false;
  private tools: string[] = [];
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private endpoint: string | null = null;

  async tryConnect(endpoint: string): Promise<boolean> {
    if (!MCP_SDK) return false;

    this.endpoint = endpoint;

    try {
      // Determine transport type from endpoint
      // If endpoint looks like a URL (http/https), use SSE transport
      // If it's a file path or command, use stdio transport
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
    if (!MCP_SDK) return false;

    const { SSEClientTransport } = await MCP_SDK;
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
    if (!MCP_SDK) return false;

    const { StdioClientTransport } = await MCP_SDK;
    const parts = command.split(" ");
    this.transport = new StdioClientTransport({
      command: parts[0],
      args: parts.slice(1),
      stderr: "pipe",
    });

    return await this.performConnection();
  }

  private async performConnection(): Promise<boolean> {
    if (!MCP_SDK || !this.transport) return false;

    const { Client } = await MCP_SDK;
    this.client = new Client({ name: "pi-guardrails", version: "0.1.0" });

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
