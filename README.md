# Agent Guardrails Template

> Standard template for AI agent safety protocols, sprint task framework, and repository best practices.

---

## Authorship / Attribution

- **Maintainer/Author:** TheArchitectit
- **AI tooling note:** Created with Claude Code and Opus (no emails, no company contact instructions)

---

## For AI Agents: How to Apply This Template

**If you are an LLM (Claude, GPT, Gemini, LLaMA, etc.) reading this README, see [docs/HOW_TO_APPLY.md](docs/HOW_TO_APPLY.md) for detailed instructions and ready-to-use prompts.**

**Quick Start Options:**

- **Option A:** Apply to an **existing repository** → [docs/HOW_TO_APPLY.md#option-a](docs/HOW_TO_APPLY.md#option-a-apply-to-an-existing-repository)
- **Option B:** Use **example AI agent prompts** → [docs/HOW_TO_APPLY.md#option-b](docs/HOW_TO_APPLY.md#option-b-example-ai-agent-prompts)
- **Option C:** Create a **NEW repository** → [docs/HOW_TO_APPLY.md#option-c](docs/HOW_TO_APPLY.md#option-c-create-a-new-repository-with-standards)
- **Option D:** **Migrate existing documentation** → [docs/HOW_TO_APPLY.md#option-d](docs/HOW_TO_APPLY.md#option-d-migrate-existing-documentation-to-guardrails-structure)
## Template Contents

### Files Included

| File | Purpose | Required? |
|------|---------|-----------|
| **Navigation Maps** | | |
| `INDEX_MAP.md` | Master navigation - find docs by keyword | **YES** |
| `HEADER_MAP.md` | Section headers with line numbers | **YES** |
| `CLAUDE.md` | Optimized context for Claude Code CLI | Recommended |
| `.claudeignore` | Token-saving ignore rules | Recommended |
| **Core Documentation** | | |
| `docs/AGENT_GUARDRAILS.md` | Core guardrails (MANDATORY) | **YES** |
| `docs/standards/TEST_PRODUCTION_SEPARATION.md` | Test/production isolation (MANDATORY) | **YES** |
| `docs/workflows/*.md` | 10 workflow documents | **YES** |
| `docs/standards/*.md` | 5 standards documents | **YES** |
| `docs/sprints/*.md` | Sprint framework | **YES** |
| **GitHub Integration** | | |
| `.github/SECRETS_MANAGEMENT.md` | GitHub Secrets guide | Recommended |
| `.github/workflows/*.yml` | CI/CD workflows | Recommended |
| `.github/PULL_REQUEST_TEMPLATE.md` | PR template with AI attribution | Recommended |
| `.github/ISSUE_TEMPLATE/bug_report.md` | Bug report template | Recommended |
| `.gitignore` | Common ignore patterns | Recommended |

### Key Documents

| Document | Description |
|----------|-------------|
| [**INDEX_MAP.md**](INDEX_MAP.md) | Start here - find docs by keyword (saves 60-80% tokens) |
| [**Agent Guardrails**](docs/AGENT_GUARDRAILS.md) | **MANDATORY** safety protocols for ALL AI agents and LLMs |
| [**Commit Workflow**](docs/workflows/COMMIT_WORKFLOW.md) | When/how to commit between to-dos |
| [**Testing Validation**](docs/workflows/TESTING_VALIDATION.md) | Double-check work before committing |
| [**Logging Patterns**](docs/standards/LOGGING_PATTERNS.md) | Array-based structured logging |
| [**Sprint Template**](docs/sprints/SPRINT_TEMPLATE.md) | Copy this to create new agent-executable tasks |

---

## Using This Template via GitHub

### Create New Repo from Template (Humans)

1. Click **"Use this template"** button above
2. Select **"Create a new repository"**
3. Name your repository and set visibility
4. Click **"Create repository"**

### Create New Repo from Template (CLI)

```bash
gh repo create my-new-project \
  --template TheArchitectit/agent-guardrails-template \
  --private \
  --clone
```

---

## PROJECT README TEMPLATE

**Copy everything below this line when creating a new project README:**

---

# Project Name

> Brief one-line description of the project.

---

## Quick Start

```bash
# Clone the repository
git clone https://github.com/YOUR_USERNAME/YOUR_REPO.git
cd YOUR_REPO

# Install dependencies
# [Add your install commands here]

# Run the project
# [Add your run commands here]
```

---

## Overview

[Provide a 2-3 paragraph overview of what this project does, why it exists, and who it's for.]

---

## Features

- Feature 1 - Brief description
- Feature 2 - Brief description
- Feature 3 - Brief description

---

## Architecture

```
project/
├── CLAUDE.md      # Claude Code CLI guidelines
├── .claudeignore  # Token-saving ignore rules
├── src/           # Source code
├── tests/         # Test files
├── docs/          # Documentation
│   ├── AGENT_GUARDRAILS.md  # AI agent safety protocols
│   └── sprints/   # Sprint task documents
└── README.md
```

---

## Documentation

| Document | Description |
|----------|-------------|
| [**Agent Guardrails**](docs/AGENT_GUARDRAILS.md) | **MANDATORY** safety protocols for ALL AI agents and LLMs |
| [**Sprint Template**](docs/sprints/SPRINT_TEMPLATE.md) | Template for creating agent-executable sprint tasks |
| [**Sprint Guide**](docs/sprints/SPRINT_GUIDE.md) | How to write effective sprint documents |

---

## Environment Variables

```bash
# Required
VARIABLE_NAME=value          # Description

# Optional
OPTIONAL_VAR=default_value   # Description
```

---

## Development

### Prerequisites

- [List required tools and versions]

### Setup

```bash
# Development setup commands
```

### Testing

```bash
# Test commands
```

---

## Contributing

> **AI Agents:** Before contributing to this codebase, you MUST read and follow the [Agent Guardrails](docs/AGENT_GUARDRAILS.md). These protocols are mandatory for all LLMs, coding assistants, and autonomous agents.

### For Human Contributors

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Commit Message Format

```
<type>(<scope>): <short description>

<longer description if needed>

AI-assisted: Claude Code and Opus
```

Types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `perf`, `security`

---

## License

BSD-3-Clause - See [LICENSE](LICENSE) file for details.

---

## Acknowledgments

- [Credit libraries, tools, or people]

---

**Last Updated:** YYYY-MM-DD
