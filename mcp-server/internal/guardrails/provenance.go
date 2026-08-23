package guardrails

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// SourceTrustLevel defines the level of trust associated with a content source.
type SourceTrustLevel string

const (
	TrustLevelTrusted   SourceTrustLevel = "trusted"
	TrustLevelUntrusted SourceTrustLevel = "untrusted"
	TrustLevelUnknown   SourceTrustLevel = "unknown"
)

// TrustAction defines the action to take when content is encountered.
type TrustAction string

const (
	ActionScanAndWarn   TrustAction = "scan_and_warn"
	ActionScanAndBlock  TrustAction = "scan_and_block"
)

// Provenance tracks the origin and trust level of a piece of content.
type Provenance struct {
	Source      string           `json:"source"`       // e.g., "file", "api", "user", "tool_output", "system"
	SourcePath  string           `json:"source_path"`  // e.g., file path, API URL
	TrustLevel  SourceTrustLevel `json:"trust_level"`  // trusted, untrusted, unknown
	ReadBy      string           `json:"read_by"`      // agent ID that read this content
	Timestamp   time.Time        `json:"timestamp"`    // when it was read
	Hash        string           `json:"hash"`         // content hash for dedup/cache
}

// TrustPolicy defines how to treat content from a specific source pattern.
type TrustPolicy struct {
	SourcePattern string          `yaml:"source_pattern"`
	TrustLevel    SourceTrustLevel `yaml:"trust_level"`
	Action        TrustAction     `yaml:"action"`
}

// ProvenanceTracker manages the provenance of content entering the agent's context.
type ProvenanceTracker struct {
	policies []TrustPolicy
	cache    ProvenanceCache
	mu       sync.RWMutex
}

// ProvenanceCache defines the interface for storing and retrieving provenance data.
type ProvenanceCache interface {
	Get(ctx context.Context, hash string) (*Provenance, bool)
	Set(ctx context.Context, hash string, prov *Provenance, ttl time.Duration) error
}

// NewProvenanceTracker creates a new provenance tracker with specified policies.
func NewProvenanceTracker(policies []TrustPolicy, cache ProvenanceCache) *ProvenanceTracker {
	return &ProvenanceTracker{
		policies: policies,
		cache:    cache,
	}
}

// TagContent analyzes a source and assigns provenance to the content.
func (pt *ProvenanceTracker) TagContent(ctx context.Context, content string, source string, sourcePath string, agentID string) (*Provenance, error) {
	hash := hashText(content)

	// Check cache first
	if pt.cache != nil {
		if prov, ok := pt.cache.Get(ctx, hash); ok {
			return prov, nil
		}
	}

	// Determine trust level based on policies
	trustLevel, _ := pt.resolveTrust(sourcePath)

	prov := &Provenance{
		Source:     source,
		SourcePath: sourcePath,
		TrustLevel: trustLevel,
		ReadBy:     agentID,
		Timestamp:  time.Now().UTC(),
		Hash:       hash,
	}

	// In a real implementation, the action (e.g., scan_and_block)
	// would be handled by the coordinator calling the pipeline.
	// Here we just store the determined provenance.

	if pt.cache != nil {
		// Use default TTL of 1 hour as per spec
		_ = pt.cache.Set(ctx, hash, prov, time.Hour)
	}

	return prov, nil
}

// resolveTrust matches a source path against registered policies.
func (pt *ProvenanceTracker) resolveTrust(path string) (SourceTrustLevel, TrustAction) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	for _, p := range pt.policies {
		if matchPattern(p.SourcePattern, path) {
			return p.TrustLevel, p.Action
		}
	}

	return TrustLevelUnknown, ActionScanAndWarn
}

// matchPattern is a helper to check if a path matches a glob-like pattern.
func matchPattern(pattern, path string) bool {
	// Wildcard matches everything
	if pattern == "*" {
		return true
	}
	// Exact match
	if pattern == path {
		return true
	}
	// Substring match for domain patterns (e.g. "github.com" in "https://github.com/user/repo")
	if !strings.Contains(pattern, "*") && strings.Contains(path, pattern) {
		return true
	}
	// Glob with **: docs/**/*.md matches docs/foo/bar.md
	if strings.Contains(pattern, "**") {
		parts := strings.SplitN(pattern, "**", 2)
		prefix := strings.TrimSuffix(parts[0], "/")
		suffix := strings.TrimPrefix(parts[1], "/")
		if prefix != "" && !strings.HasPrefix(path, prefix) {
			return false
		}
		if suffix != "" {
			// Match suffix with simple extension check
			if suffix[0] == '.' {
				ext := suffix
				if len(path) >= len(ext) && path[len(path)-len(ext):] == ext {
					return true
				}
			}
			return strings.HasSuffix(path, suffix)
		}
		return true
	}
	// Basic extension matching (*.json)
	if len(pattern) > 2 && pattern[0] == '*' && pattern[1] == '.' {
		ext := pattern[2:]
		lenPath := len(path)
		lenExt := len(ext)
		if lenPath >= lenExt && path[lenPath-lenExt:] == ext {
			return true
		}
	}
	return false
}

// SanitizeContent applies a cleanup pipeline to untrusted content.
func SanitizeContent(content string) string {
	// 1. Strip Control Characters (Spec 4.2)
	// Zero-width characters: U+200B-U+200F, U+2028-U+2029, U+2060-U+2064
	// Bidi overrides: U+202A-U+202E, U+2066-U+2069
	// Invisible format chars: U+FEFF, U+00AD

	runes := []rune(content)
	var result []rune

	for _, r := range runes {
		if (r >= 0x200B && r <= 0x200F) ||
		   (r >= 0x2028 && r <= 0x2029) ||
		   (r >= 0x2060 && r <= 0x2064) ||
		   (r >= 0x202A && r <= 0x202E) ||
		   (r >= 0x2066 && r <= 0x2069) ||
		   (r == 0xFEFF) || (r == 0x00AD) {
			continue
		}
		result = append(result, r)
	}

	return string(result)
}

// WrapWithProvenance adds markers to content based on its trust level.
func WrapWithProvenance(content string, prov *Provenance) string {
	if prov.TrustLevel == TrustLevelTrusted {
		return content
	}

	marker := fmt.Sprintf("[Content from %s source: %s] ", prov.TrustLevel, prov.SourcePath)
	if prov.TrustLevel == TrustLevelUntrusted {
		marker = fmt.Sprintf("[UNTRUSTED] %s", marker)
	}

	return marker + content
}
