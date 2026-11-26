# Tasks

## 1. Implement and verify

- [ ] 1.1 Establish the supported IntelliJ product, release and runtime matrix from the documented plugin scope and actual source usage.
- [ ] 1.2 Align Java and Kotlin targets and the generated plugin descriptor with the supported matrix. Verify the packaged artifact's class files and compatibility metadata.
- [ ] 1.3 Run the pinned wrapper and JetBrains verification against the supported builds. Exercise the verified artifact in a local plugin-load or service fixture and confirm the canonical selection request.

## 2. Review and archive

- [ ] 2.1 Independently review all scenarios, record exact build and runtime evidence, run strict OpenSpec and repository gates, then archive the accepted change.

The [packaging evidence](evidence.md) verifies the installable ZIP for the pinned IntelliJ build. The declared IDE range and installed runtime acceptance remain open.
