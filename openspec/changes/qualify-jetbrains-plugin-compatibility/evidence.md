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

This evidence originally qualified only the pinned 2023.3.2 platform build and bytecode floor. The installed-plugin observation below now adds one real IDE/JBR load check; the supported IntelliJ product/build matrix, descriptor narrowing, and selection request delivery remain open. No publisher or marketplace action was performed.

## Pinned installed-plugin runtime observation — 2026-11-27

- Official JetBrains distribution: `ideaIC-2023.3.2.tar.gz`, SHA-256 `d252110141046388e728532c5e7a312a6d40d6b75dabb493e88c0e2b8a914574`, downloaded from the JetBrains CDN and verified against its published `.sha256` file.
- Trusted plugin artifact: `jetbrains-java17-release-plugin.zip`, SHA-256 `f55b283c138f43aa4c5f4512adec40f68d3aebbee755f5e7bd23a90da8b61556`. The artifact was copied into the disposable container as an explicit input; no artifact discovery or host plugin scan was performed.
- The distribution was extracted into a disposable Docker volume and run under Debian Bookworm with an isolated `idea.config.path`, `idea.system.path`, and Xvfb display. No host IDE profile or project was used.
- Reproduction command: from a directory containing the two hash-pinned archives, run `docker run --rm -v ide05-idea-runtime:/opt/idea -v "$PWD:/evidence:ro" debian:bookworm-slim bash -lc 'set -eu; apt-get update -qq; apt-get install -y -qq tar gzip unzip xvfb libxtst6 libxi6 libxrender1 libxext6 libx11-xcb1 libfontconfig1; test -x /opt/idea/idea-IC-233.13135.103/bin/idea.sh || tar -xzf /evidence/ideaIC-2023.3.2.tar.gz -C /opt/idea; mkdir -p /tmp/config/plugins /tmp/system /tmp/log; unzip -q /evidence/jetbrains-java17-release-plugin.zip -d /tmp/config/plugins; timeout 90s xvfb-run -a /opt/idea/idea-IC-233.13135.103/bin/idea.sh dontReopenProjects -Didea.config.path=/tmp/config -Didea.system.path=/tmp/system -Didea.log.path=/tmp/log'`. The only executed IDE binary was the verified distribution's launcher.
- The actual launcher reached `IDE STARTED` for IntelliJ IDEA Community build `IC-233.13135.103` (2023.3.2). The IDE log identifies the bundled runtime as JetBrains JRE `17.0.9+7-b1087.9`, amd64.
- The packaged `jetbrains-java17-release-plugin.zip` was installed into the temporary profile's custom plugin directory. The IDE log reports `Loaded custom plugins: Guardrail (1.0.0)`, proving that the packaged plugin was accepted and loaded by the pinned IDE/JBR combination.
- The process was stopped after 90 seconds (`timeout` exit 124) after startup and plugin loading. No source project was opened and no service request was sent; canonical selection behavior therefore remains untested at this runtime boundary.
- Initial startup without `libXtst6` failed before plugin loading with `UnsatisfiedLinkError`; installing the missing GUI runtime dependency in the disposable Debian runner resolved that environmental failure. The final run had only non-fatal warnings about optional native desktop integrations and indexing.

This is installed-plugin/runtime evidence for one exact IntelliJ Community build and bundled JBR, not a complete product matrix. Other IntelliJ products/builds, Windows-native launcher behavior, plugin UI interaction, and canonical selection request delivery remain open.

The decisive log checks were `IDE STARTED`, `IDE: IntelliJ IDEA (build #IC-233.13135.103`, `JRE: 17.0.9+7-b1087.9`, and `Loaded custom plugins: Guardrail (1.0.0)`. The final run therefore establishes actual plugin loading in the pinned runtime, while the timeout and absence of an opened project are retained as explicit limits.

## Canonical selection boundary attempt

The packaged artifact does register `Guardrail.ValidateSelection` in `META-INF/plugin.xml` to `com.guardrail.plugin.actions.ValidateSelectionAction`, and the compiled action calls `GuardrailService.validateSelection(selectedText, language)`. The service builds the canonical `selection` JSON field. The actual IDE launch above proved plugin loading, but no ActionManager/editor fixture or loopback capture was run in this bounded session; therefore the end-to-end selection event remains open rather than being inferred from source registration. A future fixture should open a synthetic editor, select deterministic text, invoke `Guardrail.ValidateSelection` through ActionManager, capture the loopback JSON, and dismiss the resulting validation dialog.

## Installed selection action on UCS03

At commit `681bedeb84b8535ab39eb7c82f916a7467739210`, the lab verified the
retained plugin ZIP against SHA-256
`f55b283c138f43aa4c5f4512adec40f68d3aebbee755f5e7bd23a90da8b61556`.
The official IntelliJ Community 2023.3.2 archive matched JetBrains' published
`d252110141046388e728532c5e7a312a6d40d6b75dabb493e88c0e2b8a914574`.
IntelliJ build `IC-233.13135.103` with bundled JBR `17.0.9+7-b1087.9`
reported `Loaded custom plugins: Guardrail (1.0.0)` in an isolated Xvfb profile.

With the synthetic selection highlighted in a Kotlin editor, Find Action
invoked the installed `Guardrail.ValidateSelection` action. The private
loopback fixture recorded a POST to `/ide/validate/selection` with
`application/json; charset=utf-8` and this body.

```json
{"selection":"IDE05_SELECTION_SENTINEL_7f31","language":"kotlin"}
```

This receipt covers one product, build and runtime. Cross-build and
cross-product verification remain open.
