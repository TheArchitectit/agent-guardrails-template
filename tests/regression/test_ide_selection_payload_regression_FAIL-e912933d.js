/*
Regression test for FAILURE-ID: FAIL-e912933d

Bug: Maintained IDE clients sent selected source as `code` while the server
contract required `selection`.
Fix: All four request builders now send the canonical `selection` property;
the server retains `code`-only compatibility.
File: ide/{vscode-extension,jetbrains-plugin,vim-plugin,neovim-plugin}

Scope: Executes the compiled VS Code client callback against a mocked fetch
and checks the other three builders' source-level payload field. The Go test
separately proves the real handler route accepts canonical, legacy, and equal
dual-field payloads.
*/
'use strict';

const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const Module = require('node:module');

const root = path.resolve(__dirname, '../..');
const builders = [
  {
    name: 'VS Code',
    file: 'ide/vscode-extension/src/utils/client.ts',
    method: /async validateSelection\([\s\S]*?\n    }/,
    field: /\{\s*selection:\s*code\s*,\s*language\s*\}/,
  },
  {
    name: 'JetBrains',
    file: 'ide/jetbrains-plugin/src/main/kotlin/com/guardrail/plugin/GuardrailService.kt',
    method: /fun validateSelection\([\s\S]*?\n    }/,
    field: /"selection"\s+to\s+code/,
  },
  {
    name: 'Vim',
    file: 'ide/vim-plugin/autoload/guardrail.vim',
    method: /function! guardrail#ValidateSelection\([\s\S]*?\nendfunction/,
    field: /'selection':\s*l:code/,
  },
  {
    name: 'Neovim',
    file: 'ide/neovim-plugin/lua/guardrail/validation.lua',
    method: /function M\.validate_selection\([\s\S]*?\nend/,
    field: /selection\s*=\s*code/,
  },
];

async function main() {
  for (const builder of builders) {
    const source = fs.readFileSync(path.join(root, builder.file), 'utf8');
    const match = source.match(builder.method);
    assert.ok(match, `${builder.name} request builder exists`);
    assert.match(match[0], builder.field, `${builder.name} sends selection`);
    assert.doesNotMatch(match[0], /(?:"code"\s+to\s+code|'code'\s*:|\bcode\s*=\s*code|\{\s*code\s*,)/,
      `${builder.name} does not send a legacy code property`);
  }

  const originalLoad = Module._load;
  Module._load = function(request, parent, isMain) {
    if (request === 'vscode') {
      return { workspace: { getConfiguration: () => ({ get: (_key, fallback) => fallback }) } };
    }
    return originalLoad.call(this, request, parent, isMain);
  };
  let clientModule;
  try {
    clientModule = require(path.join(root, 'ide/vscode-extension/out/utils/client.js'));
  } finally {
    Module._load = originalLoad;
  }

  let request;
  global.fetch = async (url, options) => {
    request = { url, options };
    return { ok: true, json: async () => ({ valid: true, violations: [] }) };
  };
  const client = new clientModule.GuardrailClient({ appendLine() {} });
  await client.updateConfiguration();
  await client.validateSelection('  exact source\n', 'go');
  assert.equal(request.url, 'http://localhost:8081/ide/validate/selection');
  const payload = JSON.parse(request.options.body);
  assert.deepEqual(payload, { selection: '  exact source\n', language: 'go' });

  console.log('PASS VS Code compiled request callback');
  console.log('PASS JetBrains source builder');
  console.log('PASS Vim source builder');
  console.log('PASS Neovim source builder');
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
