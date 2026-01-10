# Agent Guardrails Template

> Standard template for AI agent safety protocols, sprint task framework, and repository best practices.

---

## For AI Agents: How to Apply This Template

**If you are an LLM (Claude, GPT, Gemini, LLaMA, etc.) reading this README, follow these instructions to apply these standards to a repository.**

### Option A: Apply to an EXISTING Repository

Execute these steps in order:

```
STEP 1: Create docs directory structure
────────────────────────────────────────
ACTION: Create directories if they don't exist
COMMAND: mkdir -p docs/sprints/archive

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
────────────────────────────────────────
FILES TO COPY:
  - docs/AGENT_GUARDRAILS.md
  - docs/sprints/SPRINT_TEMPLATE.md
  - docs/sprints/SPRINT_GUIDE.md
  - docs/sprints/INDEX.md
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
[ ] docs/AGENT_GUARDRAILS.md exists and is complete
[ ] docs/sprints/SPRINT_TEMPLATE.md exists
[ ] docs/sprints/SPRINT_GUIDE.md exists
[ ] docs/sprints/INDEX.md exists
[ ] README.md links to Agent Guardrails
[ ] README.md has AI agent notice in Contributing section
[ ] .gitignore exists (optional but recommended)
[ ] .github/PULL_REQUEST_TEMPLATE.md exists (optional)
```

---

## Template Contents

### Files Included

| File | Purpose | Required? |
|------|---------|-----------|
| `docs/AGENT_GUARDRAILS.md` | Safety protocols for all AI agents | **YES** |
| `docs/sprints/SPRINT_TEMPLATE.md` | Template for agent task documents | **YES** |
| `docs/sprints/SPRINT_GUIDE.md` | How to write effective sprints | **YES** |
| `docs/sprints/INDEX.md` | Sprint navigation and tracking | **YES** |
| `.github/PULL_REQUEST_TEMPLATE.md` | PR template with AI attribution | Recommended |
| `.github/ISSUE_TEMPLATE/bug_report.md` | Bug report template | Recommended |
| `.gitignore` | Common ignore patterns | Recommended |

### Key Documents

| Document | Description |
|----------|-------------|
| [**Agent Guardrails**](docs/AGENT_GUARDRAILS.md) | **MANDATORY** safety protocols for ALL AI agents and LLMs |
| [**Sprint Template**](docs/sprints/SPRINT_TEMPLATE.md) | Copy this to create new agent-executable tasks |
| [**Sprint Guide**](docs/sprints/SPRINT_GUIDE.md) | Instructions for writing effective sprint documents |

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
