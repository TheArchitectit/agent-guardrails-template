package guardrails

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// ExternalScanResult is the verdict for a piece of content that enters the
// agent's context from outside it (Spec 05 Sec. 3.1).
//
// It is deliberately not an InjectionResult. InjectionResult answers "does this
// text contain an injection pattern". This answers "should the agent be allowed
// to consume this text at all, given where it came from", which needs the
// trust policy and the sanitisation step even when no pattern matched.
type ExternalScanResult struct {
	ProvenanceID string           `json:"provenance_id"`
	Safe         bool             `json:"safe"`
	Blocked      bool             `json:"blocked"`
	Action       Action           `json:"action"`
	TrustLevel   SourceTrustLevel `json:"trust_level"`
	Source       string           `json:"source"`
	SourcePath   string           `json:"source_path"`
	// Sanitized is the text the agent should consume: control/invisible
	// characters stripped and any labelled obfuscation decoded. It is empty
	// only when the content itself was empty.
	Sanitized  string           `json:"sanitized,omitempty"`
	Obfuscated bool             `json:"obfuscated"`
	Injection  *InjectionResult `json:"injection,omitempty"`
	Reason     string           `json:"reason,omitempty"`
	Provenance *Provenance      `json:"provenance,omitempty"`
}

// maxProvenanceRecords bounds the in-process provenance registry so a caller
// cannot grow it without limit. It is a working set, not an archive: the
// authoritative record of what was scanned belongs in the audit trail.
const maxProvenanceRecords = 4096

// provenanceRecord is one entry in the registry, with its own expiry so an
// expired entry is never served as a fresh verdict.
type provenanceRecord struct {
	prov      *Provenance
	scan      *ExternalScanResult
	expiresAt time.Time
}

// provenanceRegistry is the lookup behind guardrail_check_provenance. Kept
// separate from ProvenanceCache (which is keyed by content hash and is a
// dedup cache) because this one is keyed by the id handed to a caller.
type provenanceRegistry struct {
	mu      sync.RWMutex
	entries map[string]provenanceRecord
	ttl     time.Duration
}

func newProvenanceRegistry(ttl time.Duration) *provenanceRegistry {
	if ttl <= 0 {
		ttl = time.Hour
	}
	return &provenanceRegistry{entries: make(map[string]provenanceRecord), ttl: ttl}
}

func (r *provenanceRegistry) put(id string, prov *Provenance, scan *ExternalScanResult, now time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.entries) >= maxProvenanceRecords {
		// Drop expired entries first; if none, refuse the write rather than
		// evicting a live verdict that a caller may still be holding.
		for k, v := range r.entries {
			if now.After(v.expiresAt) {
				delete(r.entries, k)
			}
		}
		if len(r.entries) >= maxProvenanceRecords {
			return
		}
	}
	r.entries[id] = provenanceRecord{prov: prov, scan: scan, expiresAt: now.Add(r.ttl)}
}

func (r *provenanceRegistry) get(id string, now time.Time) (provenanceRecord, bool) {
	r.mu.RLock()
	rec, ok := r.entries[id]
	r.mu.RUnlock()
	if !ok {
		return provenanceRecord{}, false
	}
	if now.After(rec.expiresAt) {
		return provenanceRecord{}, false
	}
	return rec, true
}

func (r *provenanceRegistry) len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.entries)
}

// ensureRegistry lazily creates the registry. NewEngine does it eagerly; the
// lazy path exists so an Engine built as a struct literal in a test still
// records provenance instead of panicking on a nil map.
func (e *Engine) ensureRegistry() *provenanceRegistry {
	e.provMu.Lock()
	defer e.provMu.Unlock()
	if e.provRecords == nil {
		ttl := time.Hour
		if e.config != nil && e.config.Provenance != nil {
			ttl = e.config.Provenance.CacheDuration()
		}
		e.provRecords = newProvenanceRegistry(ttl)
	}
	return e.provRecords
}

