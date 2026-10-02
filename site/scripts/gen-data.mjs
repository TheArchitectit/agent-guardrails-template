// Generates src/data/*.json from repo data so claims are not typed by hand.
// Tool manifest: parsed from the MCP server source (Name: "guardrail_*").
import { readdirSync, readFileSync, writeFileSync, mkdirSync } from 'node:fs'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'
const here = dirname(fileURLToPath(import.meta.url))
const mcp = join(here, '../../mcp-server/internal/mcp')
const names = new Set()
for (const f of readdirSync(mcp)) {
  if (!f.endsWith('.go') || f.endsWith('_test.go')) continue
  for (const m of readFileSync(join(mcp, f), 'utf8').matchAll(/Name:\s+"(guardrail_[a-z_]+)"/g)) names.add(m[1])
}
const out = join(here, '../src/data')
mkdirSync(out, { recursive: true })
const tools = [...names].sort()
writeFileSync(join(out, 'tools.json'), JSON.stringify({ count: tools.length, tools, source: 'mcp-server/internal/mcp' }, null, 2) + '\n')
console.log(`tools: ${tools.length}`)
