# TPL-32 verification

Fourteen focused tests passed at the review candidate, including committed
and clean ranges, invalid and empty ranges, divergent pull requests,
ambiguous merge bases, shallow history, and separate staged and unstaged
patches. The committed fixture invokes the same CLI wired into the workflow.

The workflow uses immutable event revisions and full checkout history.
Pull requests compare their unique merge base to their head. Pushes check
ancestry and scan from the event's before revision. Hosted confirmation is
reported on the draft PR once the update runs.
