package guardrails

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Spec 05 (indirect-injection-provenance-records & provenance-policy-and-conformance)
// & aiggp-06 Phase 3:
//   - Append-only cryptographic provenance chains (`ProvenanceChainLink`,
//     `ProvenanceChain`): transformations (summarization, extraction, quoting,
//     tool post-processing) append a new link referencing prior links by their
//     cryptographic link hashes rather than overwriting origin metadata.
//   - Integrity verification (`VerifyProvenanceChain`): recomputes every link
//     hash and parent reference in order and names the exact broken or missing
//     link on truncation or tampering.
//   - Mixed-trust inheritance (`EffectiveChainTrust`): an action or derived
//     artifact informed by multiple chains inherits a trust level no higher
//     than the lowest-trust input that materially contributed.
//   - Ledger ancestry reconstruction (`ProvenanceLedger`, `ReconstructAncestry`):
//     given any mediated action ID recorded in the ledger, reconstructs and
//     verifies the complete ordered ancestry (origins, trust classes,
//     timestamps, and transformation sequence) of every contributing input.

// ProvenanceChainLink is one immutable step in an append-only provenance chain.
type ProvenanceChainLink struct {
	LinkID         string           `json:"link_id"`
	Index          int              `json:"index"`
	ParentHashes   []string         `json:"parent_hashes,omitempty"`
	Source         string           `json:"source"`
	SourcePath     string           `json:"source_path"`
	Transformation string           `json:"transformation"` // "ingestion", "summarization", "extraction", "quoting", "tool_post_processing", "synthesis"
	TrustLevel     SourceTrustLevel `json:"trust_level"`
	AgentID        string           `json:"agent_id"`
	ContentHash    string           `json:"content_hash"`
	Timestamp      time.Time        `json:"timestamp"`
	LinkHash       string           `json:"link_hash"`
}

