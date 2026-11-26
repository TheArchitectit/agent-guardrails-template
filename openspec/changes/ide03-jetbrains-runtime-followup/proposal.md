# Proposal: Complete the IDE-03 JetBrains runtime follow-up

The merged IDE-03 selection-field change updated the JetBrains request builder,
but the plugin did not compile against the pinned IntelliJ 2023.3.2 platform.
Two source calls used unavailable APIs, and the project omitted the standard
IntelliJ instrumentation-tools dependency required by `instrumentCode`.

This follow-up ports the narrow API fixes, declares the pinned platform's
standard instrumentation tools, and records an actual compiled
`GuardrailService.validateSelection` request against the local selection
fixture. It does not upgrade the IntelliJ platform or alter runtime limits.
