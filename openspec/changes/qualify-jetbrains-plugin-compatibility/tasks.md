# Tasks

## 1. Implement and verify

- [ ] 1.1 Establish the supported IntelliJ product, release and runtime matrix from the documented plugin scope and actual source usage.
- [ ] 1.2 Align Java and Kotlin targets and the generated plugin descriptor with the supported matrix. Verify the packaged artifact's class files and compatibility metadata.
- [ ] 1.3 Run the pinned wrapper and JetBrains verification against the supported builds. Exercise the verified artifact in a local plugin-load or service fixture and confirm the canonical selection request.
- [x] 1.4 Run the pinned IntelliJ Community 2023.3.2 distribution with its bundled JBR 17.0.9 in an isolated Debian/Xvfb profile, install the packaged ZIP, and observe the actual IDE log report `Loaded custom plugins: Guardrail (1.0.0)`. See `evidence.md`; this is one-build runtime evidence, not full matrix closure.

## 2. Review and archive

- [ ] 2.1 Independently review all scenarios, record exact build and runtime evidence, run strict OpenSpec and repository gates, then archive the accepted change.

The [packaging evidence](evidence.md) verifies the installable ZIP for the pinned IntelliJ build. The installed-runtime observation now verifies actual loading for that one build; the declared IDE range, broader product matrix, and canonical selection event remain open.

The runtime evidence uses only explicit, hash-pinned JetBrains and plugin artifacts in a disposable container/profile. It does not scan or execute arbitrary host runtimes, alter host IDE profiles, or claim UI/service acceptance beyond the observed plugin load.
