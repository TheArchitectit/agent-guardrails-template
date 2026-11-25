# Classification and Mediation

## MODIFIED Requirements

### Requirement: fail-closed destructive boundaries

At each destructive or irreversible boundary claimed as guarded, suspected injection SHALL block and escalate by default before the effect occurs. Policy MAY relax a specific boundary only through the bundle with an expiring waiver. Conformance evidence SHALL identify the consumer call site and demonstrate that a denied decision prevents the effect. A caller-invoked analysis tool alone SHALL NOT establish action-boundary mediation.

#### Scenario: ambiguous classification at delete

- **GIVEN** a delete action whose triggering context includes suspected injection
- **WHEN** default policy evaluates the action
- **THEN** the action SHALL be blocked before execution and an escalation emitted

#### Scenario: analysis tool without action integration

- **GIVEN** an explicit scan tool that returns a suspected-injection verdict but has no connected action consumer
- **WHEN** conformance scope is reported
- **THEN** the claim SHALL be limited to the analysis tool and SHALL NOT state that the destructive action is intercepted
