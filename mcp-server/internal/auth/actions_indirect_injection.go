package auth

// Spec 05 (indirect prompt injection) tool classification.
//
// These live in their own map rather than inside mcpToolActions for a mundane
// but important reason: mcpToolActions is a single gofmt alignment group, and
// inserting longer keys reflows every existing line in it. One of those lines
// is the anchor of the mutation fixture
// "injection-tools-unclassified-in-action-table", which matches by exact text —
// so an unrelated, purely cosmetic reflow breaks a security regression test.
// Keeping the new entries separate leaves the audited table byte-identical.
//
// Classification is not optional: ClassifyMCPTool denies any tool it cannot
// classify, so an unregistered name is uncallable no matter how correct its
// handler is.
var indirectInjectionToolActions = map[string]ActionKind{
	// scan_external_content analyses text and returns a verdict.
	"guardrail_scan_external_content": ActionRead,
	// mark_provenance returns a decorated string; it writes no state.
	"guardrail_mark_provenance": ActionRead,
	// check_provenance reads a process-local record.
	"guardrail_check_provenance": ActionRead,
}
