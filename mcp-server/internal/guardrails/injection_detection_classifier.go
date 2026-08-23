package guardrails

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"os"
	"strings"
)

// NoOpClassifier is a classifier backend that always returns safe.
type NoOpClassifier struct{}

// Name implements InjectionClassifier.
func (n *NoOpClassifier) Name() string { return "noop" }

// Classify implements InjectionClassifier.
func (n *NoOpClassifier) Classify(_ context.Context, _ string) (bool, float64, []InjectionCategory, error) {
	return true, 0.0, nil, nil
}

// Available implements InjectionClassifier.
func (n *NoOpClassifier) Available(_ context.Context) bool { return true }

// hashText returns a SHA-256 hash of the text for privacy-preserving audit logs.
func hashText(text string) string {
	h := sha256.Sum256([]byte(text))
	return "sha256:" + hex.EncodeToString(h[:])
}

func categoriesToStrings(cats []InjectionCategory) []string {
	result := make([]string, len(cats))
	for i, c := range cats {
		result[i] = string(c)
	}
	return result
}

// categorizePattern attempts to categorize a matched pattern.
func categorizePattern(pattern string) InjectionCategory {
	p := strings.ToLower(pattern)

	if (strings.Contains(p, "print") || strings.Contains(p, "echo") || strings.Contains(p, "cat")) &&
		(strings.Contains(p, "prompt") || strings.Contains(p, "secret") || strings.Contains(p, "key") || strings.Contains(p, "token")) {
		return CategoryDataExfiltration
	}

	if strings.Contains(p, "base64") || strings.Contains(p, "rot13") || strings.Contains(p, "unicode") || strings.Contains(p, "\\u[0-9") {
		return CategoryEncodingBypass
	}

	if strings.Contains(p, "system prompt") || strings.Contains(p, "system message") || strings.Contains(p, "initial prompt") {
		return CategoryContextManipulation
	}

	if strings.Contains(p, "you are now") || strings.Contains(p, "act as") || strings.Contains(p, "pretend") || strings.Contains(p, "roleplay") {
		return CategoryRolePlay
	}

	if strings.Contains(p, "sudo") || strings.Contains(p, "chmod") || strings.Contains(p, " privilege") {
		return CategoryPrivilegeEscalation
	}

	if strings.Contains(p, "ignore") || strings.Contains(p, "disregard") || strings.Contains(p, "override") || strings.Contains(p, "forget") || strings.Contains(p, "bypass") {
		return CategoryDirectiveOverride
	}

	return CategoryDirectiveOverride
}

// DefaultAuditLogger implements AuditLogger using slog.
type DefaultAuditLogger struct{}

// LogInjection logs an injection detection event.
func (d *DefaultAuditLogger) LogInjection(_ context.Context, event AuditEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		slog.Error("Failed to marshal audit event", "error", err)
		return
	}
	slog.Info("INJECTION_AUDIT", "event", string(data))
}

// Compile time interface checks
var _ InjectionClassifier = (*NoOpClassifier)(nil)
var _ AuditLogger = (*DefaultAuditLogger)(nil)

// ReadBlocklistFile reads a blocklist file and returns non-comment lines.
func ReadBlocklistFile(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var lines []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lines = append(lines, line)
	}
	return lines, nil
}
