## ADDED Requirements

### Requirement: JetBrains selection validation compiles and sends the canonical field

The JetBrains plugin SHALL compile against the pinned IntelliJ platform and its
selection-validation service SHALL send the selected source under the canonical
`selection` JSON field to `/ide/validate/selection`.

#### Scenario: compiled selection request reaches the fixture

- **GIVEN** the plugin is built with the pinned IntelliJ platform and standard
  instrumentation tools
- **WHEN** the compiled `GuardrailService` validates Kotlin source selection
- **THEN** the fixture receives HTTP POST JSON containing the exact source under
  `selection` and the language identifier under `language`
