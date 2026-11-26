# IDE-05 JetBrains plugin compatibility scoped evidence

Base: merged IDE-03 scope `f470511c1fe6085707546654c702c9add6d65da5`.

## Current target artifact

- Source settings before the scratch target change selected IntelliJ IDEA Community 2023.3.2, declared `since-build=223` and `until-build=241.*`, used Gradle wrapper 8.6, and emitted Java/Kotlin 21 bytecode.
- Build environment: Docker image `gradle:8.5-jdk21`, OpenJDK 21.0.2; the project wrapper downloaded Gradle 8.6. The source worktree was mounted read-only and copied to `/tmp/ide04-current` before building.
- `./gradlew clean test --no-daemon --stacktrace` passed. It compiled Kotlin and ran the IntelliJ instrumentation pipeline; `test` was `NO-SOURCE`.
- Produced instrumented JAR: `/tmp/ide04-current/build/libs/guardrail-plugin-1.0.0-instrumented.jar`; retained inspection copy: `C:\Users\roger.lund\Documents\Codex\ide04-evidence\jetbrains-current.jar`; SHA-256 `521c05fbffd590d250372b233968a7bb6d9146c1828e57778108e9ae22891bf8`.
- All 19 plugin classes in the JAR had class-file major 65 (Java 21). `META-INF/plugin.xml` declared `since-build=223`, `until-build=241.*`.
- `./gradlew clean buildPlugin --no-daemon --stacktrace` failed at `:buildSearchableOptions`. The IntelliJ 2023.3.2 runtime at `.../ideaIC-2023.3.2/jbr/bin/java` rejected plugin classes with `UnsupportedClassVersionError`: plugin major 65, runtime maximum 61. This is the current target's packaging failure.

## Bounded Java 17 target experiment

- In a disposable source copy, changed Java source/target and Kotlin JVM target to 17 while retaining the JDK 21 toolchain. `clean buildPlugin` against the pinned 2023.3.2 platform passed and produced `/tmp/ide04-java17-pluginzip/build/distributions/guardrail-plugin-1.0.0.zip` (retained copy `C:\Users\roger.lund\Documents\Codex\ide04-evidence\jetbrains-java17-plugin.zip`, SHA-256 `797162829c4cf34832c4ea6cd8b5b56e2041e2282314d65cd02464ae2d078f91`). The 19 plugin classes were major 61 and the descriptor stayed at 223–241.*.
- Strengthened the scratch-only compile configuration to use Java `options.release.set(17)` and Kotlin `jvmTarget=17` plus `-Xjdk-release=17`, preventing Java 21 API references from compiling. The wrapper's JDK toolchain remained 21.0.2; the pinned platform's JBR 17 then ran `buildSearchableOptions` successfully.
- The project currently has no Java source (`compileJava NO-SOURCE`), so the Java `--release 17` setting is configured for Java compilation but not exercised by existing Java files. Kotlin compilation did run with `-Xjdk-release=17`.
- The release-level `clean buildPlugin` passed in 32 seconds. ZIP path in the disposable Linux build: `/tmp/ide05-release17/build/distributions/guardrail-plugin-1.0.0.zip`; retained copy: `C:\Users\roger.lund\Documents\Codex\ide04-evidence\jetbrains-java17-release-plugin.zip`; SHA-256 `f55b283c138f43aa4c5f4512adec40f68d3aebbee755f5e7bd23a90da8b61556`.
- ZIP descriptor path: `guardrail-plugin/lib/guardrail-plugin-1.0.0.jar!/META-INF/plugin.xml`; bounds remain `since-build=223`, `until-build=241.*`. All 19 plugin classes were major 61. Across the ZIP and nested JARs, observed class-major counts were 49: 32, 51: 217, 52: 1402, 53: 4, and 61: 19.
- IntelliJ build emitted a remaining warning that `since-build=223` is below the pinned 2023.3.2 platform build 233; the compatibility floor and upper bound have not been changed.

## Canonical selection check and limits

`node tests/regression/test_ide_selection_payload_regression_FAIL-e912933d.js` passed under Node 22 and confirmed the JetBrains builder still sends `selection` with the source text. This is the existing source-builder regression; it does not execute the installed plugin in a full IDE runtime. The IDE03 instrumented service evidence remains separate.

The new runnable regression entry is `tests/regression/test_jetbrains_bytecode_compatibility_regression_FAIL-3b71d9a2.sh`. It runs the pinned wrapper's `clean buildPlugin` in a disposable project copy and checks the packaged class major and presence of descriptor bounds. It passed using the verified Java 17 release configuration.

This evidence qualifies only the pinned 2023.3.2 platform build and bytecode floor. The supported IntelliJ product/build matrix, descriptor narrowing, verification across other IDE builds, and installed-plugin/runtime acceptance remain open. No publisher or marketplace action was performed.
