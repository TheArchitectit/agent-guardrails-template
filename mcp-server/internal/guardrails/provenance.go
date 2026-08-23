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

	// Hash the sanitized+decoded content + source + source path + agentID for
	// caching. Including source and agentID prevents re-tagging the same path
	// with a different source/agent from returning stale cached provenance
	// (wrong .Source/.ReadBy).
	hash := hashText(decoded + "::" + source + "::" + sourcePath + "::" + agentID)

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

	// Cache the provenance — use configured TTL or default to 1 hour
	ttl := time.Hour
	if pt.cache != nil {
		_ = pt.cache.Set(ctx, hash, prov, ttl)
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
// Matching is case-insensitive: on case-insensitive filesystems (the common
// case for project config) this only broadens trust, and untrusted is the
// fail-safe side, so case folding is safe.
func matchPattern(pattern, path string) bool {
	pattern = strings.ToLower(pattern)
	path = strings.ToLower(path)

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
	// ** glob: prefix / ** / suffix  →  path must start with prefix, contain
	// at least one path segment between prefix and suffix (so ** never collapses
	// to zero segments), and end with suffix. Bare "**" matches any path.
	if strings.Contains(pattern, "**") {
		// Bare "**" matches everything.
		if pattern == "**" {
			return true
		}
		parts := strings.SplitN(pattern, "**", 2)
		prefix := strings.TrimSuffix(parts[0], "/")
		suffix := strings.TrimLeft(parts[1], "*/")
		if prefix != "" && !strings.HasPrefix(path, prefix) {
			return false
		}
		if suffix != "" && !strings.HasSuffix(path, suffix) {
			return false
		}
		// ** must match at least one path segment: the substring strictly
		// between prefix and suffix must contain a '/' and have non-empty
		// trimmed content, so ** cannot collapse to zero segments
		// (e.g. "a/**/c.md" must NOT match "a/c.md").
		between := path
		if prefix != "" {
			between = between[len(prefix):]
		}
		if suffix != "" {
			between = between[:len(between)-len(suffix)]
		}
		return strings.Contains(between, "/") && strings.Trim(between, "/") != ""
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
	// A trailing '*' (pattern ends with *) imposes a single-segment boundary:
	// the matched remainder must not span further '.' or '/' separators, so
	// "api.internal.*" matches "api.internal.foo" but NOT
	// "api.internal.evil.attacker.com".
	if strings.HasSuffix(pattern, "*") {
		tail := path[idx:]
		if strings.ContainsAny(tail, "./") {
			return false
		}
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

	// 1. Try base64 decode — only when explicitly marked. We require an explicit
	// label/marker (e.g. "base64:" prefix or a data-URI scheme) rather than
	// guessing on bare alphabet runs, which would rewrite ordinary tokens to garbage.
	if hasBase64Marker(content) {
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

// hasBase64Marker reports whether content carries an explicit base64 label or
// data-URI scheme, signalling intentional base64 encoding rather than an
// incidental alphabet run.
func hasBase64Marker(content string) bool {
	lower := strings.ToLower(content)
	return strings.Contains(lower, "base64:") ||
		strings.Contains(lower, "data:") ||
		strings.Contains(lower, ";base64,")
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
	runes := []rune(s)
	if len(runes) == 0 {
		return false
	}
	printable := 0
	for _, r := range runes {
		if r >= 32 && r < 127 || r == '\n' || r == '\r' || r == '\t' {
			printable++
		}
	}
	return float64(printable)/float64(len(runes)) > 0.8
}

// looksLikeROT13 checks if content might be ROT13-encoded using two signals:
// (1) a high proportion of known rotated English words (gur→the, naq→and,
//     gung→that, jvgu→with, sbk→fox, etc.), which letter-frequency analysis
//     misses because ROT13 preserves letter frequencies; and
// (2) improved bigram frequency after ROT13 decoding, a structural check that
//     resists single-letter-frequency games. Either signal alone is enough.
func looksLikeROT13(content string) bool {
	if len(content) < 10 {
		return false
	}
	// Only consider content that is predominantly letters
	letterCount := 0
	for _, r := range content {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			letterCount++
		}
	}
	if letterCount < 10 {
		return false
	}

	// Signal 1: known rotated-word density. These are common English words in
	// ROT13 form; seeing several in a short string is a strong indicator.
	rotatedWords := []string{
		"gur", "naq", "gung", "jvgu", "sbk", "oebja", "dhvpx", "bzcf",
		"nah", "ber", "lhe", "znx", "rire", "jvgu", "guvf", "gurl",
		"jung", "urer", "oybbq", "jbeyq", "ceboyrz", "hfr",
	}
	words := strings.Fields(strings.ToLower(content))
	if len(words) == 0 {
		return false
	}
	matchCount := 0
	for _, w := range words {
		// Strip trailing punctuation for matching.
		cleaned := strings.Trim(w, ".,;:!?")
		for _, rw := range rotatedWords {
			if cleaned == rw {
				matchCount++
				break
			}
		}
	}
	// At least 30% of words are known rotated forms, with an absolute minimum
	// of 2 matches (avoids false positives on 1-word coincidences).
	if matchCount >= 2 && float64(matchCount)/float64(len(words)) >= 0.30 {
		return true
	}

	// Signal 2: bigram frequency improvement after ROT13 decoding. English has
	// characteristic common bigrams (th, he, in, er, an, re, on, en); ROT13
	// shifts these to (gu, ur, va, re, na, er, ba, ra). A meaningful jump in
	// common-English-bigram count after decoding signals ROT13.
	commonBigrams := []string{
		"th", "he", "in", "er", "an", "re", "on", "en", "nd", "ti",
		"es", "or", "te", "of", "ed", "is", "it", "al", "ar", "st",
	}
	bigramSet := make(map[string]bool, len(commonBigrams))
	for _, b := range commonBigrams {
		bigramSet[b] = true
	}
	countBigrams := func(s string) int {
		lower := strings.ToLower(s)
		clean := make([]rune, 0, len(lower))
		for _, r := range lower {
			if r >= 'a' && r <= 'z' {
				clean = append(clean, r)
			}
		}
		n := 0
		for i := 0; i < len(clean)-1; i++ {
			if bigramSet[string(clean[i:i+2])] {
				n++
			}
		}
		return n
	}

	originalBigrams := countBigrams(content)
	decoded := tryROT13(content)
	decodedBigrams := countBigrams(decoded)

	// Decoded must show a clear bigram improvement (at least 50% more common
	// bigrams than the original, and at least 3).
	return decodedBigrams >= 3 && float64(decodedBigrams) > float64(originalBigrams)*1.5
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
