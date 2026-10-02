import { Verdict, normalizeVerdict } from '../status'
const ICON: Record<Verdict, string> = { PASS: '\u2713', FAIL: '\u2717', UNKNOWN: '?', ERROR: '!' }
export default function VerdictBadge({ verdict }: { verdict: unknown }) {
  const v = normalizeVerdict(verdict)
  return (
    <span className={`badge badge-${v.toLowerCase()}`} data-verdict={v}>
      <span aria-hidden="true">{ICON[v]}</span> {v}
    </span>
  )
}
