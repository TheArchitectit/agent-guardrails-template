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
			for _, pattern := range v.scopePatterns {
				if strings.Contains(lowerNorm, strings.ToLower(pattern)) {
					return &ValidatorResult{
						Passed:     false,
						Reason:     fmt.Sprintf("Law 2 violation: output exceeds declared scope (%s)", pattern),
						Violations: []string{"scope_violation"},
						Confidence: 0.9,
					}, nil
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
