'use strict';

/**
 * Regression for failure FAIL-a01027ef.
 * Bug: Configure started asynchronous SecretStorage-backed configuration
 * refresh without awaiting it, so the first readiness probe could use the
 * previous/default URL instead of the saved non-default endpoint.
 * Fix: await GuardrailClient.updateConfiguration before probing connectivity.
 * Source: ide/vscode-extension/src/commands.ts.
 */

const assert = require('node:assert/strict');
const Module = require('node:module');
const path = require('node:path');

const commandHandlers = new Map();
const settings = {
    serverUrl: 'http://localhost:8081',
    apiKey: '',
    projectSlug: ''
};
const prompts = [
    'https://configured.example.test',
    'configured-secret',
    'synthetic-project'
];

const vscode = {
    commands: {
        registerCommand: (name, handler) => {
            commandHandlers.set(name, handler);
            return { dispose() {} };
        }
    },
    workspace: {
        getConfiguration: () => ({
            get: (key, fallback) => settings[key] ?? fallback,
            update: async (key, value) => { settings[key] = value; }
        })
    },
    window: {
        showInputBox: async () => prompts.shift(),
        showInformationMessage() {},
        showErrorMessage() {}
    }
};

const originalLoad = Module._load;
Module._load = function(request, parent, isMain) {
    if (request === 'vscode') return vscode;
    return originalLoad.call(this, request, parent, isMain);
};

const { registerCommands } = require(path.resolve(
    __dirname,
    '../../ide/vscode-extension/out/commands.js'
));
const { GuardrailClient } = require(path.resolve(
    __dirname,
    '../../ide/vscode-extension/out/utils/client.js'
));

async function run() {
    const calls = [];
    const originalFetch = global.fetch;
    let releaseSecretRead;
    try {
        global.fetch = async (url) => {
            calls.push(url);
            return { status: 200, ok: true, statusText: 'OK', json: async () => ({}) };
        };

        const client = new GuardrailClient({ appendLine() {} });
        const initialContext = { secrets: { get: async () => undefined } };
        await client.updateConfiguration(initialContext);

        const context = {
            subscriptions: [],
            secrets: {
                store: async () => {},
                get: () => new Promise((resolve) => { releaseSecretRead = resolve; })
            }
        };
        const statusBar = { setConnected() {}, setDisconnected() {} };
        registerCommands(context, client, {}, statusBar, { show() {} });

        const configure = commandHandlers.get('guardrail.configure');
        assert.equal(typeof configure, 'function', 'Configure command is registered');
        const configureRun = configure();

        await new Promise((resolve) => setImmediate(resolve));
        assert.deepEqual(
            calls,
            [],
            `Configure must await SecretStorage-backed configuration before probing; premature requests: ${calls.join(', ')}`
        );

        releaseSecretRead('configured-secret');
        await configureRun;
        assert.deepEqual(calls, ['https://configured.example.test/health/ready']);
    } finally {
        global.fetch = originalFetch;
        Module._load = originalLoad;
    }
}

run().catch((error) => {
    console.error(error);
    process.exitCode = 1;
});
