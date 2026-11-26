# Tasks

## 1. Prepare and validate release metadata

- [ ] 1.1 After release version and notes are approved, update `VERSION` and `CHANGELOG.md` to identify the reviewed commit; do not modify Release Coverage's tag-at-HEAD check.
- [x] 1.2 Add isolated release-gate tests in a temporary Git repository: matching annotated version tag at HEAD plus changelog entry passes; missing tag, lightweight tag, stale tag at parent, or missing changelog entry fails. Assert the test does not mutate real refs or contact a remote.

## 2. Verify the local release contract

- [x] 2.1 Run the release coverage test command and `openspec validate release-tpl35-publish-approved-main --strict`; record outcomes and confirm the mismatched-tag fixture still fails.
- [x] 2.2 Run release validation against the temporary matching fixture and a stale-tag fixture; verify no publication step is invoked.

## Workflow follow-up

- After required checks pass, the authorized release owner may create the annotated tag and publish the release for the approved commit. This external release action is not performed by the test or planning package.
