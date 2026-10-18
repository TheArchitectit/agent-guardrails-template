// Deprecated shim — superseded by scripts/check_policy_pack_lock.py (Spec 07).
//
// This file used to be the only policy-pack verifier in the repository, and:
//
//   1. No workflow invoked it. `docs/platform-current-state.md` says so in
//      plain text: "No active workflow ... runs `validate-policy-packs.mjs`."
//   2. It passed vacuously. With zero packs and an empty LOCK.json it printed
//      "lock ok" and exited 0, because its expected-lock loop iterated over an
//      empty set.
//   3. It could not carry the lock record Spec 07 requires. R07-4.1.3 needs
//      pack name, version label, immutable revision, digest, resolver version,
//      overlay digest and resolution time. The flat { path: { version, sha256 } }
//      map it wrote has none of the latter four.
//
// The Python gate is a strict superset: it discovers packs non-vacuously,
// schema-validates each pack as inert data (R07-4.3.2), requires every rule to
// name a registered tool, verifies the lock on every identity field, and
// recomputes the effective policy digest for determinism (R07-4.1.2). It is
// wired into regression-guard.yml.
//
// Keeping a second implementation of the same lock is how the two drift apart,
// so this file now delegates. It will be deleted once nothing references the
// command; until then it preserves the muscle-memory path and cannot disagree
// with CI.
import { spawnSync } from 'node:child_process'

const relock = process.argv.includes('--relock')
const args = ['scripts/check_policy_pack_lock.py', '--repo', '.']
if (relock) args.push('--relock')

console.warn(
  'NOTE: validate-policy-packs.mjs is deprecated; running scripts/check_policy_pack_lock.py.\n' +
    '      The lock record format changed to the Spec 07 R07-4.1.3 shape.'
)

const { status } = spawnSync('python3', args, { stdio: 'inherit' })
process.exit(status === null ? 1 : status)
