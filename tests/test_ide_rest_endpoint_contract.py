"""Contract tests for IDE REST defaults and explicit endpoint overrides."""

from __future__ import annotations

import re
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[1]
REST_URL = "http://localhost:8081"
MCP_URL = "http://localhost:8080"

CLIENTS = {
    "VS Code": (
        "ide/vscode-extension/src/utils/client.ts",
        re.compile(r"serverUrl:\s*config\.get<string>\('serverUrl',\s*'([^']+)'\)"),
        ("ide/vscode-extension/install.md",),
    ),
    "JetBrains": (
        "ide/jetbrains-plugin/src/main/kotlin/com/guardrail/plugin/GuardrailService.kt",
        re.compile(r'var serverUrl:\s*String\s*=\s*"([^"]+)"'),
        ("ide/jetbrains-plugin/README.md",),
    ),
    "Vim": (
        "ide/vim-plugin/plugin/guardrail.vim",
        re.compile(r"let g:guardrail_server_url\s*=\s*get\(g:,\s*'guardrail_server_url',\s*'([^']+)'\)"),
        ("ide/vim-plugin/README.md", "ide/vim-plugin/install.md"),
    ),
    "Neovim": (
        "ide/neovim-plugin/lua/guardrail/init.lua",
        re.compile(r'server_url\s*=\s*"([^"]+)"'),
        ("ide/neovim-plugin/README.md",),
    ),
}


def read_repo_file(relative_path: str) -> str:
    path = REPO_ROOT / relative_path
    assert path.is_file(), f"required contract file is missing: {relative_path}"
    return path.read_text(encoding="utf-8")


def mismatched_defaults(defaults: dict[str, str]) -> dict[str, str]:
    return {client: value for client, value in defaults.items() if value != REST_URL}


def test_all_ide_defaults_and_install_examples_target_rest_listener() -> None:
    observed_defaults: dict[str, str] = {}
    for client, (source_path, pattern, example_paths) in CLIENTS.items():
        source = read_repo_file(source_path)
        match = pattern.search(source)
        assert match is not None, f"{client}: declared default was not found in {source_path}"
        observed_defaults[client] = match.group(1)

        for example_path in example_paths:
            example = read_repo_file(example_path)
            assert REST_URL in example, f"{client}: REST default/example missing from {example_path}"

    assert set(observed_defaults) == {"VS Code", "JetBrains", "Vim", "Neovim"}
    mismatches = mismatched_defaults(observed_defaults)
    assert not mismatches, f"IDE defaults must target REST port 8081; mismatches: {mismatches}"

    shared_ide_readme = read_repo_file("ide/README.md")
    assert shared_ide_readme.count(REST_URL) >= 5, "shared IDE setup examples must cover each client and common settings"
    assert REST_URL in read_repo_file("ide/testing-guide.md"), "IDE testing guide must target the REST listener"
    assert REST_URL in read_repo_file("ide/extensions-plan.md"), "local extension plan example must target REST port 8081"

    vscode_manifest = read_repo_file("ide/vscode-extension/package.json")
    assert '"default": "http://localhost:8081"' in vscode_manifest
    assert '"description": "Guardrail REST server URL"' in vscode_manifest
    assert "Guardrail REST Server URL" in read_repo_file("ide/vscode-extension/src/commands.ts")
    assert "await client.updateConfiguration(context)" in read_repo_file("ide/vscode-extension/src/commands.ts")

    # The package manifest points VS Code at checked-in runtime output, so its
    # fallbacks must stay aligned with the TypeScript sources too.
    compiled_client = read_repo_file("ide/vscode-extension/out/utils/client.js")
    compiled_commands = read_repo_file("ide/vscode-extension/out/commands.js")
    assert "config.get('serverUrl', 'http://localhost:8081')" in compiled_client
    assert "config.get('serverUrl', 'http://localhost:8081')" in compiled_commands
    assert "Guardrail REST Server URL" in compiled_commands

    scoped_docs = (
        "ide/README.md",
        "ide/extensions-plan.md",
        "ide/testing-guide.md",
        "ide/vscode-extension/install.md",
        "ide/jetbrains-plugin/README.md",
        "ide/vim-plugin/README.md",
        "ide/vim-plugin/install.md",
        "ide/neovim-plugin/README.md",
    )
    for path in scoped_docs:
        assert "http://localhost:8095" not in read_repo_file(path), f"stale localhost REST default remains in {path}"


def test_web_and_mcp_listener_defaults_remain_separate() -> None:
    config = read_repo_file("mcp-server/internal/config/config.go")
    assert re.search(r'WebPort\s+int\s+`env:"WEB_PORT"\s+envDefault:"8081"`', config)
    assert re.search(r'MCPPort\s+int\s+`env:"MCP_PORT"\s+envDefault:"8080"', config)
    assert MCP_URL in read_repo_file("integrations/copilot/.vscode/mcp.json")


def test_explicit_user_endpoint_overrides_remain_authoritative() -> None:
    vscode = read_repo_file("ide/vscode-extension/src/utils/client.ts")
    assert "config.get<string>('serverUrl', 'http://localhost:8081')" in vscode
    assert "`${this.config.serverUrl}${path}`" in vscode

    jetbrains = read_repo_file("ide/jetbrains-plugin/src/main/kotlin/com/guardrail/plugin/GuardrailService.kt")
    assert "settings = state" in jetbrains
    assert "settings = newSettings" in jetbrains
    assert 'settings.serverUrl.trim().removeSuffix("/")' in jetbrains

    vim_plugin = read_repo_file("ide/vim-plugin/plugin/guardrail.vim")
    vim_client = read_repo_file("ide/vim-plugin/autoload/guardrail.vim")
    assert "get(g:, 'guardrail_server_url', 'http://localhost:8081')" in vim_plugin
    assert "'server_url': g:guardrail_server_url" in vim_client
    assert "l:config.server_url . '/health/ready'" in vim_client

    neovim = read_repo_file("ide/neovim-plugin/lua/guardrail/init.lua")
    neovim_client = read_repo_file("ide/neovim-plugin/lua/guardrail/validation.lua")
    assert 'vim.tbl_deep_extend("force", M.config, opts or {})' in neovim
    assert "config.server_url" in neovim_client

    # This non-default production URL is an intentional configured override.
    assert '"url": "http://192.0.2.20:8095"' in read_repo_file("ide/extensions-plan.md")


def test_negative_fixture_names_a_stale_8095_default() -> None:
    stale_fixture = {
        "VS Code": "http://localhost:8095",
        "JetBrains": REST_URL,
        "Vim": REST_URL,
        "Neovim": REST_URL,
    }
    assert mismatched_defaults(stale_fixture) == {"VS Code": "http://localhost:8095"}
