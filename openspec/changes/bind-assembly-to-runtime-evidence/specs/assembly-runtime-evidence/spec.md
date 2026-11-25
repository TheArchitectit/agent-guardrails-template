# Assembly Runtime Evidence

## Purpose

Bind selected assembly inputs to the runtime checks that exercise them.

## ADDED Requirements

### Requirement: exact assembly-to-runtime binding

Every runtime conformance claim for an assembled MCP deployment SHALL identify the immutable assembly selection, selected MCP build digest, active policy digest, and runtime profile. Runtime evidence SHALL report the build and policy identities observed by the checks. A passing binding SHALL require exact equality between selected and observed identities; missing or mismatching values SHALL result in incomplete or failed evidence and SHALL NOT be reported as a pass.

#### Scenario: selected inputs match runtime observation

- **GIVEN** an assembly selecting an MCP build and policy digest
- **WHEN** runtime checks execute and observe those same identities
- **THEN** the evidence binds the checks to that assembly and records the exact identities and outcomes

#### Scenario: runtime observes a different build or policy

- **GIVEN** an assembly selecting an MCP build or policy digest that differs from the runtime observation
- **WHEN** the evidence binding is evaluated
- **THEN** the binding is rejected as incomplete or failed and no passing conformance claim is emitted

### Requirement: bounded binding fixtures

Assembly-to-runtime binding SHALL be verified with an authorized matching fixture and a controlled mismatch fixture. Evidence SHALL identify the fixture scope and SHALL NOT imply acceptance by an external consumer or a deployed transport path.

#### Scenario: local binding fixtures complete

- **GIVEN** authorized matching and mismatch fixtures for the selected assembly
- **WHEN** the binding checks run
- **THEN** the matching fixture passes, the mismatch fixture is rejected, and the report limits its claim to the exercised runtime evidence path
