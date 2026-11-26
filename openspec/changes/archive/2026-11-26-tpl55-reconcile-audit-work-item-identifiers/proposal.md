# Give audit work items unique identifiers

## Why

The future-state audit reused TPL-49, TPL-50, and TPL-51 for open architecture work. The same identifiers already refer to archived cleanup, local CI verification, and the regex-cache repair. A reviewer following an identifier alone can reach the wrong acceptance record.

## What changes

Allocate unused identifiers to the three active architecture proposals. Align their change paths, the gap register, the integration audit, and the verification queue. Preserve the identifiers and evidence for the previously archived work.

## Capabilities

Extend `openspec-catalog-identity` with a requirement for unambiguous audit work-item identifiers.

## Impact

The public audit, task queue, and three change paths change. Each architecture requirement and its acceptance status stays tied to the same proposal content.
