export type Verdict = 'PASS' | 'FAIL' | 'UNKNOWN' | 'ERROR'
/** Anything that is not exactly a known verdict is UNKNOWN. Never green by default. */
export function normalizeVerdict(v: unknown): Verdict {
  return v === 'PASS' || v === 'FAIL' || v === 'ERROR' ? v : 'UNKNOWN'
}
export function isStale(testedOn: string | null | undefined, now = new Date(), days = 90): boolean {
  if (!testedOn) return false
  const t = Date.parse(testedOn)
  if (Number.isNaN(t)) return true
  return (now.getTime() - t) / 86400000 > days
}
