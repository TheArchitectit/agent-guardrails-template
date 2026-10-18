package guardrails

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
)

// InjectionDefenseValidator delegates to the Spec 01 injection detection pipeline.
type InjectionDefenseValidator struct {
	pipeline interface {
		Detect(ctx context.Context, text string, source Source) InjectionResult
	}
}

// NewInjectionDefenseValidator creates an injection validator wrapping a pipeline.
func NewInjectionDefenseValidator(pipeline interface {
	Detect(ctx context.Context, text string, source Source) InjectionResult
}) *InjectionDefenseValidator {
	return &InjectionDefenseValidator{pipeline: pipeline}
}

// Validate implements SafetyValidator.
func (v *InjectionDefenseValidator) Validate(ctx context.Context, input ValidatorInput) (*ValidatorResult, error) {
	result := v.pipeline.Detect(ctx, input.Output, SourceToolOutput)
	return &ValidatorResult{
		Passed:     result.Safe,
		Reason:     result.Reason,
		Confidence: result.Confidence,
		Violations: result.Categories,
	}, nil
}

// Name implements SafetyValidator.
func (v *InjectionDefenseValidator) Name() string {
	return "injection_defense"
}

// Description implements SafetyValidator.
func (v *InjectionDefenseValidator) Description() string {
	return "Detects prompt injection in agent output (Spec 01)"
}

// ContentFilterValidator delegates to the Spec 02 semantic content filter.
type ContentFilterValidator struct {
	filter *ContentFilter
}

// NewContentFilterValidator creates a content filter validator.
func NewContentFilterValidator(filter *ContentFilter) *ContentFilterValidator {
	return &ContentFilterValidator{filter: filter}
}

// Validate implements SafetyValidator.
func (v *ContentFilterValidator) Validate(ctx context.Context, input ValidatorInput) (*ValidatorResult, error) {
	result, err := v.filter.Classify(ctx, input.Output, input.Direction)
	if err != nil {
		return nil, err
	}

	var violations []string
	for _, c := range result.Categories {
		if c.Action == ActionBlock {
			violations = append(violations, fmt.Sprintf("%s: %s", c.ID, c.Name))
		}
	}

	return &ValidatorResult{
		Passed:     result.Safe,
		Reason:     formatFilterReason(result),
		Violations: violations,
		Confidence: result.OverallRisk,
	}, nil
}

func formatFilterReason(result *ClassificationResult) string {
	if result.Safe {
		return ""
	}
	var blocked []string
	for _, c := range result.Categories {
		if c.Action == ActionBlock {
			blocked = append(blocked, c.ID)
		}
	}
	if len(blocked) > 0 {
		return fmt.Sprintf("Content blocked: %s", strings.Join(blocked, ", "))
	}
	return "Content flagged by policy"
}

// Name implements SafetyValidator.
func (v *ContentFilterValidator) Name() string {
	return "content_filter"
}

// Description implements SafetyValidator.
func (v *ContentFilterValidator) Description() string {
	return "Classifies content against safety taxonomy (Spec 02)"
}

// FourLawsValidator enforces the "Four Laws" of agent safety on agent output.
type FourLawsValidator struct {
	scopePatterns []string
}

// NewFourLawsValidator creates a Four Laws validator.
func NewFourLawsValidator(scopePatterns []string) *FourLawsValidator {
	return &FourLawsValidator{scopePatterns: scopePatterns}
}

// NewFourLawsValidatorChecked creates a Four Laws validator, refusing a scope
// list that cannot be enforced.
//
// An empty or whitespace-only pattern is not a harmless entry:
// strings.Contains(text, "") is always true, so one blank entry turns the Law-2
// check into "fail every output". A validator that cannot be configured is
// reported at construction rather than at the first agent call.
func NewFourLawsValidatorChecked(scopePatterns []string) (*FourLawsValidator, error) {
	for i, p := range scopePatterns {
		if strings.TrimSpace(p) == "" {
			return nil, fmt.Errorf("scope pattern %d is empty; an empty pattern matches every output", i)
		}
	}
	return &FourLawsValidator{scopePatterns: scopePatterns}, nil
}

