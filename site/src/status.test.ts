import { normalizeVerdict, isStale } from './status'
test('anything unrecognized is UNKNOWN', () => {
  for (const v of [undefined, null, '', 'pass', 'GREEN', 1, {}]) expect(normalizeVerdict(v)).toBe('UNKNOWN')
  expect(normalizeVerdict('PASS')).toBe('PASS')
})
test('stale after 90 days', () => {
  const now = new Date('2026-10-01')
  expect(isStale('2026-09-01', now)).toBe(false)
  expect(isStale('2026-05-01', now)).toBe(true)
  expect(isStale('not-a-date', now)).toBe(true)
  expect(isStale(null, now)).toBe(false)
})
