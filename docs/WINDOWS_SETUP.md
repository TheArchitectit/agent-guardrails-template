# Windows Setup Guide

> Get Agent Guardrails running on Windows 10/11 — for teams building with AI at full velocity.

---

## Prerequisites

You need **Python 3.10+**, **Go 1.23+**, and **Docker Desktop** installed.

### Option 1: Install via winget (Recommended)

Open PowerShell or Command Prompt and run:

```powershell
winget install Python.Python.3.12 --accept-package-agreements --accept-source-agreements
winget install GoLang.Go --accept-package-agreements --accept-source-agreements
winget install Docker.DockerDesktop --accept-package-agreements --accept-source-agreements
```

After installation, **restart your terminal** and verify:

```powershell
python --version   # Expected: Python 3.12.x or higher
go version         # Expected: go1.26.x or higher
docker --version   # Expected: Docker 29.x or higher
```

### Option 2: Manual Download

- **Python:** https://python.org/downloads (choose "Add to PATH" during install)
- **Go:** https://go.dev/dl/ (Windows MSI installer)

### Option 3: Microsoft Store Python

> **Note:** The Microsoft Store Python stub (`python.exe` in `WindowsApps`) will prompt you to install Python from the Store. If you see this prompt, either install from the Store or use the winget/manual methods above.

---

## Clone the Repository

```powershell
git clone https://github.com/TheArchitectit/agent-guardrails-template.git
cd agent-guardrails-template
```

---

## Run the Setup Script

The setup script is fully Windows-compatible and will create PowerShell hooks (`.ps1`) instead of Unix shell scripts (`.sh`).

```powershell
python scripts/setup_agents.py --claude --opencode --full
```

**What gets created:**

- `.claude/skills/` — Claude Code skill definitions
- `.claude/hooks/*.ps1` — Windows PowerShell hooks
- `.opencode/` — OpenCode configuration and agents

---

## Run Tests

### Python End-to-End Tests (Recommended)

These validate the full team-management workflow:

```powershell
python -m unittest scripts.e2e_tests
```

**Expected:** 18 tests, all passing.

### Regression Check

```powershell
python scripts/regression_check.py --all
```

**Expected:** `[OK] No potential regressions detected`.

### Team Manager Unit Tests

```powershell
python -m unittest scripts.test_team_manager
```

**Status:** This test suite has a handful of pre-existing failures (8 failures, 2 errors at last count) related to authorization-level expectations and structured-logging mocks. These are **not Windows-specific** and also occur on Linux. The core functionality is validated by the E2E tests above.

---

## Build Go Components

The MCP server and team CLI are written in Go and build cleanly on Windows.

### MCP Server

```powershell
cd mcp-server
go build ./...
```

### Team CLI

```powershell
cd cmd/team-cli
go build .
```

### Running Go Tests

Unit tests (no database required):

```powershell
cd mcp-server
go test ./internal/models ./internal/security ./internal/validation
```

**Note:** Full integration tests require PostgreSQL and Redis. See `mcp-server/DEPLOYMENT_GUIDE.md` for Docker/Podman setup instructions.

---

## MCP Server (Docker)

The MCP server can be started with Docker Compose, which brings up PostgreSQL, Redis, and the server itself.

### Start the Stack

```powershell
cd mcp-server
docker compose up --build
```

**Services:**
- `guardrail-mcp-server` — MCP SSE + JSON-RPC on `:8080`, Web UI on `:8081`
- `guardrail-postgres` — PostgreSQL on `:5432`
- `guardrail-redis` — Redis on `:6379`

### Environment Setup

Copy the example environment file and fill in your values:

```powershell
cd mcp-server
copy .env.example .env
```

Required variables:
- `MCP_API_KEY` — 32+ characters with mixed case and digits
- `JWT_SECRET` — at least 32 bytes
- `DATABASE_URL` — PostgreSQL connection string

### Vision Pipeline

If you are using the vision pipeline (3D screenshot review), configure the endpoint first:

```powershell
python scripts/setup_vision_wizard.py
```

Then set the environment variable before starting Docker:

```powershell
$env:VISION_ENABLED = "true"
docker compose up
```

---

## Web Dashboard

The dashboard is a static HTML file. Open it directly in your browser:

```powershell
start web/index.html
# or
start mcp-server/web/index.html
```

---

## Windows-Specific Fixes Applied

If you're maintaining the codebase, here are the Windows compatibility changes already in place:

| Issue | Fix |
|-------|-----|
| `fcntl` not available on Windows | Fallback using `msvcrt.locking` in `scripts/team_manager.py` |
| `os.chmod` on hooks fails silently | Wrapped in `try/except` in `scripts/setup_agents.py` |
| Hooks created as `.sh` (Unix-only) | Script auto-detects Windows and creates `.ps1` PowerShell hooks |
| Unicode characters (`✓`, `⚠️`, `❌`, `✅`) break Windows console | Replaced with ASCII equivalents (`[OK]`, `[WARN]`, `[ERR]`) |
| Absolute vs. relative `Path` in tests | Updated `test_team_manager.py` assertion to check path name instead of exact equality |

---

## Troubleshooting

### "python" is not recognized

If `python --version` fails after winget install, restart your terminal. If it still fails, check that Python is on your PATH:

```powershell
$env:Path = [Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + [Environment]::GetEnvironmentVariable("Path", "User")
```

### "go" is not recognized

Same as above — restart your terminal after installing Go.

### `UnicodeEncodeError: 'charmap' codec can't encode character`

If you see this when running tests or scripts, it means the codebase still contains Unicode emoji characters in print statements. Run Python with UTF-8 mode:

```powershell
$env:PYTHONIOENCODING = "utf-8"
python scripts/setup_agents.py --claude --full
```

### Permission denied on `.teams/` directory

The team manager creates a `.teams/` folder in the repo root. If you get permission errors, ensure your terminal has write access to the repository directory.

### MCP Server won't start (missing PostgreSQL)

The MCP server requires PostgreSQL and Redis for full operation. For local development without a database:

1. Copy `mcp-server/.env.example` to `mcp-server/.env`
2. Set `DATABASE_URL` to a local PostgreSQL instance, or use the SQLite fallback if available
3. Or skip the server and use the Python scripts directly

---

## Next Steps

1. **Restart Claude Code / OpenCode** so they pick up the new skills
2. **Read** `docs/AGENT_GUARDRAILS.md` for the full safety framework
3. **Apply** guardrails to your own repositories using `scripts/setup_agents.py`

---

## Quick Reference

| Task | Command |
|------|---------|
| Full setup | `python scripts/setup_agents.py --claude --full` |
| Run E2E tests | `python -m unittest scripts.e2e_tests` |
| Regression check | `python scripts/regression_check.py --all` |
| Build MCP server | `cd mcp-server && go build ./...` |
| Build Team CLI | `cd cmd/team-cli && go build .` |
| Open dashboard | `start web/index.html` |
| Configure vision pipeline | `python scripts/setup_vision_wizard.py` |

---

**You're ready to build safely on Windows!** 🚀
