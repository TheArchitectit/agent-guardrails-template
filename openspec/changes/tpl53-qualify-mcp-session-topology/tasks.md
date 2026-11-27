# Tasks

## 1. Implement and verify

- [x] 1.1 Inventory session state, authentication, routing and resource ownership from the supported entry points. Define the observed modes and their bounds; see the source-pinned inventory in `evidence/2026-11-25-session-topology.md`.
- [x] 1.2 Make render and preflight reject a profile whose replica or routing configuration is inconsistent with its session mode.
- [ ] 1.3 Run an isolated deployment drill for restart, reinitialization, revocation, identity separation and mode-specific traffic routing. Attach image and profile digests.

## 2. Review and archive

- [ ] 2.1 Independently review evidence for all three scenarios, including expired, revoked and mismatched-principal sessions; run pinned strict OpenSpec and relevant repository gates, then archive only after scenario 3 identity binding is evidenced.

The [session source inventory](evidence/2026-11-25-session-topology.md) records the one-process control and the deployment evidence still needed.