// ScanExternalContent runs the Spec 05 pipeline over content that came from
// outside the agent's own instruction stream: sanitise, decode labelled
// obfuscation, resolve the source trust policy, and — for anything not
// explicitly trusted — scan the SANITISED text for injection.
//
// Scanning the sanitised/decoded text is the whole point. Until this existed,
// the only injection scan in the tree ran on raw text, so L1's patterns were
// evaded by inserting a single zero-width character inside a keyword:
// "ig\u200Bnore all previous instructions" does not match `ignore\s+`, and
// stripping it happens upstream of the matcher or not at all.
//
// Fail-closed: no tracker (provenance disabled) or a disabled engine is an
// error, not an allow. An untracked source cannot be judged trusted.
func (e *Engine) ScanExternalContent(ctx context.Context, content, source, sourcePath, agentID string) (*ExternalScanResult, error) {
	if content == "" {
		return nil, fmt.Errorf("content is required")
	}
	if source == "" {
		return nil, fmt.Errorf("source_type is required")
	}
	if e.config != nil && !e.config.Enabled {
		return nil, fmt.Errorf("guardrails engine is disabled; external content cannot be vetted")
	}
	if e.tracker == nil {
		return nil, fmt.Errorf("provenance tracking is not configured; external content cannot be vetted")
	}

	prov, err := e.tracker.TagContent(ctx, content, source, sourcePath, agentID)
	if err != nil {
		return nil, fmt.Errorf("provenance tagging failed: %w", err)
	}

	// The trust decision governs whether a scan is required, so resolve it
	// from the same policy set the tracker used rather than re-deriving it.
	trustLevel, action := e.tracker.resolveTrust(sourcePath)

	sanitized := SanitizeContent(content)
	decoded, obfuscated := DecodeObfuscation(sanitized)

	result := &ExternalScanResult{
		Safe:         true,
		Action:       action,
		TrustLevel:   trustLevel,
		Source:       source,
		SourcePath:   sourcePath,
		Sanitized:    decoded,
		Obfuscated:   obfuscated,
		Provenance:   prov,
		ProvenanceID: provenanceID(prov.Hash),
	}

	// An explicitly trusted source is consumed without a deep scan (Spec 05
	// Sec. 2.3: action "allow"). It is still sanitised — invisible characters
	// are never meaningful in trusted prose either.
	if action == ActionAllow {
		result.Reason = "trusted source; injection scan skipped by policy"
		e.remember(result, prov)
		return result, nil
	}

	// Untrusted or unknown: scan what the agent would actually read.
	if e.pipeline == nil {
		return nil, fmt.Errorf("injection pipeline is not initialized; external content cannot be vetted")
	}
	inj := e.pipeline.Detect(ctx, decoded, Source(source))
	result.Injection = &inj

	if inj.Safe {
		e.remember(result, prov)
		return result, nil
	}

	result.Safe = false
	switch action {
	case ActionScanAndBlock:
		result.Blocked = true
		result.Reason = fmt.Sprintf("injection detected in %s content at %s: %s",
			trustLevel, inj.Layer, inj.Reason)
	default: // ActionScanAndWarn and any unrecognised action
		// An unrecognised action is not silently promoted to allow: the
		// content is reported unsafe and the caller is told to warn.
		result.Reason = fmt.Sprintf("injection detected in %s content at %s (warn): %s",
			trustLevel, inj.Layer, inj.Reason)
	}

	e.remember(result, prov)
	return result, nil
}

// remember stores the scan for guardrail_check_provenance. A full registry is
// not an error: the verdict returned to the caller is still correct, only the
// later lookup is unavailable.
func (e *Engine) remember(result *ExternalScanResult, prov *Provenance) {
	e.ensureRegistry().put(result.ProvenanceID, prov, result, time.Now())
}

// LookupProvenance returns a previously recorded scan by provenance id.
func (e *Engine) LookupProvenance(id string) (*ExternalScanResult, bool) {
	if id == "" {
		return nil, false
	}
	rec, ok := e.ensureRegistry().get(id, time.Now())
	if !ok {
		return nil, false
	}
	return rec.scan, true
}

// ProvenanceRecordCount reports how many scans the registry currently holds.
// Used by tests to prove the registry is actually written.
func (e *Engine) ProvenanceRecordCount() int {
	return e.ensureRegistry().len()
}

// MarkContent applies the provenance marker the spec requires the agent to see
// (Spec 05 Sec. 2.4): untrusted content is wrapped so an instruction inside it
// cannot present itself as a system directive.
func MarkContent(content string, prov *Provenance, id string) string {
	if prov == nil {
		return "[UNTRUSTED SOURCE: unknown]\n" + content
	}
	label := string(prov.TrustLevel)
	if label == "" {
		label = string(TrustLevelUnknown)
	}
	source := prov.SourcePath
	if source == "" {
		source = prov.Source
	}
	if source == "" {
		source = "unspecified"
	}
	header := fmt.Sprintf("[UNTRUSTED SOURCE: %s | id=%s]", source, id)
	if prov.TrustLevel == TrustLevelTrusted {
		header = fmt.Sprintf("[TRUSTED SOURCE: %s | id=%s]", source, id)
	}
	return header + "\n" + content + "\n[END " + strings.ToUpper(label) + " CONTENT]"
}

// provenanceID derives the caller-visible id from the content hash. It is
// truncated because it is an opaque handle, not a security token: the hash is
// already over sanitised content and carries no secret.
func provenanceID(hash string) string {
	h := strings.TrimPrefix(hash, "sha256:")
	if len(h) > 16 {
		h = h[:16]
	}
	if h == "" {
		return "prov_unknown"
	}
	return "prov_" + h
}
