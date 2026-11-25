'use strict';

/**
 * Regression for failure FAIL-4c35b1a8.
 * Bug: activate() created GuardrailStatusBar before client configuration, so
 * status rendering threw while reading config.enabled before readiness ran.
 * Fix: initialize with ExtensionContext before status UI/request setup and
 * retain that context for configuration refreshes.
 * Source: ide/vscode-extension/src/extension.ts.
 */

const assert = require('node:assert/strict');
const Module = require('node:module');
const path = require('node:path');

const settings = {
    enabled: true,
    serverUrl: 'https://guardrail.example.test',
    apiKey: '',
    projectSlug: '',
    validateOnSave: true,
    validateOnType: false,
    severityThreshold: 'warning'
};
const calls = [];
let configurationListener;
let statusItem;

const vscode = {
    StatusBarAlignment: { Right: 2 },
    ThemeColor: class ThemeColor {
        constructor(id) { this.id = id; }
    },
    window: {
        createOutputChannel: () => ({ appendLine() {}, show() {} }),
        createStatusBarItem: () => {
            statusItem = {
                text: '', tooltip: '', backgroundColor: undefined,
                show() {}, dispose() {}
            };
            return statusItem;
        }
    },
    workspace: {
        getConfiguration: () => ({ get: (key, fallback) => settings[key] ?? fallback }),
        onDidSaveTextDocument: () => ({ dispose() {} }),
        onDidChangeTextDocument: () => ({ dispose() {} }),
        onDidCloseTextDocument: () => ({ dispose() {} }),
        onDidChangeConfiguration: (listener) => {
            configurationListener = listener;
            return { dispose() {} };
        }
    },
    languages: {
        createDiagnosticCollection: () => ({
            set() {}, delete() {}, dispose() {}
        })
    },
    commands: { registerCommand: () => ({ dispose() {} }) }
};

const originalLoad = Module._load;
Module._load = function(request, parent, isMain) {
    if (request === 'vscode') return vscode;
    return originalLoad.call(this, request, parent, isMain);
};

const extension = require(path.resolve(
    __dirname,
    '../../ide/vscode-extension/out/extension.js'
));

async function activateWith(fetchImplementation) {
    calls.length = 0;
    configurationListener = undefined;
    global.fetch = async (url, options) => {
        calls.push({ url, options });
        return fetchImplementation(url, options);
    };

    const context = {
        subscriptions: [],
        secrets: { get: async (key) => key === 'guardrail.apiKey' ? 'fixture-secret' : undefined }
    };
    await extension.activate(context);
    return context;
}

function successfulResponse() {
    return { status: 200, ok: true, statusText: 'OK', json: async () => ({}) };
}

async function run() {
    const originalFetch = global.fetch;
    try {
        await activateWith(async () => successfulResponse());
        assert.equal(calls.length, 1, 'activation makes one readiness request');
        assert.equal(calls[0].url, 'https://guardrail.example.test/health/ready');
        assert.equal(calls[0].options.headers.Authorization, 'Bearer fixture-secret');
        assert.match(statusItem.text, /Guardrail$/);
        assert.equal(statusItem.tooltip, 'Guardrail connected - Click to configure');

        settings.serverUrl = 'https://refreshed.example.test';
        await configurationListener({ affectsConfiguration: (name) => name === 'guardrail' });
        assert.equal(calls[1].url, 'https://refreshed.example.test/health/ready');
        assert.equal(calls[1].options.headers.Authorization, 'Bearer fixture-secret');
        extension.deactivate();

        await activateWith(async () => { throw new Error('fixture server unreachable'); });
        assert.equal(calls[0].url, 'https://refreshed.example.test/health/ready');
        assert.equal(calls[0].options.headers.Authorization, 'Bearer fixture-secret');
        assert.equal(statusItem.tooltip, 'Guardrail disconnected: Connection failed - Click to configure');
        extension.deactivate();
    } finally {
        global.fetch = originalFetch;
        Module._load = originalLoad;
    }
}

run().catch((error) => {
    console.error(error);
    process.exitCode = 1;
});
