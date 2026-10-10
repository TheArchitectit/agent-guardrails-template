// Live pi->OMCP bridge compatibility run (spec 22 Pi-bridge row).
// Drives the REAL bridge module: pi-extension/mcp-bridge/mcp-client.ts.
// NOT committed: standalone evidence runner, run via `node live-bridge-run.ts`.
import { MCPClient } from "./mcp-bridge/mcp-client.ts";

const endpoint = process.env.MCP_ENDPOINT ?? "http://127.0.0.1:8081";

function hr(t: string) {
  console.log(`\n=== ${t} ===`);
}

const client = new MCPClient();
let exit = 0;

hr("connect (tryConnect)");
console.log("endpoint:", endpoint);
const ok = await client.tryConnect(endpoint);
console.log("tryConnect:", ok);
console.log("isConnected:", client.isConnected());
console.log("getTools:", JSON.stringify(client.getTools()));
if (!ok) {
  console.log("BLOCKER: bridge could not connect");
  process.exit(2);
}

hr("tools/call success: read_file {path: Cargo.toml}");
const success = await client.callTool("read_file", { path: "Cargo.toml" });
console.log(JSON.stringify(success, null, 2));

hr("tools/call rejected: read_file {} (missing required path)");
const rejected = await client.callTool("read_file", {});
console.log(JSON.stringify(rejected, null, 2));

hr("close");
await client.close();
console.log("closed; isConnected:", client.isConnected());

// Verdict logic: success must carry content; rejected must be an error the
// client parsed (never a silent success).
const successOk = !!success && !success.error && Array.isArray(success.content);
const rejectedOk = !!rejected && !!rejected.error;
console.log("\nSTEP RESULTS: success=" + successOk + " rejected=" + rejectedOk);
if (!successOk || !rejectedOk) exit = 1;

process.exit(exit);
