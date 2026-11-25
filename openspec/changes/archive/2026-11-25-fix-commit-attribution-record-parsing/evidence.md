# TPL-48 baseline evidence

A local Linux replay from the guardrails template draft #13 head
`109c95213a1a64518707dcfd27d3b81a2433021d` used the exact `tr` invocation
from the Guardrails Compliance attribution step. With two synthetic commit
records it returned `unrecognized option '---COMMIT_SEP---'`. The workflow
reads the command through process substitution, which allows an empty loop
and misleading success.

The replacement reads the event helper's checked `%H` output as one validated
full SHA-1 per line and retrieves each complete message with an individually
checked `git show -s --format=%B`. Attribution remains advisory; malformed
records, range errors, and unreadable commit messages fail the step. Message
matching uses here-strings so `pipefail` cannot turn a successful early grep
match on a large message into a false negative.

Focused Linux Bash Git fixtures executed the exact workflow run block and
passed (`6 passed`): multiline attribution, delimiter-like body text,
large attributed and unattributed messages, missing target, successful empty
range, unreadable message, and malformed SHA output. The uncredited commit was
reported by full SHA while attributed commits were not.

Local Python discovery inventory passed (`4 discovered suites`, all covered by
`.guardrails/python-test-floor.json`); its 14 self-tests passed. The mandatory
CI rows workflow lint passed, including all 18 self-tests. Strict OpenSpec
validation passed (`63 passed, 0 failed`). `git diff --check` also passed.

The independent laptop replay passed all six focused tests from a clean
Linux checkout on Docker Desktop. Mandatory workflow rows, Python suite
discovery, and runner isolation also passed with their controls. Hosted
check results remain tracked with draft PR #13.
