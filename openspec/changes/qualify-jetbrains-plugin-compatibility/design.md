# Qualify JetBrains plugin compatibility

## Current evidence

The exact merged IDE-03 follow-up source is `5ab8ce605fb1fd025d67910ce4564977b927d47b`. An independent run used the project's Gradle 8.6 wrapper and JDK 21.0.2. Kotlin compilation and IntelliJ instrumentation passed, and the test task reported no test sources. The build listed warnings about the declared IntelliJ floor and Java/Kotlin targets. The separate pinned `clean buildPlugin` run failed in searchable-options generation. The bundled Java 17 runtime rejected major-version 65 plugin classes and reported a maximum of 61. The Gradle file selects IntelliJ IDEA 2023.3.2, advertises `223` through `241.*`, and sets the Java and Kotlin targets to 21. [JetBrains build number ranges](https://plugins.jetbrains.com/docs/intellij/build-number-ranges.html)

## Decisions

Record an explicit support matrix for the enabled product and IDE builds. Choose the compiler target from the oldest supported runtime and derive plugin descriptor bounds from the builds exercised against the artifact. Check Java and Kotlin bytecode and run the JetBrains verifier against the support matrix. Preserve the IDE-03 selection contract and capture a local plugin load or service-level result using the verified artifact and runtime.

## Verification

Build and instrument the project through its pinned wrapper. Inspect metadata and the class-file version in the built artifact. Exercise the declared support boundaries and record verifier findings with their dispositions. Verify selection sends the canonical field using synthetic source and record the runtime deployment boundary.
