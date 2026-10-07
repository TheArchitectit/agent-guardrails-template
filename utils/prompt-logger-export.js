'use strict';
const fs = require('fs');
const path = require('path');
const { getLogPath } = require('./prompt-logger');

function exportLastNToCSV(n = 100, outPath = path.resolve(__dirname, '../../prompt-logs/prompts-lastN.csv')) {
  try {
    const logPath = getLogPath();
    const data = fs.readFileSync(logPath, 'utf8').trim().split('\n').slice(-n).map(line => {
      try {
        const obj = JSON.parse(line);
        const ts = obj.ts || '';
        const id = obj.id || '';
        const role = obj.role || '';
        const prompt = obj.prompt ? (typeof obj.prompt === 'string' ? obj.prompt.replace(/\n/g, ' ') : '') : '';
        const context = obj.context ? JSON.stringify(obj.context) : '';
        const extra = obj.extra ? JSON.stringify(obj.extra) : '';
        const escaped = [...[ts, id, role, prompt, context, extra].map(v => {
          if (!v) return '';
          return `"${String(v).replace(/"/g, '""')}"`;
        })].join(',');
        return escaped;
      } catch (e) {
        return '';
      }
    }).filter(Boolean).join('\n');
    fs.writeFileSync(outPath, data, 'utf8');
    return { path: outPath, lines: data.split('\n').length };
  } catch (e) {
    return { error: e.message };
  }
}

module.exports = { exportLastNToCSV };