// computeLinkHash deterministically computes the SHA-256 link digest over all
// immutable fields of a ProvenanceChainLink, binding it to its ParentHashes.
func computeLinkHash(l ProvenanceChainLink) string {
	parents := strings.Join(l.ParentHashes, ",")
	canonical := fmt.Sprintf(
		"v1|id=%s|idx=%d|parents=%s|src=%s|path=%s|tx=%s|trust=%s|agent=%s|content=%s|ts=%s",
		l.LinkID,
		l.Index,
		parents,
		l.Source,
		l.SourcePath,
		l.Transformation,
		l.TrustLevel,
		l.AgentID,
		l.ContentHash,
		l.Timestamp.UTC().Format(time.RFC3339Nano),
	)
	sum := sha256.Sum256([]byte(canonical))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// LeastTrustedLevel returns the lower-trust of two SourceTrustLevels.
// Ordering from least trusted to most trusted:
//
//	untrusted < unknown < trusted
func LeastTrustedLevel(a, b SourceTrustLevel) SourceTrustLevel {
	if trustRank(b) < trustRank(a) {
		return b
	}
	return a
}

func trustRank(t SourceTrustLevel) int {
	switch t {
	case TrustLevelUntrusted:
		return 0
	case TrustLevelUnknown:
		return 1
	case TrustLevelTrusted:
		return 2
	default:
		// Any unrecognized trust label is treated as untrusted (fail-closed).
		return 0
	}
}

// EffectiveChainTrust returns the effective trust level across one or more
// provenance links: never higher than the lowest-trust link in the set.
func EffectiveChainTrust(links []ProvenanceChainLink) SourceTrustLevel {
	if len(links) == 0 {
		return TrustLevelUnknown
	}
	eff := links[0].TrustLevel
	for _, l := range links[1:] {
		eff = LeastTrustedLevel(eff, l.TrustLevel)
	}
	return eff
}

// VerifyProvenanceChain verifies that a provenance chain is non-empty, starts
// at index 0 with an ingestion root (or valid parent links), has contiguous
// indices, has intact cryptographic LinkHashes on every link, and never
// elevates trust above its parent links.
//
// If any link has been stripped, reordered, or altered, VerifyProvenanceChain
// returns an error naming the exact broken link ID and index.
func VerifyProvenanceChain(links []ProvenanceChainLink) error {
	if len(links) == 0 {
		return fmt.Errorf("provenance chain is empty")
	}

	byHash := make(map[string]ProvenanceChainLink, len(links))
	for i, l := range links {
		if l.Index != i {
			return fmt.Errorf("broken provenance chain at link %q: expected index %d, got %d (chain truncated or reordered)",
				l.LinkID, i, l.Index)
		}
		expected := computeLinkHash(l)
		if l.LinkHash != expected {
			return fmt.Errorf("broken provenance chain at link %q (index %d): link_hash mismatch (expected %s, got %s)",
				l.LinkID, i, expected, l.LinkHash)
		}
		if i == 0 && len(l.ParentHashes) > 0 {
			return fmt.Errorf("broken provenance chain at root link %q (index 0): root link must not declare parent_hashes",
				l.LinkID)
		}
		if i > 0 && len(l.ParentHashes) == 0 {
			return fmt.Errorf("broken provenance chain at link %q (index %d): derived link missing parent_hashes",
				l.LinkID, i)
		}
		for _, ph := range l.ParentHashes {
			parent, ok := byHash[ph]
			if !ok {
				return fmt.Errorf("broken provenance chain at link %q (index %d): missing or stripped parent link %q",
					l.LinkID, i, ph)
			}
			if trustRank(l.TrustLevel) > trustRank(parent.TrustLevel) {
				return fmt.Errorf("broken provenance chain at link %q (index %d): unlawful trust elevation from %s to %s",
					l.LinkID, i, parent.TrustLevel, l.TrustLevel)
			}
		}
		byHash[l.LinkHash] = l
	}

	return nil
}

// MediatedActionRecord binds a mediated agent action (command, file edit, or
// tool call) in the ledger to the terminal link hashes of every provenance
// chain that contributed to the decision.
type MediatedActionRecord struct {
	ActionID         string           `json:"action_id"`
	ToolName         string           `json:"tool_name"`
	AgentID          string           `json:"agent_id"`
	InputChainHashes []string         `json:"input_chain_hashes"`
	EffectiveTrust   SourceTrustLevel `json:"effective_trust"`
	Decision         Action           `json:"decision"`
	RecordedAt       time.Time        `json:"recorded_at"`
}

// AncestryTrace is the reconstructed, verified provenance history for a
// mediated action queried by an auditor.
type AncestryTrace struct {
	Action           MediatedActionRecord    `json:"action"`
	InputChains      [][]ProvenanceChainLink `json:"input_chains"`
	EffectiveTrust   SourceTrustLevel        `json:"effective_trust"`
	Verified         bool                    `json:"verified"`
	VerificationNote string                  `json:"verification_note"`
}

// ProvenanceLedger is an append-only store of provenance chain links and
// mediated action records supporting full multi-hop ancestry reconstruction.
type ProvenanceLedger struct {
	mu      sync.RWMutex
	links   map[string]ProvenanceChainLink // keyed by LinkHash
	chains  map[string][]string            // head LinkHash -> ordered slice of LinkHashes from root to head
	actions map[string]MediatedActionRecord
	seq     int
}

// NewProvenanceLedger constructs an empty append-only provenance ledger.
func NewProvenanceLedger() *ProvenanceLedger {
	return &ProvenanceLedger{
		links:   make(map[string]ProvenanceChainLink),
		chains:  make(map[string][]string),
		actions: make(map[string]MediatedActionRecord),
	}
}

// RecordIngestion creates a root provenance chain link for newly ingested
// external content. Uninstrumented or empty source classes default to
// TrustLevelUnknown / "anonymous" and never to trusted.
func (l *ProvenanceLedger) RecordIngestion(content, source, sourcePath, agentID string, trust SourceTrustLevel) ProvenanceChainLink {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.seq++
	if strings.TrimSpace(source) == "" {
		source = "anonymous"
	}
	if trust == "" {
		trust = TrustLevelUnknown
	}
	if source == "anonymous" && trust == TrustLevelTrusted {
		trust = TrustLevelUntrusted
	}

	link := ProvenanceChainLink{
		LinkID:         fmt.Sprintf("prov-link-%04d", l.seq),
		Index:          0,
		Source:         source,
		SourcePath:     sourcePath,
		Transformation: "ingestion",
		TrustLevel:     trust,
		AgentID:        agentID,
		ContentHash:    hashText(content),
		Timestamp:      time.Now().UTC(),
	}
	link.LinkHash = computeLinkHash(link)

	l.links[link.LinkHash] = link
	l.chains[link.LinkHash] = []string{link.LinkHash}
	return link
}

// AppendTransformation appends a derived link (summarization, extraction,
// quoting, tool_post_processing, synthesis) to one or more existing parent
// chain heads without mutating or replacing prior links. The derived link
// inherits the least-trusted level across all parent chains.
func (l *ProvenanceLedger) AppendTransformation(parentHeadHashes []string, transformedContent, transformation, sourcePath, agentID string) (ProvenanceChainLink, error) {
	if len(parentHeadHashes) == 0 {
		return ProvenanceChainLink{}, fmt.Errorf("parent_head_hashes is required for derived transformation")
	}
	if strings.TrimSpace(transformation) == "" {
		return ProvenanceChainLink{}, fmt.Errorf("transformation type is required")
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	var combinedHistory []string
	seen := make(map[string]bool)
	effectiveTrust := TrustLevelTrusted

	for i, ph := range parentHeadHashes {
		parentChain, ok := l.chains[ph]
		if !ok {
			return ProvenanceChainLink{}, fmt.Errorf("unknown parent chain head hash %q", ph)
		}
		parentLink := l.links[ph]
		if i == 0 {
			effectiveTrust = parentLink.TrustLevel
		} else {
			effectiveTrust = LeastTrustedLevel(effectiveTrust, parentLink.TrustLevel)
		}
		for _, h := range parentChain {
			if !seen[h] {
				seen[h] = true
				combinedHistory = append(combinedHistory, h)
			}
		}
	}

	l.seq++
	link := ProvenanceChainLink{
		LinkID:         fmt.Sprintf("prov-link-%04d", l.seq),
		Index:          len(combinedHistory),
		ParentHashes:   append([]string(nil), parentHeadHashes...),
		Source:         "derived",
		SourcePath:     sourcePath,
		Transformation: transformation,
		TrustLevel:     effectiveTrust,
		AgentID:        agentID,
		ContentHash:    hashText(transformedContent),
		Timestamp:      time.Now().UTC(),
	}
	link.LinkHash = computeLinkHash(link)

	newChain := append(combinedHistory, link.LinkHash)
	l.links[link.LinkHash] = link
	l.chains[link.LinkHash] = newChain
	return link, nil
}

// RecordMediatedAction records a mediated tool/command decision along with the
// provenance chain heads that informed it, computing the effective trust as the
// lowest-trust contributing chain. If any referenced chain fails cryptographic
// verification, RecordMediatedAction refuses to record and returns an error.
func (l *ProvenanceLedger) RecordMediatedAction(actionID, toolName, agentID string, inputHeadHashes []string, decision Action) (MediatedActionRecord, error) {
	if strings.TrimSpace(actionID) == "" {
		return MediatedActionRecord{}, fmt.Errorf("action_id is required")
	}
	if len(inputHeadHashes) == 0 {
		return MediatedActionRecord{}, fmt.Errorf("at least one input provenance chain hash is required")
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	effective := TrustLevelTrusted
	for i, head := range inputHeadHashes {
		chainHashes, ok := l.chains[head]
		if !ok {
			return MediatedActionRecord{}, fmt.Errorf("unknown input provenance chain %q", head)
		}
		chainLinks := make([]ProvenanceChainLink, 0, len(chainHashes))
		for _, h := range chainHashes {
			chainLinks = append(chainLinks, l.links[h])
		}
		if err := VerifyProvenanceChain(chainLinks); err != nil {
			return MediatedActionRecord{}, fmt.Errorf("input chain %q failed verification: %w", head, err)
		}
		chainTrust := EffectiveChainTrust(chainLinks)
		if i == 0 {
			effective = chainTrust
		} else {
			effective = LeastTrustedLevel(effective, chainTrust)
		}
	}

	rec := MediatedActionRecord{
		ActionID:         actionID,
		ToolName:         toolName,
		AgentID:          agentID,
		InputChainHashes: append([]string(nil), inputHeadHashes...),
		EffectiveTrust:   effective,
		Decision:         decision,
		RecordedAt:       time.Now().UTC(),
	}
	l.actions[actionID] = rec
	return rec, nil
}

// ReconstructAncestry queries the ledger for a mediated action ID and returns
// the complete ordered provenance chain for every contributing input, re-running
// cryptographic chain verification over each chain.
func (l *ProvenanceLedger) ReconstructAncestry(actionID string) (*AncestryTrace, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	act, ok := l.actions[actionID]
	if !ok {
		return nil, fmt.Errorf("mediated action %q not found in provenance ledger", actionID)
	}

	trace := &AncestryTrace{
		Action:         act,
		EffectiveTrust: act.EffectiveTrust,
		Verified:       true,
	}

	for _, head := range act.InputChainHashes {
		hashes, ok := l.chains[head]
		if !ok {
			trace.Verified = false
			trace.VerificationNote = fmt.Sprintf("missing chain for head %q", head)
			return trace, fmt.Errorf("ancestry reconstruction failed: %s", trace.VerificationNote)
		}
		chain := make([]ProvenanceChainLink, 0, len(hashes))
		for _, h := range hashes {
			link, exists := l.links[h]
			if !exists {
				trace.Verified = false
				trace.VerificationNote = fmt.Sprintf("missing link %q in chain %q", h, head)
				return trace, fmt.Errorf("ancestry reconstruction failed: %s", trace.VerificationNote)
			}
			chain = append(chain, link)
		}
		if err := VerifyProvenanceChain(chain); err != nil {
			trace.Verified = false
			trace.VerificationNote = err.Error()
			return trace, err
		}
		trace.InputChains = append(trace.InputChains, chain)
	}

	trace.VerificationNote = fmt.Sprintf("verified %d contributing provenance chain(s)", len(trace.InputChains))
	return trace, nil
}
