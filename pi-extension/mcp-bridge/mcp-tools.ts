import { Type } from "@sinclair/typebox";
import type { MCPClient } from "./mcp-client.js";

export function registerMCPBridgeTool(pi: any, mcpClient: MCPClient): void {
  pi.registerTool({
    name: "guardrail_mcp",
    label: "Guardrails MCP Bridge",
    description:
      "Proxy to the Guardrail MCP server. Requires the Go server to be running. Use action parameter to specify the MCP tool name.",
    promptSnippet: "Access MCP guardrail server tools",
    parameters: Type.Object({
      action: Type.String({ description: "MCP tool name to call (e.g. validate_bash)" }),
      params: Type.Optional(
        Type.Record(Type.String(), Type.Unknown(), { description: "Parameters for the MCP tool" }),
      ),
    }),
    async execute(_id: string, args: any) {
      if (!mcpClient.isConnected()) {
        return {
          error: "MCP server not connected. Start it and run guardrail_init with MCP endpoint configured to connect.",
        };
      }
      return mcpClient.callTool(args.action, args.params || {});
    },
  });
}
