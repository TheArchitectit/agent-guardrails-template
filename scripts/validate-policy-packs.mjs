// F011/F012: policy packs must reference real server tools and never claim more than "advisory" without a test record.
import { readFileSync, readdirSync, statSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'
import { createHash } from 'node:crypto'
import { existsSync } from 'node:fs'
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
walk('policy-packs')
// F016: packs are pinned by sha256 in policy-packs/LOCK.json. Changing a pack without re-locking fails.
// Re-lock deliberately with: node scripts/validate-policy-packs.mjs --relock
const packs = []
const collect = d => readdirSync(d).forEach(f => { const p = join(d, f); statSync(p).isDirectory() ? collect(p) : p.endsWith('.pack.json') && packs.push(p) })
collect('policy-packs')
const lock = {}
for (const p of packs.sort()) {
  const pk = JSON.parse(readFileSync(p, 'utf8'))
  if (!/^\d+\.\d+\.\d+$/.test(pk.version)) { console.error(`FAIL ${p}: version must be semver`); bad++ }
  lock[p] = { version: pk.version, sha256: createHash('sha256').update(readFileSync(p)).digest('hex') }
}
const lp = 'policy-packs/LOCK.json'
if (process.argv.includes('--relock')) { writeFileSync(lp, JSON.stringify(lock, null, 2) + '\n'); console.log('relocked') }
else if (!existsSync(lp)) { console.error('FAIL no LOCK.json (run --relock once)'); bad++ }
else { const cur = JSON.parse(readFileSync(lp, 'utf8')); if (JSON.stringify(cur) !== JSON.stringify(lock)) { console.error('FAIL pack contents differ from LOCK.json; bump version and --relock deliberately'); bad++ } else console.log('lock ok') }
process.exit(bad ? 1 : 0)
