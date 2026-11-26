## ADDED Requirements

### Requirement: Advertised Team CLI output is executable

The Team CLI SHALL expose the supported output modes for each command. Supported JSON commands SHALL emit one parseable JSON result on stdout and preserve documented result fields. The CLI SHALL route diagnostics separately and reject an unsupported format before the operation starts.

#### Scenario: Status JSON mode

- **WHEN** a caller requests JSON status with a selected phase or with all phases
- **THEN** the installed command returns a JSON document with the matching phase and progress data.

#### Scenario: Existing output remains usable

- **WHEN** a caller requests a supported text or JSON format
- **THEN** the command returns the documented output and preserves the relevant existing fields.

#### Scenario: Unsupported output stops before effects

- **WHEN** a caller requests an unsupported mode for an operation with side effects
- **THEN** the CLI exits with an actionable diagnostic and the operation leaves team state untouched.
