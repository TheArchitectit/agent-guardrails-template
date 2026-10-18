#!/usr/bin/env python3
"""Red-on-drift self-test for scripts/check_tool_count.py (F001).

The gate this replaces went vacuous without anyone noticing: its scope was a
regex over prose, a doc rewrite stopped matching, the loop body never ran, and
it exited 0. So the cases below are not only "does it catch a wrong number" —
they also prove the gate FAILS when a claim site stops matching at all, and
when the source scan finds nothing.

Run with: python -m pytest scripts/test_check_tool_count.py -v
Or: python scripts/test_check_tool_count.py
"""

import os
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from scripts import check_tool_count as gate

# A minimal source tree: 3 core tools, 2 vision, 1 webhook, 1 budget, 1 lifecycle.
CORE_LIST = '''package mcp
func coreTools() []mcp.Tool {
	tools := []mcp.Tool{
		{Name: "guardrail_alpha"},
		{Name: "guardrail_beta"},
	}
	return tools
}
'''
CORE_EXT = '''package mcp
func coreToolsExtended() []mcp.Tool {
	return []mcp.Tool{{Name: "guardrail_gamma"}}
}
'''
CORE_GUARD = '''package mcp
func (s *MCPServer) guardrailsToolList() []mcp.Tool {
	return []mcp.Tool{}
}
'''
# Spec 05 tools are appended by guardrailsToolList() with no backing store, so
# they are core. The fixture must contain this registry for the same reason the
# real tree must: a core registry that is missing from a tree is a gate
# FAILURE, not a skip (that is the vacuous-scope rule this gate exists to
# enforce). This file is empty of declarations here on purpose — it proves the
# gate tolerates a registry that currently declares nothing, while still
# requiring the registry to EXIST.
CORE_INDIRECT = '''package mcp
func (s *MCPServer) indirectInjectionTools() []mcp.Tool {
	return []mcp.Tool{}
}
'''
# Spec 04 tools are also registered unconditionally by toolList(), so they are
# core, and the fixture must contain the registry for the same reason Spec 05's
# file is here: a core registry missing from a tree is a FAILURE, not a skip.
# This is not hypothetical — adding tools_multi_agent.go to the real tree
# without adding it to CORE_REGISTRIES made the gate print core=42 and exit 0
# while toolList() advertised 45.
CORE_MULTI = '''package mcp
func (s *MCPServer) multiAgentTools() []mcp.Tool {
	return []mcp.Tool{}
}
'''
# Spec 06 compliance tools are registered unconditionally too, so they are core
# and the fixture must contain the registry for the same reason as the two
# above. This is the FOURTH occurrence of the class: without the real
# CORE_REGISTRIES entry the gate printed core=45 and exited 0 while toolList()
# advertised 48. The fixture is empty of declarations on purpose — it proves the
# gate tolerates a registry declaring nothing while still requiring it to exist.
CORE_COMPLIANCE = '''package mcp
func (s *MCPServer) complianceTools() []mcp.Tool {
	return []mcp.Tool{}
}
'''
# Spec 03 sandbox tools are registered unconditionally too, so they are core.
# FIFTH occurrence of the identical class: without the real CORE_REGISTRIES
# entry the gate printed core=48 and exited 0 while toolList() advertised 50.
# The fixture declares nothing of its own on purpose — it proves the gate
# requires the registry to exist while tolerating one that declares no tools,
# which is exactly the property the real vacuity came through.
CORE_SANDBOX = '''package mcp
func (s *MCPServer) sandboxTools() []mcp.Tool {
	return []mcp.Tool{}
}
'''
VISION = '''package mcp
func (vt *VisionToolSet) visionToolList() []mcp.Tool {
	return []mcp.Tool{{Name: "vision_one"}, {Name: "vision_two"}}
}
'''
WEBHOOK = '''package mcp
func (s *MCPServer) notificationToolList() []mcp.Tool {
	return []mcp.Tool{{Name: "configure_webhook"}}
}
'''
BUDGET = '''package mcp
func (s *MCPServer) budgetToolList() []mcp.Tool {
	return []mcp.Tool{{Name: "configure_budget"}}
}
'''
LIFECYCLE = '''package mcp
func (s *MCPServer) lifecycleToolList() []mcp.Tool {
	return []mcp.Tool{{Name: "create_agent_session"}}
}
'''

# 3 core, 5 conditional (2+1+1+1), max 8.
TOOLS_REF = """# MCP Tools Reference

The guardrails MCP server registers **3 core tools**, dispatched from the
switch and defined by `toolList()`.

A further **5 tools** are registered only when their backing store is
initialised — 2 vision, 1 webhook, 1 budget, 1 agent-lifecycle. So a
deployment exposes between 3 and 8 tools depending on configuration.
"""

COND_PAGE = """# Conditionally Registered and Integration Tools

5 tools registered **only when their backing store is initialised**. A
default deployment exposes the 3 core tools; a fully wired deployment
exposes 8. `toolList()` in `tools_registry.go` governs the gating.

| Group | Count | Registered when | Dispatch route |
|-------|-------|-----------------|----------------|
| Vision | 2 | `s.visionTools != nil` | Vision sub-router |
| Webhooks | 1 | `s.webhookStore != nil` | Main switch |
| Budgets | 1 | `s.budgetStore != nil` | Main switch |
| Agent lifecycle | 1 | `s.agentStateStore != nil` | Main switch |
"""

