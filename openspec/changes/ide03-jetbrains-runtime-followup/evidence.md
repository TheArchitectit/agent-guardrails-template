# IDE-03 JetBrains runtime evidence

Base: merged `origin/main` `7c52f2091b9b0fa508500daa1d93acd3f091a396`.

- `./gradlew test --no-daemon --stacktrace` in Docker `gradle:8.5-jdk21`:
  passed. The run executed `compileKotlin`, `instrumentCode`, `instrumentTestCode`,
  `instrumentedJar`, `composedJar`, `prepareTest`, and `test` (14 actionable
  tasks; `test` had no test sources).
- The prior compile errors are resolved by `PsiDocumentManager` document lookup
  and PSI-backed language lookup.
- A compiled Java harness instantiated `GuardrailService` and called
  `validateSelection("jetbrains selection payload", "kotlin")` while a local
  capture server listened on port 8081. The observed request was:

  ```json
  {"path":"/ide/validate/selection","body":{"selection":"jetbrains selection payload","language":"kotlin"}}
  ```

- The probe and normal build ran in disposable Linux containers; no credentials
  or external consumer systems were used.
