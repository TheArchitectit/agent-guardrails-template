# Design

## Record boundaries

Call the existing `scripts/changed_commits.sh --format %H` with the event environment and keep its exit status. Validate the newline-separated full commit IDs and ask Git for each complete body. Git supplies the record boundaries so a multiline body or a delimiter-like phrase cannot split a commit. Print only advisory status and the relevant commit identifier. Preserve the existing keyword list, attribution text, and advisory result for uncredited messages.

A range error, malformed record, or unreadable commit fails the job before any passing summary. State that zero commits were available when the event legitimately yields an empty range.

## Verification

Use a disposable Git repository with same-event commits containing a multiline keyword, the expected attribution in a later line, literal separator-like text, and an unattributed message. Verify the checked commit count, advisory count, and fail-closed behavior when the range helper cannot resolve the target branch. Retain the workflow's trigger, runner, and fork guard.
