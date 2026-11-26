## ADDED Requirements

### Requirement: JetBrains plugin compatibility claims are supported by its artifact

The JetBrains plugin SHALL declare only supported IDE builds. Its compiled artifact and bundled code SHALL be compatible with the runtimes and APIs of the declared builds. The verified artifact SHALL preserve the canonical IDE-03 selection request.

#### Scenario: Packaged metadata matches verified support

- **WHEN** the pinned build produces an installable plugin artifact
- **THEN** its descriptor bounds and Java/Kotlin class-file levels match the recorded supported IDE and runtime matrix.

#### Scenario: The supported builds pass compatibility verification

- **WHEN** the packaged artifact is checked against the supported IntelliJ builds
- **THEN** the compatibility checks accept the artifact and the evidence identifies the tested builds, wrapper, runtime and artifact.

#### Scenario: The verified artifact sends canonical selection

- **WHEN** an adopted local fixture uses the verified plugin artifact to validate synthetic source
- **THEN** the fixture receives the original selected text in the canonical request field.
