# DevGate Pin Ownership

## Purpose

Keep template CI pinning separate from per-consumer immutable pin selection.
The import snapshot is template commit `abc8c730e066a3377d4cac83d6605d249c6bf5b2`; its [pinned engine tree](https://github.com/TheArchitectit/agent-guardrails-template/tree/abc8c730e066a3377d4cac83d6605d249c6bf5b2/guardrails-control-plane/engines) proposes engine pin `17b3369908a2e1f4b7356fde0486d179c81eedcd`. The [pin comparison](https://github.com/TheArchitectit/AIGGP-Agentic-Framework/compare/17b3369908a2e1f4b7356fde0486d179c81eedcd...8ccb4467c70614c37f4a533428d2aa1e09a80517) records the comparison context.

## ADDED Requirements

### Requirement: template and consumer pins have explicit ownership

Ordinary template CI SHALL use the current root `.devgate` gitlink `8ccb4467c70614c37f4a533428d2aa1e09a80517` unless a reviewed change establishes a compatible replacement. Each consumer build SHALL select one immutable pin per component and use it consistently for assembly, bootstrap, and verification. The branch-only engine pin SHALL NOT be treated as a duplicate `.devgate` declaration or as a current-main defect.

#### Scenario: ordinary template CI uses the compatible pin

- **GIVEN** ordinary template CI without a reviewed pin update
- **WHEN** the workflow resolves its DevGate dependency
- **THEN** it uses the declared compatible immutable root gitlink `8ccb4467c70614c37f4a533428d2aa1e09a80517`

#### Scenario: consumer selects a separate pin

- **GIVEN** a consumer build selecting an immutable DevGate commit
- **WHEN** assembly, bootstrap, and verification resolve the dependency
- **THEN** each uses the same selected commit and reports that identity

#### Scenario: imported engine pin remains scoped to its branch

- **GIVEN** the imported branch snapshot at template commit `abc8c730e066a3377d4cac83d6605d249c6bf5b2` with proposed engine pin `17b3369908a2e1f4b7356fde0486d179c81eedcd`
- **WHEN** the snapshot is compared with the current template root gitlink
- **THEN** the review identifies these as distinct component pins and does not report the branch-only proposal as a current-main defect

### Requirement: pin changes require validated fetch evidence

Before pin ownership or values change, affected scripts and tests SHALL be validated and a clean authorized recursive or standalone fetch SHALL confirm the resolved immutable commit. Evidence SHALL record fetch mode and resolved commit without exposing private repository topology or credentials. Reusable bootstrap behavior governed by DG-08 remains independently specified.

#### Scenario: proposed pin change validated

- **GIVEN** a proposed update to template or consumer pin ownership
- **WHEN** the change is reviewed
- **THEN** script/test results and clean fetch evidence identify the same selected commit before the pin change is accepted
