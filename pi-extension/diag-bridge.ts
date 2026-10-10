// Diagnosis only (not the bridge run): prove the REAL bridge module cannot
// resolve its SDK root import, and that the SDK's actual entrypoints work.
import { MCPClient } from "./mcp-bridge/mcp-client.ts";

console.log("--- REAL bridge module import ---");
const client = new MCPClient();
console.log("typeof client.tryConnect:", typeof client.tryConnect);
// Directly attempt the bridge's own dependency import, as it does at module top.
try {
  await import("@modelcontextprotocol/sdk");
  console.log("root import: OK");
} catch (e: any) {
  console.log("root import FAILED:", e?.code, "-", String(e?.message).split("\n")[0]);
}
// The bridge's own connect path:
const ok = await client.tryConnect("http://127.0.0.1:8081");
console.log("bridge tryConnect ->", ok, "| isConnected ->", client.isConnected(), "| tools ->", JSON.stringify(client.getTools()));
