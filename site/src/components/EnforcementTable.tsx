import compat from '../data/compat.json'
import { normalizeVerdict } from '../status'
import VerdictBadge from './VerdictBadge'

const ROWS = [
  { key: 'advisory', name: 'Advisory', text: 'Your agent can ask. It can also skip.' },
  { key: 'checked', name: 'Checked operation', text: 'The server verifies an operation it controls. A permit is tied to that exact action and expires.' },
  { key: 'blocking', name: 'Blocking', text: 'Only where a tested hook or wrapper intercepts and denies. Shown per host and version.' },
] as const

type Row = Record<string, unknown>
export default function EnforcementTable({ hosts = compat.hosts as Row[] }: { hosts?: Row[] }) {
  return (
    <table className="enforcement">
      <caption>What each level means, and what has been proven per host</caption>
      <thead>
        <tr><th scope="col">Level</th><th scope="col">What it means</th>
          {hosts.map(h => <th scope="col" key={String(h.host)}>{String(h.host)}</th>)}</tr>
      </thead>
      <tbody>
        {ROWS.map(r => (
          <tr key={r.key}>
            <th scope="row">{r.name}</th>
            <td>{r.text}</td>
            {hosts.map(h => {
              const v = normalizeVerdict(h[r.key])
              return <td key={String(h.host)}>
                {v === 'PASS' ? <VerdictBadge verdict="PASS" /> :
                 v === 'UNKNOWN' ? <span className="badge badge-unknown" data-verdict="UNKNOWN">? Unknown</span> :
                 <span className="badge badge-fail" data-verdict={v}>Not enforced</span>}
              </td>
            })}
          </tr>
        ))}
      </tbody>
    </table>
  )
}
