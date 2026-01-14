# Project Guidelines

## 0. Navigation Maps (READ FIRST)
* **INDEX_MAP.md**: Read this FIRST to find documents by keyword/category. Saves 60-80% tokens.
* **HEADER_MAP.md**: Find specific sections with file:line references for targeted reading.
* **Flow**: INDEX_MAP → identify doc → HEADER_MAP → read specific section with offset

## 1. Context & Setup
* **Stack Detection**: Read configuration files (package.json, requirements.txt, Makefile, etc) to determine stack. Do NOT read lockfiles.
* **Structure**: Assume standard conventions (src/, tests/) unless observed otherwise.
* **Guardrails**: Read [docs/AGENT_GUARDRAILS.md](docs/AGENT_GUARDRAILS.md) before any code changes.

## 2. Token-Saving Rules (STRICT)
* **NO EXPLORATION**: Do not use "ls -R" or explore file structure.
* **NO RE-READING**: Trust your context; do not re-read files just edited.
* **TARGETED CONTEXT**: Read ONLY files explicitly relevant to the request.
* **CONCISE PLANS**: Bullet points only. No "thinking out loud".
* **USE MAPS**: Always check INDEX_MAP.md before reading full documents.

## 3. Workflow
* **Tests**: Run ONLY relevant tests.
* **Edits**: Prefer small, single-file edits.
* **Commits**: Commit after each to-do item (see [COMMIT_WORKFLOW.md](docs/workflows/COMMIT_WORKFLOW.md)).
* **Checkpoints**: Use MCP checkpoints before/after critical operations.

## 4. Documentation Standards
* **500-Line Max**: No document over 500 lines.
* **Update Maps**: Update INDEX_MAP.md and HEADER_MAP.md when adding/changing docs.
