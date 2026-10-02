// F011/F012: policy packs must reference real server tools and never claim more than "advisory" without a test record.
import { readFileSync, readdirSync, statSync } from 'node:fs'
import { join } from 'node:path'
const tools = new Set(JSON.parse(readFileSync('site/src/data/tools.json', 'utf8')).tools)
let bad = 0
const walk = d => readdirSync(d).forEach(f => { const p = join(d, f); statSync(p).isDirectory() ? walk(p) : p.endsWith('.pack.json') && check(p) })
function check(p) {
  const pk = JSON.parse(readFileSync(p, 'utf8'))
  for (const r of pk.rules) {
    if (!tools.has(r.tool)) { console.error(`FAIL ${p} ${r.id}: unknown tool ${r.tool}`); bad++ }
    if (!['advisory', 'checked', 'blocking'].includes(r.enforcement)) { console.error(`FAIL ${p} ${r.id}: bad enforcement`); bad++ }
    if (r.enforcement !== 'advisory' && !r.evidence) { console.error(`FAIL ${p} ${r.id}: ${r.enforcement} needs evidence`); bad++ }
    if (r.on_unknown !== 'stop') { console.error(`FAIL ${p} ${r.id}: unknown must stop`); bad++ }
  }
  console.log(`checked ${p}: ${pk.rules.length} rules`)
}
walk('policy-packs'); process.exit(bad ? 1 : 0)
