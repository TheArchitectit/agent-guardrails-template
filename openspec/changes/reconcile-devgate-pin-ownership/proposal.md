# Proposal: Reconcile DevGate pin ownership

## Why

The ordinary template CI contract needs a clear compatible DevGate pin while an AIGGP import branch carries a separate engine pin. The current template root's `.devgate` gitlink is `8ccb4467c70614c37f4a533428d2aa1e09a80517`; import commit `abc8c730e066a3377d4cac83d6605d249c6bf5b2` records proposed engine pin `17b3369908a2e1f4b7356fde0486d179c81eedcd`. These are separate scopes and do not establish a current-main defect. See the [pinned import tree](https://github.com/TheArchitectit/agent-guardrails-template/tree/abc8c730e066a3377d4cac83d6605d249c6bf5b2/guardrails-control-plane/engines) and [pin comparison](https://github.com/TheArchitectit/AIGGP-Agentic-Framework/compare/17b3369908a2e1f4b7356fde0486d179c81eedcd...8ccb4467c70614c37f4a533428d2aa1e09a80517).

## What Changes

- Define ordinary template CI pin ownership around the current `.devgate` root gitlink `8ccb4467c70614c37f4a533428d2aa1e09a80517`.
- Keep the import branch's proposed engine pin `17b3369908a2e1f4b7356fde0486d179c81eedcd` distinct and require each AIGGP consumer build to select one immutable pin consistently.
- Require script/test validation and a clean authorized recursive or standalone fetch before changing ownership or pin values.
- Keep private repository topology and details out of the specification.

## Capabilities

### New Capabilities

- `devgate-pin-ownership`: defines how template CI and consumer builds select immutable DevGate revisions.

### Modified Capabilities

None.

## Impact

Template CI configuration, consumer bootstrap scripts, pin checks, and fetch validation. DG-08 remains the reusable bootstrap contract; this proposal does not revise its scope.
