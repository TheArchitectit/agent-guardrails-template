# Tasks

## 1. Deliver the verified scanner command

- [x] 1.1 Allocate private scratch and clean up only the invocation's own files. Exercise concurrent invocations and a pre-existing old scratch path.
- [x] 1.2 Verify the pinned official archive digest before extraction and verify the executable version before scanning. Exercise wrong bytes, wrong version, and a preinstalled scanner on PATH.
- [x] 1.3 Use the command in Secret Validation while keeping the existing triggers, full checkout depth, read-only permission, redaction, and fail-closed outcomes. Remove the licensed Action and use a job summary for the scan outcome.

## 2. Verify commit scope and local execution

- [x] 2.1 Verify pinned PR and push ranges, merge ancestry, new refs, malformed or missing revisions, and nonempty scan scope in disposable repositories.
- [x] 2.2 Run real scanner controls and scan the draft's pinned commit range on the laptop. Record the version, digest verification, exit status, and finding count without disclosure of values.
- [x] 2.3 Independently review the new workflow and local evidence. Run the focused regression tests and strict OpenSpec. Track any hosted result separately.
