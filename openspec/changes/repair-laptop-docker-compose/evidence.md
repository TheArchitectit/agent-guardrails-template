# TPL-39 verification

Daemonless Compose v2 configuration and image pin checks passed. The build
context resolves to the MCP module, both published ports use IPv4 loopback,
and startup waits for healthy PostgreSQL and Redis. The runtime uses the
local deployment profile.

The launcher downloads the approved source archive at
`ba9ff630b24821507d6c93c6c6117ac7be4e9b1e` and verifies its SHA-256.
PowerShell 7.4 parsed the launcher. Mock startup and restart used the actual
archive and daemonless Compose rendering. Restart preserved the synthetic
env file with Windows line endings, and the launcher kept generated values
out of its output. Mock checks rejected an altered archive, a nonlocal
Docker context, an ambient credential override, and a conflicting container
before startup.

The pilot operator reported that Docker Desktop is installed on the target
Windows laptop. Local Linux engine readiness, the Docker image build,
live service health, and teardown remain pending on that machine.