// mentionMarkers are words that turn an occurrence of a forbidden token into a
// discussion of it. They exist because Law 2 shipped a false-positive
// generator: the check flagged an output for merely CONTAINING a configured
// scope keyword, so a runbook entry, a policy quote, or an explicit prohibition
// ("never run delete-db") failed as if the agent had performed the action.
// Spec 04 records that as a bug to fix before wiring the chain.
var mentionMarkers = []string{
	"mention", "mentions", "mentioned",
	"document", "documents", "documented",
	"describe", "describes", "described", "description",
	"forbid", "forbids", "forbidden", "prohibit", "prohibits", "prohibited",
	"never", "avoid", "avoids", "do not", "don't", "does not", "doesn't",
	"example", "e.g.", "such as", "named", "called", "the word", "the token",
}

// isMention reports whether the match at [start,end) is a discussion of the
// pattern rather than an instruction to perform it.
//
// Two contexts are recognised: the occurrence is quoted, or a mention marker
// appears shortly before it. The window is deliberately small (40 bytes) so a
// genuine directive elsewhere in a long output is not excused by a stray
// "never" at the top of the document.
//
// Named limit: this is a heuristic over prose, not a parser. An output that
// embeds a real directive inside a quotation ("the log said: run X") is treated
// as a mention and passes Law 2. That is the correct trade for a check whose
// previous behaviour failed compliant output, and it is stated rather than
// implied.
func isMention(norm string, start, end int) bool {
	if quotedAt(norm, start, end) {
		return true
	}
	windowStart := start - 40
	if windowStart < 0 {
		windowStart = 0
	}
	window := strings.ToLower(norm[windowStart:start])
	for _, marker := range mentionMarkers {
		if strings.Contains(window, marker) {
			return true
		}
	}
	return false
}

// quotedAt reports whether the span is enclosed by a matching quote character
// that opens before it and closes after it.
func quotedAt(s string, start, end int) bool {
	for _, q := range []string{`"`, "'", "`"} {
		open := strings.LastIndex(s[:start], q)
		if open < 0 {
			continue
		}
		// An odd number of the quote character before the span means the span
		// sits inside a quoted region; a closing quote after it confirms.
		if strings.Count(s[:start], q)%2 == 1 && strings.Contains(s[end:], q) {
			return true
		}
	}
	return false
}

func normalizeWhitespace(s string) string {
	zeroWidth := regexp.MustCompile(`[\x{200B}\x{200C}\x{200D}\x{2060}\x{FEFF}\x{00AD}]`)
	s = zeroWidth.ReplaceAllString(s, "")
	ws := regexp.MustCompile(`\s+`)
	return ws.ReplaceAllString(s, " ")
}

