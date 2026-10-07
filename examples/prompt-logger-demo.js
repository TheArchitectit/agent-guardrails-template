'use strict';

// Demo script to showcase prompt-logger usage
// Run: node examples/prompt-logger-demo.js

const promptLogger = require('../agent-guardrails-template/utils/prompt-logger');

async function runDemo(){
  // Log a user prompt
  const uid = 'demo-001';
  promptLogger.logPrompt({
    id: uid,
    role: 'user',
    prompt: 'Describe the plan for recording prompts in a contest.',
    context: { scenario: 'contest', stage: 'demo' },
    extra: { note: 'user prompt for demo' }
  });

  // Log an assistant response
  promptLogger.logPrompt({
    id: uid,
    role: 'assistant',
    prompt: 'We log prompts as JSONL with timestamps for auditability.',
    context: { scenario: 'contest', stage: 'demo' },
    extra: { note: 'assistant response for demo' }
  });

  console.log('Prompt logging demo complete. Log file path:', promptLogger.getLogPath());
}

runDemo().catch(err => {
  console.error('Demo failed', err);
});