MCP_README = "The server registers 3 `guardrail_*` tools. The core validation set:\n"
ARCH = "| Tools | Business logic | Team, guardrail, agent operations (3 core tools) |\n"
STATUS = "- Features: Full tool set (3 core tools, 11 resources), migrations applied cleanly\n"
SPEC09 = (
    "- The MCP server has 3 core tools and up to 5 conditionally registered\n"
    "  tools. Published schemas and handlers disagree.\n"
)


def _write(root: Path, rel: str, body: str) -> None:
    p = root / rel
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(body, encoding="utf-8")


def _build(root: Path) -> None:
    mcp = "mcp-server/internal/mcp"
    _write(root, f"{mcp}/tools_core_list.go", CORE_LIST)
    _write(root, f"{mcp}/tools_core_list_extended.go", CORE_EXT)
    _write(root, f"{mcp}/tools_guardrails.go", CORE_GUARD)
    _write(root, f"{mcp}/tools_indirect_injection.go", CORE_INDIRECT)
    _write(root, f"{mcp}/tools_multi_agent.go", CORE_MULTI)
    _write(root, f"{mcp}/tools_compliance.go", CORE_COMPLIANCE)
    _write(root, f"{mcp}/tools_sandbox.go", CORE_SANDBOX)
    _write(root, f"{mcp}/vision_tools.go", VISION)
    _write(root, f"{mcp}/tools_notifications.go", WEBHOOK)
    _write(root, f"{mcp}/tools_budget.go", BUDGET)
    _write(root, f"{mcp}/tools_lifecycle.go", LIFECYCLE)
    _write(root, "docs/mcp-server/tools-reference.md", TOOLS_REF)
    _write(root, "docs/mcp-server/tools/conditional-and-integrations.md", COND_PAGE)
    _write(root, "mcp-server/README.md", MCP_README)
    _write(root, "docs/architecture/system-architecture.md", ARCH)
    _write(root, "docs/status.md", STATUS)
    _write(root, "docs/specs/09-system-roadmap-and-phase-gates.md", SPEC09)


class CheckToolCountTest(unittest.TestCase):
    def _run(self, root: Path) -> int:
        return gate.main(["--repo", str(root)])

    def test_clean_tree_passes(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _build(root)
            self.assertEqual(self._run(root), gate.EXIT_OK)

    def test_wrong_core_count_fails(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _build(root)
            _write(root, "docs/architecture/system-architecture.md", ARCH.replace("(3 core", "(9 core"))
            self.assertEqual(self._run(root), gate.EXIT_VIOLATIONS)

    def test_wrong_conditional_count_fails(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _build(root)
            _write(root, "docs/mcp-server/tools/conditional-and-integrations.md",
                   COND_PAGE.replace("| Vision | 2 |", "| Vision | 7 |"))
            self.assertEqual(self._run(root), gate.EXIT_VIOLATIONS)

    def test_rewritten_prose_fails_instead_of_going_vacuous(self):
        """The exact regression: the phrase changes, the old gate passed silently."""
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _build(root)
            # Same meaning, different words, no number where the pattern looks.
            _write(root, "mcp-server/README.md",
                   "The server exposes a set of `guardrail_*` tools.\n")
            self.assertEqual(self._run(root), gate.EXIT_VIOLATIONS)

    def test_missing_claim_site_fails(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _build(root)
            (root / "docs/status.md").unlink()
            self.assertEqual(self._run(root), gate.EXIT_VIOLATIONS)

    def test_zero_core_tools_fails(self):
        """Non-vacuity: an empty scan must never be green."""
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _build(root)
            _write(root, "mcp-server/internal/mcp/tools_core_list.go",
                   "package mcp\nfunc coreTools() []mcp.Tool { return nil }\n")
            _write(root, "mcp-server/internal/mcp/tools_core_list_extended.go",
                   "package mcp\nfunc coreToolsExtended() []mcp.Tool { return nil }\n")
            self.assertEqual(self._run(root), gate.EXIT_VIOLATIONS)

    def test_tool_counted_as_both_core_and_conditional_fails(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _build(root)
            # Declare a core tool inside a conditional registry.
            _write(root, "mcp-server/internal/mcp/tools_budget.go",
                   'package mcp\nfunc (s *MCPServer) budgetToolList() []mcp.Tool {\n'
                   '\treturn []mcp.Tool{{Name: "guardrail_alpha"}}\n}\n')
            self.assertEqual(self._run(root), gate.EXIT_VIOLATIONS)

    def test_detects_a_tool_added_to_source_without_doc_update(self):
        """The drift direction that actually bit us."""
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _build(root)
            _write(root, "mcp-server/internal/mcp/tools_guardrails.go",
                   'package mcp\nfunc (s *MCPServer) guardrailsToolList() []mcp.Tool {\n'
                   '\treturn []mcp.Tool{{Name: "guardrail_new"}}\n}\n')
            self.assertEqual(self._run(root), gate.EXIT_VIOLATIONS)


if __name__ == "__main__":
    unittest.main(verbosity=2)
