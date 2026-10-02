import { useState } from 'react'
import { isStale } from '../status'
import compat from '../data/compat.json'

const SNIPPETS: Record<string, string> = {
  'VS Code + GitHub Copilot': '{\n  "servers": {\n    "guardrails": { "type": "http", "url": "http://localhost:8080/mcp" }\n  }\n}',
  'Claude Code': 'claude mcp add --transport http guardrails http://localhost:8080/mcp',
  'Cursor': '{\n  "mcpServers": {\n    "guardrails": { "url": "http://localhost:8080/mcp" }\n  }\n}',
}
export default function HostTabs() {
  const hosts = compat.hosts
  const [i, setI] = useState(0)
  const h = hosts[i]
  const stale = isStale(h.testedOn)
  return (
    <div className="tabs">
      <div role="tablist" aria-label="Host">
        {hosts.map((x, n) => (
          <button key={x.host} role="tab" id={`tab-${n}`} aria-selected={n === i} aria-controls="tabpanel" onClick={() => setI(n)}>{x.host}</button>
        ))}
      </div>
      <div role="tabpanel" id="tabpanel" aria-labelledby={`tab-${i}`}>
        <pre><code>{SNIPPETS[h.host]}</code></pre>
        <button onClick={() => navigator.clipboard?.writeText(SNIPPETS[h.host])}>Copy</button>
        <p className={stale ? 'stale' : ''}>
          {h.testedVersion ? `Tested with ${h.testedVersion} on ${h.testedOn}${stale ? ' (stale: older than 90 days)' : ''}` : 'Not yet tested. Treat as unverified.'}
        </p>
      </div>
    </div>
  )
}
