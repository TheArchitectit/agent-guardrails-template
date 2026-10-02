// Fails if public site content contains IPs, hostnames of private networks, tokens or home paths.
import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join } from 'node:path'
const bad = [/\b(?:10|192\.168|172\.(?:1[6-9]|2\d|3[01]))\.\d+\.\d+(?:\.\d+)?\b/, /\b\d{1,3}(?:\.\d{1,3}){3}\b(?!\/)/, /ghp_[A-Za-z0-9]{20,}/, /sk-[A-Za-z0-9]{20,}/, /\/home\/\w+|\/Users\/\w+/, /\.(?:lan|local|internal)\b/]
let hits = 0
function walk(d) { for (const f of readdirSync(d)) { const p = join(d, f); if (statSync(p).isDirectory()) walk(p); else if (/\.(tsx?|json|md|css|html)$/.test(f)) {
  const t = readFileSync(p, 'utf8'); for (const r of bad) { const m = t.match(r); if (m) { hits++; console.error(`${p}: ${m[0]}`) } } } } }
walk('src'); walk('index.html'.replace('index.html', 'src'))
if (hits) { console.error(`scrub: ${hits} hit(s)`); process.exit(1) } console.log('scrub: clean')
