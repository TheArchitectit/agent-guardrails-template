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

// matchPattern checks if a path matches a glob-like pattern.
// Supported: exact match, * (any chars within one segment), ** (any path depth),
// basename match for simple filenames (no wildcards, no slashes).
func matchPattern(pattern, path string) bool {
	// Wildcard matches everything
	if pattern == "*" {
		return true
	}
	// Exact match
	if pattern == path {
		return true
	}
	// No wildcards: smart matching based on pattern shape
	if !strings.Contains(pattern, "*") {
		if strings.Contains(pattern, "/") {
			// Path/URL pattern — substring match
			return strings.Contains(path, pattern)
		}
		if strings.Contains(pattern, ".") || strings.Contains(pattern, ":") {
			// Looks like a hostname/domain — substring match
			// (github.com matches https://github.com/..., api.internal matches api.internal.example.com)
			return strings.Contains(path, pattern)
		}
		// Simple name without dots/slashes — basename match only
		// (prevents "localhost" matching "evil-localhost.xyz")
		basename := path
		if i := strings.LastIndex(path, "/"); i >= 0 {
			basename = path[i+1:]
		}
		return pattern == basename
	}
	// ** glob: prefix / ** / suffix  →  path must start with prefix and end with suffix
	// The ** itself is the recursive wildcard; remaining suffix after it is a literal tail.
	if strings.Contains(pattern, "**") {
		parts := strings.SplitN(pattern, "**", 2)
		prefix := strings.TrimSuffix(parts[0], "/")
		suffix := strings.TrimLeft(parts[1], "*/")
		if prefix != "" && !strings.HasPrefix(path, prefix) {
			return false
		}
		if suffix != "" && !strings.HasSuffix(path, suffix) {
			return false
		}
		return true
	}
	// Single * glob: split on *, each segment must appear in order
	// e.g. "api.internal.*" matches "api.internal.example.com"
	// e.g. "*.json" matches "config.json"
	return globMatch(pattern, path)
}

// globMatch handles single-* glob patterns by splitting on * and checking
// that each literal segment appears in the path in order.
func globMatch(pattern, path string) bool {
	segments := strings.Split(pattern, "*")
	if len(segments) == 1 {
		return pattern == path
	}
	idx := 0
	for i, seg := range segments {
		if seg == "" {
			continue
		}
		pos := strings.Index(path[idx:], seg)
		if pos < 0 {
			return false
		}
		// First segment must match at start; last segment must match at end
		if i == 0 && pos != 0 {
			return false
		}
		if i == len(segments)-1 && idx+pos+len(seg) != len(path) {
			return false
		}
		idx += pos + len(seg)
	}
	return true
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
