# Design

## Context

Template CI and downstream consumer builds have distinct pin responsibilities. The current template root `.devgate` gitlink is `8ccb4467c70614c37f4a533428d2aa1e09a80517`. Import commit `abc8c730e066a3377d4cac83d6605d249c6bf5b2` proposes engine pin `17b3369908a2e1f4b7356fde0486d179c81eedcd` for a distinct consumer component. The [pinned import tree](https://github.com/TheArchitectit/agent-guardrails-template/tree/abc8c730e066a3377d4cac83d6605d249c6bf5b2/guardrails-control-plane/engines) and [pin comparison](https://github.com/TheArchitectit/AIGGP-Agentic-Framework/compare/17b3369908a2e1f4b7356fde0486d179c81eedcd...8ccb4467c70614c37f4a533428d2aa1e09a80517) document branch-only context. These IDs must not be treated as duplicate declarations of one component or as a current-main defect.

## Decisions

- Keep ordinary template CI on the exact root gitlink `8ccb4467c70614c37f4a533428d2aa1e09a80517` until a reviewed update proves another compatible revision.
- Treat `17b3369908a2e1f4b7356fde0486d179c81eedcd` as the import branch's proposed immutable engine pin for that consumer scope; require its consumer build to propagate the same value through assembly, reusable bootstrap, and verification.
- Before changing pin ownership, validate the affected scripts and tests, then perform a clean authorized recursive or standalone fetch and confirm its resolved commit.
- Record fetch mode and resolved commit without recording private repository topology or credentials.
- Treat DG-08 as the existing reusable-bootstrap requirement; this item addresses pin selection and ownership only.

## Rollout

Run validation against the current root gitlink and the proposed import pin separately. Change ownership only with script/test results and clean authorized fetch evidence attached to the same proposal. Preserve ordinary template CI behavior unless that evidence justifies a reviewed update.
