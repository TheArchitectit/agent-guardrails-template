// Isolated control (NOT the bridge): same transport the bridge intends, but
// using the SDK's real subpath entrypoints, to prove the SERVER + Streamable
// HTTP path work and isolate the failure to the bridge's root import.
import { Client } from "@modelcontextprotocol/sdk/client/index.js";
import { StreamableHTTPClientTransport } from "@modelcontextprotocol/sdk/client/streamableHttp.js";

const url = new URL("http://127.0.0.1:8081/mcp/stream");
const transport = new StreamableHTTPClientTransport(url);
const client = new Client({ name: "pi-guardrails", version: "0.1.0" });
await client.connect(transport);
console.log("control initialize: OK");
const tools = await client.listTools();
console.log("control tools:", tools.tools.map((t: any) => t.name).join(", "));
const okCall = await client.callTool({ name: "read_file", arguments: { path: "Cargo.toml" } });
console.log("control call success content types:", (okCall.content as any[])?.map((c) => c.type).join(","));
try {
  await client.callTool({ name: "read_file", arguments: {} });
  console.log("control rejected call: UNEXPECTED SUCCESS");
} catch (e: any) {
  console.log("control rejected call -> error:", String(e?.message).split("\n")[0]);
}
await client.close();
console.log("control done");
