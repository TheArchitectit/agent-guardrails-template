'use strict';
// Lightweight wrapper utilities to propagate prompt logging across all workflow steps
// This module provides a generic higher-order wrapper to log prompts before/after a workflow step.
// Usage:
// const { withPromptLogging } = require('./utils/prompt-logger-wrappers');
// const wrappedStep = withPromptLogging(stepFn, {
//   logSpecBefore: (...args) => ({ turn: 1, role: 'user', text: args[0], context: { stage: 'prompt' } }),
//   logSpecAfter: (result, ...args) => ({ turn: 1, role: 'assistant', text: result?.response, context: { stage: 'response' } })
// });
// await wrappedStep(input);

const promptLogger = require('./prompt-logger');

function withPromptLogging(stepFn, { logSpecBefore, logSpecAfter } = {}) {
  if (typeof stepFn !== 'function') throw new TypeError('stepFn must be a function');
  return async function(...args) {
    // Pre-step logging
    try {
      if (typeof logSpecBefore === 'function') {
        const spec = logSpecBefore(...args);
        if (spec && typeof spec === 'object' && spec.prompt) {
          // log the prompt using the existing logTurn/logPrompt interface
          promptLogger.logTurn ? promptLogger.logTurn({ turn: spec.turn || 1, role: spec.role || 'user', text: spec.prompt, context: spec.context || {}, extra: spec.extra || {} }) : null;
        }
      }
    } catch (e) {
      // Silence wrapper errors to avoid breaking core flow
    }

    const result = await stepFn.apply(this, args);

    // Post-step logging
    try {
      if (typeof logSpecAfter === 'function') {
        const spec = logSpecAfter(result, ...args);
        if (spec && typeof spec === 'object' && spec.prompt) {
          promptLogger.logTurn ? promptLogger.logTurn({ turn: spec.turn || 1, role: spec.role || 'assistant', text: spec.prompt, context: spec.context || {}, extra: spec.extra || {} }) : null;
        }
      }
    } catch (e) {
      // Silently swallow logging errors
    }

    return result;
  };
}

module.exports = {
  withPromptLogging
};
