import { Link } from 'react-router-dom'
import EnforcementTable from '../components/EnforcementTable'
import HostTabs from '../components/HostTabs'
import VerdictBadge from '../components/VerdictBadge'
import compat from '../data/compat.json'
import roadmap from '../data/roadmap.json'
import tools from '../data/tools.json'

const LAWS = [
  ['Read before editing', 'Never modify a file the agent has not read.'],
  ['Stay in scope', 'Only touch files inside the authorized task.'],
  ['Verify before committing', 'Compile, lint and test before a commit counts.'],
  ['Halt when uncertain', 'Ask instead of guessing.'],
]
export function Home() {
  return (<>
    <section className="hero">
      <h1>Boundaries for AI agents, with receipts.</h1>
      <p>An MCP server and editor extensions that check what your AI agent is about to do, and tell you plainly what was and was not enforced.</p>
      <p><Link className="btn" to="/install">Install in 5 minutes</Link> <Link to="/story">Read our story</Link></p>
    </section>
    <section><h2>The problem in one screen</h2>
      <p>Speed without boundaries turns small mistakes into big ones. A tool can help, but only if it is honest about what it can stop.</p></section>
    <section><h2>Advisory, checked, blocking</h2><EnforcementTable /></section>
    <section><h2>The Four Laws</h2>
      <ul className="cards">{LAWS.map(([t, d], i) => <li key={t}><h3>{i + 1}. {t}</h3><p>{d}</p></li>)}</ul></section>
    <section><h2>Works with</h2>
      <p>Hosts appear here only after a passing test record. Today: none are verified. Planned: VS Code with GitHub Copilot first, then Claude Code and Cursor.</p></section>
    <section><h2>Install</h2><HostTabs /></section>
  </>)
}
export const Story = () => (<article>
  <h1>Why we are rebooting</h1>
  <p>This is the maintainer's own account, limited to verified facts. The main branch of the original template repository was reduced to a retirement notice. The code survived because tag v3.7.1 still holds all 816 files, including the MCP server and the editor adapters. We are recovering from that tag with a new branch and no history rewrite.</p>
  <p>The lesson is the product: a boundary is only worth what you can prove it stopped. So this project labels every check as advisory, checked, or blocking, and says "Not enforced" when it cannot show evidence.</p>
</article>)
export const How = () => (<article><h1>How it works</h1>
  <p>Connecting an MCP tool to an assistant does not force the assistant to call it. Advisory checks can be skipped. Checked operations are verified by the server for actions it controls. Blocking needs a tested hook or wrapper on a named host version.</p>
  <EnforcementTable />
  <p>The server currently exposes {tools.count} tools, generated from source.</p></article>)
export const Install = () => (<article><h1>Install</h1><p>Start the server, then point your host at it.</p><HostTabs />
  <p>Bind the server to localhost only. Authentication on the MCP endpoint is not yet verified, so do not expose it to a network. Never commit keys.</p></article>)
export const Compatibility = () => (<article><h1>Compatibility</h1><p>{compat.generatedNote}</p>
  <ul>{compat.hosts.map(h => <li key={h.host}>{h.host}: <VerdictBadge verdict={h.blocking} /> blocking</li>)}</ul></article>)
export const Policies = () => (<article><h1>Policies</h1><ul>{LAWS.map(([t, d]) => <li key={t}><strong>{t}.</strong> {d}</li>)}</ul></article>)
export const Roadmap = () => (<article><h1>Roadmap</h1><ol>{roadmap.map(m => <li key={m.id}><strong>{m.id}</strong> {m.title} <em>({m.status})</em></li>)}</ol></article>)
export const Security = () => (<article><h1>Safety and security</h1>
  <p>No claim of complete safety. Threat model and reporting channel live in the repository security policy.</p></article>)
export const Community = () => (<article><h1>Community</h1><p>Contributions welcome. Start with issues labeled good first issue.</p></article>)
