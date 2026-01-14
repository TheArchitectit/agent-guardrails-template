# Agent Guardrails Template

> Standard template for AI agent safety protocols, sprint task framework, and repository best practices.

---

## For AI Agents: How to Apply This Template

**If you are an LLM (Claude, GPT, Gemini, LLaMA, etc.) reading this README, follow these instructions to apply these standards to a repository.**

### Option A: Apply to an EXISTING Repository

Execute these steps in order:

```
STEP 1: Create docs directory structure
────────────────────────────────
ACTION: Create directories if they don't exist
COMMAND: mkdir -p docs/sprints/archive

STEP 1.5: Copy Claude Configuration
────────────────────────────────
ACTION: Read CLAUDE.md from this template
ACTION: Write to TARGET_REPO/CLAUDE.md

ACTION: Read .claudeignore from this template
ACTION: Write to TARGET_REPO/.claudeignore

STEP 2: Copy AGENT_GUARDRAILS.md
────────────────────────────────────────
ACTION: Read docs/AGENT_GUARDRAILS.md from this template
ACTION: Write to TARGET_REPO/docs/AGENT_GUARDRAILS.md

STEP 3: Copy Sprint Framework
────────────────────────────────────────
ACTION: Read docs/sprints/SPRINT_TEMPLATE.md from this template
ACTION: Write to TARGET_REPO/docs/sprints/SPRINT_TEMPLATE.md

ACTION: Read docs/sprints/SPRINT_GUIDE.md from this template
ACTION: Write to TARGET_REPO/docs/sprints/SPRINT_GUIDE.md

ACTION: Read docs/sprints/INDEX.md from this template
ACTION: Write to TARGET_REPO/docs/sprints/INDEX.md

STEP 4: Update target README.md
────────────────────────────────────────
ACTION: Add to Documentation section:
| [**Agent Guardrails**](docs/AGENT_GUARDRAILS.md) | **MANDATORY** safety protocols for ALL AI agents |

ACTION: Add to Contributing section:
> **AI Agents:** Before contributing, read [Agent Guardrails](docs/AGENT_GUARDRAILS.md)

STEP 5: Copy GitHub templates (optional)
────────────────────────────────────────
ACTION: Create .github directory if needed
ACTION: Copy .github/PULL_REQUEST_TEMPLATE.md
ACTION: Copy .github/ISSUE_TEMPLATE/bug_report.md

STEP 6: Commit changes
────────────────────────────────────────
COMMAND: git add docs/ .github/
COMMAND: git commit -m "docs: add agent guardrails and sprint framework

Added standardized AI agent safety protocols and sprint task templates.
Applies to all LLMs and automated systems.

Co-Authored-By: [Your Agent Name] <noreply@provider.com>"
```

### Option B: Create a NEW Repository with Standards

Execute these steps in order:

```
STEP 1: Create new repository
────────────────────────────────────────
COMMAND: mkdir new-project && cd new-project
COMMAND: git init

STEP 2: Create directory structure
────────────────────────────────────────
COMMAND: mkdir -p src tests docs/sprints/archive .github/ISSUE_TEMPLATE

STEP 3: Copy all template files
────────────────────────────────
FILES TO COPY:
  - INDEX_MAP.md (navigation map)
  - HEADER_MAP.md (section lookup)
  - CLAUDE.md
  - .claudeignore
  - docs/AGENT_GUARDRAILS.md
  - docs/workflows/ (all 8 files)
  - docs/standards/ (all 4 files)
  - docs/sprints/SPRINT_TEMPLATE.md
  - docs/sprints/SPRINT_GUIDE.md
  - docs/sprints/INDEX.md
  - .github/SECRETS_MANAGEMENT.md
  - .github/workflows/ (all 3 files)
  - .github/PULL_REQUEST_TEMPLATE.md
  - .github/ISSUE_TEMPLATE/bug_report.md
  - .gitignore

STEP 4: Create README.md
────────────────────────────────────────
ACTION: Use the PROJECT README TEMPLATE section below
ACTION: Customize for the specific project

STEP 5: Initial commit
────────────────────────────────────────
COMMAND: git add -A
COMMAND: git commit -m "feat: initial project setup with agent guardrails

- Project structure initialized
- Agent guardrails and sprint framework included
- GitHub templates configured

Co-Authored-By: [Your Agent Name] <noreply@provider.com>"

STEP 6: Create GitHub repo (if requested)
────────────────────────────────────────
COMMAND: gh repo create PROJECT_NAME --private --source=. --push
```

### Verification Checklist

After applying the template, verify:

```
NAVIGATION MAPS:
[ ] INDEX_MAP.md exists at root
[ ] HEADER_MAP.md exists at root

CORE DOCUMENTATION:
[ ] docs/AGENT_GUARDRAILS.md exists and is complete
[ ] docs/workflows/ contains 8 workflow documents
[ ] docs/standards/ contains 4 standards documents
[ ] docs/sprints/SPRINT_TEMPLATE.md exists
[ ] docs/sprints/SPRINT_GUIDE.md exists
[ ] docs/sprints/INDEX.md exists

GITHUB INTEGRATION:
[ ] .github/SECRETS_MANAGEMENT.md exists
[ ] .github/workflows/ contains 3 CI workflows
[ ] .github/PULL_REQUEST_TEMPLATE.md exists

PROJECT FILES:
[ ] CLAUDE.md configured for project
[ ] README.md links to INDEX_MAP.md and Agent Guardrails
[ ] .gitignore exists
[ ] All docs under 500 lines
```

---

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
| `docs/AGENT_GUARDRAILS.md` | Safety protocols for all AI agents | **YES** |
| `docs/workflows/*.md` | 8 workflow documents | **YES** |
| `docs/standards/*.md` | 4 standards documents | **YES** |
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

Co-Authored-By: Your Name <email@example.com>
```

Types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `perf`, `security`

---

## License

[Choose your license - MIT, Apache 2.0, etc.]

---

## Acknowledgments

- [Credit libraries, tools, or people]

---

**Last Updated:** YYYY-MM-DD
