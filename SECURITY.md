# Security policy

## Reporting
Do not open a public issue for a vulnerability. Use GitHub private vulnerability reporting on this repository (Security tab, "Report a vulnerability").
OWNER ACTION NEEDED: confirm private reporting is enabled and add a monitored contact address here before the repo is promoted. No address is listed because none has been chosen.

## Response targets (goals, not guarantees)
Acknowledge within 3 business days. Triage within 10. Fix timing depends on severity; we will say so plainly.

## Scope
The MCP server (`mcp-server/`), editor adapters, install scripts and the site. Out of scope: social engineering of maintainers, third-party hosts we do not control.

## Honest-claims policy
We do not claim protection we have not shown. Each check is labeled advisory, checked, or blocking, and a missing test record reads "Not enforced". If you find a claim that overstates what the code does, that is a security report.

## Known operating notes
- `/mcp` requires `Authorization: Bearer <MCP_API_KEY>`. Keep the key out of source control and logs.
- Run the server on a trusted host or behind your own network controls; do not expose it publicly.