// Validate implements SafetyValidator.
func (v *FourLawsValidator) Validate(ctx context.Context, input ValidatorInput) (*ValidatorResult, error) {
	output := input.Output
	norm := normalizeWhitespace(output)
	lowerNorm := strings.ToLower(norm)

	harmPatterns := []*regexp.Regexp{
		// rm with any ordering of recursive/force flags (-rf, -fr, -Rf,
		// -r -f, -f -r, -rR) followed by an absolute path, home-relative
		// path, or variable expansion. Catches `rm -fr /`,
		// `rm -rf --no-preserve-root /`, `rm -rf /home`, `rm -rf "$var"`,
		// `rm -r -f /`.
		regexp.MustCompile(`(?i)\brm\b[^;|&\n]*-[a-z]*[rR][a-z]*[fF][a-z]*\b[^;|&\n]*\s+(\/|~|\$\{?\w|\$\w|"[^"]*\$)`),
		regexp.MustCompile(`(?i)\brm\b[^;|&\n]*-[a-z]*[fF][a-z]*[rR][a-z]*\b[^;|&\n]*\s+(\/|~|\$\{?\w|\$\w|"[^"]*\$)`),
		regexp.MustCompile(`(?i)\brm\b[^;|&\n]*-[a-z]*[rR]\b[^;|&\n]*-[a-z]*[fF]\b[^;|&\n]*\s+(\/|~|\$\{?\w|\$\w|"[^"]*\$)`),
		regexp.MustCompile(`(?i)\brm\b[^;|&\n]*-[a-z]*[fF]\b[^;|&\n]*-[a-z]*[rR]\b[^;|&\n]*\s+(\/|~|\$\{?\w|\$\w|"[^"]*\$)`),
		regexp.MustCompile(`(?i)mkfs\.[a-z0-9]+`),
		regexp.MustCompile(`(?i)dd\s+if=/dev/zero`),
		regexp.MustCompile(`:\(\)\s*\{\s*:\s*\|\s*:\s*&\s*\}\s*;\s*:`),
		// chmod 777 (with or without -R) on an absolute path. Catches
		// `chmod -R 777 /`, `chmod 777 /etc`, `chmod 777 /`.
		regexp.MustCompile(`(?i)\bchmod\b[^;|&\n]*(?:-[a-z]*[rR]?\b[^;|&\n]*)?\b777\b\s+(\/|\$\{?\w|\$\w)`),
		regexp.MustCompile(`(?i)>\s*/dev/sda`),
	}

	for _, re := range harmPatterns {
		if re.MatchString(lowerNorm) {
			return &ValidatorResult{
				Passed:     false,
				Reason:     fmt.Sprintf("Law 1 violation: destructive command detected (%s)", re.String()),
				Violations: []string{"harmful_command"},
				Confidence: 1.0,
			}, nil
		}
	}

	// Law 2: scope check — use normalized text for consistent matching.
	// Scope validation runs even when Context is empty: with no declared
	// scope the output cannot be verified against one, so the check must
	// flag that scope could not be confirmed rather than silently skip it.
	{
		if input.Context == "" {
			if len(v.scopePatterns) > 0 {
				return &ValidatorResult{
					Passed:     false,
					Reason:     "Law 2 violation: scope could not be verified (no declared context)",
					Violations: []string{"scope_unverified"},
					Confidence: 0.9,
				}, nil
			}
		} else {
			// A scope pattern is a forbidden action, not a forbidden word. The
			// shipped check failed any output CONTAINING the token, so a
			// runbook entry, a quoted policy, or the prohibition itself
			// ("never run delete-db") was reported as a violation. Occurrences
			// that are quoted or introduced by a mention marker are treated as
			// discussion; anything else still fails.
			for _, pattern := range v.scopePatterns {
				needle := strings.ToLower(pattern)
				searchFrom := 0
				for {
					idx := strings.Index(lowerNorm[searchFrom:], needle)
					if idx < 0 {
						break
					}
					start := searchFrom + idx
					if !isMention(norm, start, start+len(needle)) {
						return &ValidatorResult{
							Passed:     false,
							Reason:     fmt.Sprintf("Law 2 violation: output exceeds declared scope (%s)", pattern),
							Violations: []string{"scope_violation"},
							Confidence: 0.9,
						}, nil
					}
					searchFrom = start + len(needle)
				}
			}
		}
	}

	// Law 3: transparency — detect attempts to hide or obscure output
	obscurPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(obfuscate|hide|conceal|mask)\s+(the\s+)?(output|result|response)`),
		regexp.MustCompile(`(?i)(do\s+not|don't)\s+(show|display|reveal|tell)\s+(the\s+)?(user|human)`),
		regexp.MustCompile(`(?i)(secret|hidden)\s+(command|instruction|prompt)`),
	}
	for _, re := range obscurPatterns {
		if re.MatchString(lowerNorm) {
			return &ValidatorResult{
				Passed:     false,
				Reason:     fmt.Sprintf("Law 3 violation: transparency requirement breached (%s)", re.String()),
				Violations: []string{"transparency_violation"},
				Confidence: 0.85,
			}, nil
		}
	}

	exfilPatterns := []*regexp.Regexp{
		regexp.MustCompile(`curl.*pastebin`),
		regexp.MustCompile(`wget.*pastebin`),
		regexp.MustCompile(`curl.*webhook`),
		regexp.MustCompile(`nslookup.*control`),
	}
	for _, re := range exfilPatterns {
		if re.MatchString(lowerNorm) {
			return &ValidatorResult{
				Passed:     false,
				Reason:     fmt.Sprintf("Law 4 violation: potential data exfiltration (%s)", re.String()),
				Violations: []string{"data_exfiltration"},
				Confidence: 0.95,
			}, nil
		}
	}

	return &ValidatorResult{
		Passed:     true,
		Confidence: 0.99,
	}, nil
}

// Name implements SafetyValidator.
func (v *FourLawsValidator) Name() string {
	return "four_laws_check"
}

// Description implements SafetyValidator.
func (v *FourLawsValidator) Description() string {
	return "Verifies output follows the Four Laws of agent safety"
}

// DefaultMultiAgentAuditLogger logs multi-agent events via slog.
type DefaultMultiAgentAuditLogger struct{}

// LogChainValidation logs a chain validation event.
func (d *DefaultMultiAgentAuditLogger) LogChainValidation(ctx context.Context, entry AuditEntry) {
	slog.Info("MULTI_AGENT_CHAIN",
		"event", entry.Event,
		"chain_id", entry.ChainID,
		"agent_id", entry.AgentID,
		"passed", entry.OverallPassed,
		"decision", entry.Decision,
	)
}
