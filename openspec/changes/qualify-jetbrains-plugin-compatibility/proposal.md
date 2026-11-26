# Qualify JetBrains plugin compatibility

## Why

The IDE-03 follow-up restored Kotlin compilation and IntelliJ instrumentation against IntelliJ IDEA 2023.3.2. The current plugin advertises build branches 223 through 241 and compiles Java and Kotlin to the Java 21 target. JetBrains lists Java 17 as the runtime for these branches. The independent Gradle build emitted compatibility warnings while completing without test sources. The installable build then failed in searchable-options generation when the bundled runtime rejected Java 21 class files. [JetBrains build number ranges](https://plugins.jetbrains.com/docs/intellij/build-number-ranges.html)

## What changes

Establish the IntelliJ builds the packaged plugin supports. Align generated plugin metadata and emitted bytecode with the verified runtime floor and exercise the production plugin artifact against the declared range.

## Capabilities

Add `jetbrains-plugin-compatibility`.

## Impact

The change covers the JetBrains Gradle build configuration, artifact verification, supported IDE version documentation, and a local plugin-load or equivalent acceptance fixture.
