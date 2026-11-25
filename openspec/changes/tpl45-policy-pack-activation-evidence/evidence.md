# Evidence and verification limits

## Source inspection

- Resolver and activation: `mcp-server/internal/guardrails/policy_pack.go` (`PolicyPackManager.ActivateFrom`, `ActivateFromVerified`).
- Signature trust: `mcp-server/internal/guardrails/policy_signature.go` (`TrustedKeySet`, signature verification).
- MCP status: `mcp-server/internal/mcp/policy_pack.go` and `policy_pack_test.go` include active digest/attribution and failed-candidate cases.
- Existing named tests include `TestActivateFrom_FailedCandidateLeavesPriorSetInForce`, `TestPolicyPackStatus_ActiveReportsDigestAndAttribution`, `TestSignatureFromUnknownKeyIsRefused`, and `TestTamperedBytesFailDespiteAValidLookingSignature`. They were source-inspected, not run for this package.

## Not run / not established

No integrated MCP server acceptance was run for TPL-45. No production pack, key, credentials, or live environment were accessed. This evidence file records code/test pointers only and does not claim activation or signature verification in a running instance.
