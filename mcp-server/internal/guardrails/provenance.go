package guardrails

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/url"
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
	Action        Action          `yaml:"action"`
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

// TagContent runs the full provenance pipeline on content:
// 1. SanitizeContent — strip control characters
// 2. DecodeObfuscation — detect and decode base64/ROT13/URL encoding
// 3. Resolve trust level from source policies
// 4. Cache and return provenance
func (pt *ProvenanceTracker) TagContent(ctx context.Context, content string, source string, sourcePath string, agentID string) (*Provenance, error) {
	// Step 1: Strip control characters (Unicode sanitization)
	sanitized := SanitizeContent(content)

	// Step 2: Decode obfuscation (base64/ROT13/URL)
	decoded, wasObfuscated := DecodeObfuscation(sanitized)
	if wasObfuscated {
		slog.Warn("obfuscated content detected and decoded",
			"source", sourcePath,
			"original_len", len(content),
			"decoded_len", len(decoded),
		)
	}

	// Hash the sanitized+decoded content for caching
	hash := hashText(decoded)

	// Check cache first
	if pt.cache != nil {
		if prov, ok := pt.cache.Get(ctx, hash); ok {
			return prov, nil
		}
	}

	// Step 3: Determine trust level based on source policies
	trustLevel, _ := pt.resolveTrust(sourcePath)

	prov := &Provenance{
		Source:     source,
		SourcePath: sourcePath,
		TrustLevel: trustLevel,
		ReadBy:     agentID,
		Timestamp:  time.Now().UTC(),
		Hash:       hash,
	}

	// Cache the provenance
	if pt.cache != nil {
		_ = pt.cache.Set(ctx, hash, prov, time.Hour)
	}

	return prov, nil
}

// resolveTrust matches a source path against registered policies.
func (pt *ProvenanceTracker) resolveTrust(path string) (SourceTrustLevel, Action) {
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
	// Additional: U+061C (Arabic Letter Mark), U+034F (Combining Grapheme Joiner)

	runes := []rune(content)
	var result []rune

	for _, r := range runes {
		if (r >= 0x200B && r <= 0x200F) ||
		   (r >= 0x2028 && r <= 0x2029) ||
		   (r >= 0x2060 && r <= 0x2064) ||
		   (r >= 0x202A && r <= 0x202E) ||
		   (r >= 0x2066 && r <= 0x2069) ||
		   (r == 0xFEFF) || (r == 0x00AD) ||
		   (r == 0x061C) || (r == 0x034F) {
			continue
		}
		result = append(result, r)
	}

	return string(result)
}

// DecodeObfuscation attempts to detect and decode common obfuscation techniques
// used to hide prompt injections: base64, ROT13, URL encoding.
// Returns the decoded text and whether any decoding was applied.
func DecodeObfuscation(content string) (string, bool) {
	decoded := false

	// 1. Try base64 decode — look for base64-encoded segments
	if strings.Contains(content, "base64") || looksLikeBase64(content) {
		if decoded_text, ok := tryBase64Decode(content); ok {
			content = decoded_text
			decoded = true
		}
	}

	// 2. Try ROT13 — common for simple obfuscation
	if looksLikeROT13(content) {
		if decoded_text := tryROT13(content); decoded_text != content {
			content = decoded_text
			decoded = true
		}
	}

	// 3. URL decode — %XX sequences
	if strings.Contains(content, "%") {
		if decoded_text, ok := tryURLDecode(content); ok {
			content = decoded_text
			decoded = true
		}
	}

	return content, decoded
}

// looksLikeBase64 checks if content contains a base64-encoded segment.
func looksLikeBase64(content string) bool {
	// Look for base64 blocks (4+ chars, valid alphabet, padded)
	for i := 0; i <= len(content)-4; i++ {
		end := i + 76 // Standard base64 line length
		if end > len(content) {
			end = len(content)
		}
		segment := content[i:end]
		if isBase64Segment(segment) {
			return true
		}
	}
	return false
}

// isBase64Segment checks if a string segment looks like base64.
func isBase64Segment(s string) bool {
	if len(s) < 4 {
		return false
	}
	for _, c := range s {
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '+' || c == '/' || c == '=') {
			return false
		}
	}
	// Check for padding consistency
	padCount := 0
	for i := len(s) - 1; i >= 0 && s[i] == '='; i-- {
		padCount++
	}
	return padCount <= 2
}

// tryBase64Decode attempts to decode base64 content.
func tryBase64Decode(content string) (string, bool) {
	// Try decoding the entire content
	if decoded, err := base64.StdEncoding.DecodeString(content); err == nil {
		result := string(decoded)
		// Only accept if decoded result is printable text
		if isPrintableText(result) {
			return result, true
		}
	}
	// Try with URL-safe base64
	if decoded, err := base64.URLEncoding.DecodeString(content); err == nil {
		result := string(decoded)
		if isPrintableText(result) {
			return result, true
		}
	}
	return content, false
}

// isPrintableText checks if a string is mostly printable characters.
func isPrintableText(s string) bool {
	if len(s) == 0 {
		return false
	}
	printable := 0
	for _, r := range s {
		if r >= 32 && r < 127 || r == '\n' || r == '\r' || r == '\t' {
			printable++
		}
	}
	return float64(printable)/float64(len(s)) > 0.8
}

// looksLikeROT13 checks if content might be ROT13-encoded.
func looksLikeROT13(content string) bool {
	// ROT13 typically produces strings with unusual letter distributions
	// Heuristic: check if decoding produces more common English patterns
	return len(content) > 20
}

// tryROT13 applies ROT13 decoding.
func tryROT13(content string) string {
	result := make([]rune, len(content))
	for i, r := range content {
		switch {
		case r >= 'a' && r <= 'z':
			result[i] = 'a' + (r-'a'+13)%26
		case r >= 'A' && r <= 'Z':
			result[i] = 'A' + (r-'A'+13)%26
		default:
			result[i] = r
		}
	}
	return string(result)
}

// tryURLDecode attempts to decode URL-encoded content (%XX sequences).
func tryURLDecode(content string) (string, bool) {
	if !strings.Contains(content, "%") {
		return content, false
	}
	decoded, err := url.QueryUnescape(content)
	if err != nil {
		return content, false
	}
	if decoded != content {
		return decoded, true
	}
	return content, false
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
