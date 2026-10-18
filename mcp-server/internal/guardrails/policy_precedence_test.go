package guardrails

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Spec 07 R07-4.2 — precedence, composition and the exception ledger.
//
// Every case below drives the REAL loader over REAL pack files with a REAL lock
// record. A test that called resolvePolicyPrecedence directly would prove the
// algorithm and say nothing about whether a server can reach it — which is the
// failure this whole program keeps finding: complete library, no request path.

// precedenceFixture writes packs, an optional exception ledger, and a lock that
// is consistent with whatever was written, so a refusal in a case is the rule
// under test rather than an incidental lock mismatch.
func precedenceFixture(t *testing.T, packs map[string]string, exceptions string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range packs {
		writePack(t, dir, name, body)
	}
	if exceptions != "" {
		if err := writeRawFile(dir, exceptionLedgerName, exceptions); err != nil {
			t.Fatalf("write exceptions: %v", err)
		}
	}
	writeLockFor(t, dir, "")
	return dir
}

// writeRawFile exists because writePackRaw appends ".pack.json"; a ledger file
// needs its own name.
func writeRawFile(dir, name, body string) error {
	return os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600)
}

func nowAt(s string) time.Time {
	ts, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return ts
}

const corePack = `{"pack":"core/base","version":"1.0.0","tier":"core","rules":[{"id":"L1","law":"core","enforcement":"blocking","tool":"guardrail_validate_scope","on_unknown":"stop","evidence":"unit"}]}`

// TestPrecedence_LowerTierCannotWeakenCoreWithoutAnException is R07-4.2.1 in its
// plainest form: a domain pack lowering a core rule from blocking to advisory
// used to be impossible to express at all — and once expressible, it must be
// refused unless a reviewed exception names it.
func TestPrecedence_LowerTierCannotWeakenCoreWithoutAnException(t *testing.T) {
	dir := precedenceFixture(t, map[string]string{
		"core/base":    corePack,
		"domain/local": `{"pack":"domain/local","version":"1.0.0","tier":"domain","rules":[{"id":"L1","law":"core","enforcement":"advisory","tool":"guardrail_validate_scope","on_unknown":"stop"}]}`,
	}, "")

	_, err := LoadPolicySet(dir, nil)
	if err == nil {
		t.Fatal("a domain pack lowered a core rule with no exception and resolution succeeded")
	}
	if !strings.Contains(err.Error(), "no allowlisted exception") || !strings.Contains(err.Error(), `"blocking" to "advisory"`) {
		t.Fatalf("error does not name the weakening and the missing exception: %v", err)
	}
}

// TestPrecedence_RetargetingACoreRuleNeedsAnException: dropping a core rule's
// tool coverage is a weakening even at equal severity — the rule stops
// governing the tool it was written for.
func TestPrecedence_RetargetingACoreRuleNeedsAnException(t *testing.T) {
	dir := precedenceFixture(t, map[string]string{
		"core/base":    corePack,
		"domain/local": `{"pack":"domain/local","version":"1.0.0","tier":"domain","rules":[{"id":"L1","law":"core","enforcement":"blocking","tool":"guardrail_validate_commit","on_unknown":"stop","evidence":"moved"}]}`,
	}, "")

	_, err := LoadPolicySet(dir, nil)
	if err == nil {
		t.Fatal("a domain pack retargeted a core rule's tool with no exception and resolution succeeded")
	}
	if !strings.Contains(err.Error(), "retargeted") {
		t.Fatalf("error does not name the retargeting: %v", err)
	}
}

// TestPrecedence_HigherTierMayStrengthenWithoutAnException: the requirement only
// restricts weakening. A domain pack raising advisory to blocking must resolve,
// or the control would be unusable in the only direction that is safe.
func TestPrecedence_HigherTierMayStrengthenWithoutAnException(t *testing.T) {
	dir := precedenceFixture(t, map[string]string{
		"core/base":    `{"pack":"core/base","version":"1.0.0","tier":"core","rules":[{"id":"L1","law":"core","enforcement":"advisory","tool":"guardrail_validate_scope","on_unknown":"stop"}]}`,
		"domain/local": `{"pack":"domain/local","version":"1.0.0","tier":"domain","rules":[{"id":"L1","law":"core","enforcement":"blocking","tool":"guardrail_validate_scope","on_unknown":"stop","evidence":"raised"}]}`,
	}, "")

	set, err := LoadPolicySet(dir, nil)
	if err != nil {
		t.Fatalf("strengthening a core rule must resolve: %v", err)
	}
	apps := set.Applications()
	if len(apps) != 1 {
		t.Fatalf("one rule id resolved to %d applications", len(apps))
	}
	if apps[0].Effective != "blocking" || apps[0].Base != "advisory" {
		t.Fatalf("application = %+v, want base advisory and effective blocking", apps[0])
	}
	if apps[0].Weakened {
		t.Fatal("a strengthened rule was reported as weakened")
	}
	if got := set.Rules()[0].Enforcement; got != "blocking" {
		t.Fatalf("effective rule enforcement = %q, want blocking", got)
	}
}

// TestPrecedence_EqualTierConflictIsRefusedNotOrdered: two packs of equal
// standing disagreeing about one control have no deterministic winner. Picking
// one by file name would make the outcome depend on spelling.
func TestPrecedence_EqualTierConflictIsRefusedNotOrdered(t *testing.T) {
	dir := precedenceFixture(t, map[string]string{
		"core/alpha": `{"pack":"core/alpha","version":"1.0.0","tier":"core","rules":[{"id":"L1","law":"a","enforcement":"advisory","tool":"guardrail_validate_scope","on_unknown":"stop"}]}`,
		"core/beta":  `{"pack":"core/beta","version":"1.0.0","tier":"core","rules":[{"id":"L1","law":"b","enforcement":"blocking","tool":"guardrail_validate_scope","on_unknown":"stop","evidence":"x"}]}`,
	}, "")

	_, err := LoadPolicySet(dir, nil)
	if err == nil {
		t.Fatal("an equal-tier duplicate rule id resolved")
	}
	if !strings.Contains(err.Error(), "equal tiers have no deterministic precedence") {
		t.Fatalf("error does not name the ambiguity: %v", err)
	}
}

// TestPrecedence_DeclaredTierMustMatchThePath: precedence must be visible in the
// declaration, and the declaration must not be able to contradict the filing.
func TestPrecedence_DeclaredTierMustMatchThePath(t *testing.T) {
	dir := precedenceFixture(t, map[string]string{
		"domain/local": `{"pack":"domain/local","version":"1.0.0","tier":"core","rules":[{"id":"L1","law":"x","enforcement":"advisory","tool":"guardrail_validate_scope","on_unknown":"stop"}]}`,
	}, "")

	_, err := LoadPolicySet(dir, nil)
	if err == nil {
		t.Fatal("a domain pack declaring tier core was accepted")
	}
	if !strings.Contains(err.Error(), "its path implies") {
		t.Fatalf("error does not name the path disagreement: %v", err)
	}
}

// TestPrecedence_UndeclaredTierIsRefused: an absent tier must not default to
// core (which would grant override authority) or to overlay (which would accept
// an unchecked weakening).
func TestPrecedence_UndeclaredTierIsRefused(t *testing.T) {
	dir := precedenceFixture(t, map[string]string{
		"core/base": `{"pack":"core/base","version":"1.0.0","rules":[{"id":"L1","law":"x","enforcement":"advisory","tool":"guardrail_validate_scope","on_unknown":"stop"}]}`,
	}, "")

	_, err := LoadPolicySet(dir, nil)
	if err == nil {
		t.Fatal("a pack with no declared tier was accepted")
	}
	if !strings.Contains(err.Error(), "want one of core, domain, overlay") {
		t.Fatalf("error does not name the accepted tiers: %v", err)
	}
}

// TestComposition_TwoRulesOnOneToolTakeTheStronger is R07-4.2.2: composition is
// deterministic and never resolves to the weaker action.
func TestComposition_TwoRulesOnOneToolTakeTheStronger(t *testing.T) {
	dir := precedenceFixture(t, map[string]string{
		"core/base":    `{"pack":"core/base","version":"1.0.0","tier":"core","rules":[{"id":"A1","law":"a","enforcement":"advisory","tool":"guardrail_validate_scope","on_unknown":"stop"}]}`,
		"domain/local": `{"pack":"domain/local","version":"1.0.0","tier":"domain","rules":[{"id":"D1","law":"d","enforcement":"blocking","tool":"guardrail_validate_scope","on_unknown":"stop","evidence":"unit"}]}`,
	}, "")

	set, err := LoadPolicySet(dir, nil)
	if err != nil {
		t.Fatalf("two additive rules on one tool must compose: %v", err)
	}
	if _, ruleID, ok := set.Attribution("guardrail_validate_scope"); !ok || ruleID != "D1" {
		t.Fatalf("attribution = %q (ok=%v), want the stronger rule D1", ruleID, ok)
	}
	conflicts := set.Conflicts()
	if len(conflicts) != 1 {
		t.Fatalf("reported %d conflicts, want 1", len(conflicts))
	}
	if conflicts[0].Winner != "D1" || conflicts[0].Loser != "A1" || conflicts[0].Effective != "blocking" {
		t.Fatalf("conflict = %+v, want D1 to win at blocking", conflicts[0])
	}
}

// TestExceptions_ReviewedExceptionAllowsAndIsRecorded is R07-4.2.3's positive
// path: with a complete, unexpired, correctly bound exception the weakening
// resolves AND the record says which exception allowed it.
func TestExceptions_ReviewedExceptionAllowsAndIsRecorded(t *testing.T) {
	packs := map[string]string{
		"core/base":    corePack,
		"domain/local": `{"pack":"domain/local","version":"1.0.0","tier":"domain","rules":[{"id":"L1","law":"core","enforcement":"advisory","tool":"guardrail_validate_scope","on_unknown":"stop"}]}`,
	}
	// The digest binds the set the exception produced, so it must be computed
	// from that set rather than guessed.
	digest := precedenceDigestFor(t, packs)
	ledger := fmt.Sprintf(`{"exceptions":[{"rule":"L1","pack":"domain/local","scope":"sandbox-only","reason":"local authoring flow needs a warning, not a block","approver":"roger","expires":"2027-01-01T00:00:00Z","resulting_rule_set_digest":%q}]}`, digest)

	dir := precedenceFixture(t, packs, ledger)
	set, err := LoadPolicySetAt(dir, nil, nowAt("2026-10-18T21:00:00Z"))
	if err != nil {
		t.Fatalf("a complete, unexpired, correctly bound exception must resolve: %v", err)
	}
	apps := set.Applications()
	if len(apps) != 1 || !apps[0].Weakened {
		t.Fatalf("applications = %+v, want one weakened application", apps)
	}
	if apps[0].Exception != "sandbox-only" {
		t.Fatalf("application exception scope = %q, want sandbox-only", apps[0].Exception)
	}
	if got := len(set.Exceptions()); got != 1 {
		t.Fatalf("snapshot reports %d exceptions, want 1", got)
	}
	if set.RuleSetDigest() == "" || set.RuleSetDigest() != digest {
		t.Fatalf("rule set digest = %q, want %q", set.RuleSetDigest(), digest)
	}
}

// TestExceptions_ExpiredIsRefused: an elapsed exception is refused rather than
// honoured. The clock is injected so the case is a fact, not a wait.
func TestExceptions_ExpiredIsRefused(t *testing.T) {
	packs := map[string]string{
		"core/base":    corePack,
		"domain/local": `{"pack":"domain/local","version":"1.0.0","tier":"domain","rules":[{"id":"L1","law":"core","enforcement":"advisory","tool":"guardrail_validate_scope","on_unknown":"stop"}]}`,
	}
	digest := precedenceDigestFor(t, packs)
	ledger := fmt.Sprintf(`{"exceptions":[{"rule":"L1","pack":"domain/local","scope":"s","reason":"r","approver":"roger","expires":"2026-10-01T00:00:00Z","resulting_rule_set_digest":%q}]}`, digest)

	dir := precedenceFixture(t, packs, ledger)
	_, err := LoadPolicySetAt(dir, nil, nowAt("2026-10-18T21:00:00Z"))
	if err == nil {
		t.Fatal("an expired exception was honoured")
	}
	if !strings.Contains(err.Error(), "expired at") {
		t.Fatalf("error does not name the expiry: %v", err)
	}
}

// TestExceptions_StaleBindingIsRefused: an exception bound to a different rule
// set is stale — the approval was for a composition that no longer exists.
func TestExceptions_StaleBindingIsRefused(t *testing.T) {
	packs := map[string]string{
		"core/base":    corePack,
		"domain/local": `{"pack":"domain/local","version":"1.0.0","tier":"domain","rules":[{"id":"L1","law":"core","enforcement":"advisory","tool":"guardrail_validate_scope","on_unknown":"stop"}]}`,
	}
	ledger := `{"exceptions":[{"rule":"L1","pack":"domain/local","scope":"s","reason":"r","approver":"roger","expires":"2027-01-01T00:00:00Z","resulting_rule_set_digest":"sha256:0000000000000000000000000000000000000000000000000000000000000000"}]}`

	dir := precedenceFixture(t, packs, ledger)
	_, err := LoadPolicySetAt(dir, nil, nowAt("2026-10-18T21:00:00Z"))
	if err == nil {
		t.Fatal("an exception bound to a different rule set was honoured")
	}
	if !strings.Contains(err.Error(), "stale") {
		t.Fatalf("error does not name the staleness: %v", err)
	}
}

// TestExceptions_UnusedIsRefused: an exception permitting no weakening is a
// permanent hole recorded as diligence. It is refused, not carried.
func TestExceptions_UnusedIsRefused(t *testing.T) {
	// A well-formed, unexpired, correctly bound exception that authorises an
	// override nobody made. There is no other pack, so nothing is weakened.
	digest := precedenceDigestFor(t, map[string]string{"core/base": corePack})
	ledger := fmt.Sprintf(`{"exceptions":[{"rule":"L1","pack":"core/base","scope":"s","reason":"r","approver":"roger","expires":"2027-01-01T00:00:00Z","resulting_rule_set_digest":%q}]}`, digest)
	dir := precedenceFixture(t, map[string]string{"core/base": corePack}, ledger)

	_, err := LoadPolicySetAt(dir, nil, nowAt("2026-10-18T21:00:00Z"))
	if err == nil {
		t.Fatal("an exception permitting no weakening was carried")
	}
	if !strings.Contains(err.Error(), "permits no weakening") {
		t.Fatalf("error does not name the unused exception: %v", err)
	}
}

// TestExceptions_IncompleteIsRefused: every R07-4.2.3 field is required. A
// record missing its approver is a comment, and a comment must not weaken a
// mandatory control.
func TestExceptions_IncompleteIsRefused(t *testing.T) {
	for _, missing := range []string{"rule", "pack", "scope", "reason", "approver", "expires", "resulting_rule_set_digest"} {
		t.Run("missing="+missing, func(t *testing.T) {
			fields := map[string]string{
				"rule": "L1", "pack": "domain/local", "scope": "s", "reason": "r",
				"approver": "roger", "expires": "2027-01-01T00:00:00Z",
				"resulting_rule_set_digest": "sha256:0",
			}
			delete(fields, missing)
			var parts []string
			for k, v := range fields {
				parts = append(parts, fmt.Sprintf("%q:%q", k, v))
			}
			ledger := `{"exceptions":[{` + strings.Join(parts, ",") + `}]}`

			dir := precedenceFixture(t, map[string]string{
				"core/base":    corePack,
				"domain/local": `{"pack":"domain/local","version":"1.0.0","tier":"domain","rules":[{"id":"L1","law":"core","enforcement":"advisory","tool":"guardrail_validate_scope","on_unknown":"stop"}]}`,
			}, ledger)
			_, err := LoadPolicySetAt(dir, nil, nowAt("2026-10-18T21:00:00Z"))
			if err == nil {
				t.Fatalf("an exception missing %q was accepted", missing)
			}
			if !strings.Contains(err.Error(), missing) {
				t.Fatalf("error does not name the missing field %q: %v", missing, err)
			}
		})
	}
}

// TestExceptions_MalformedLedgerIsRefused: a typo in the ledger must not read as
// "no exceptions", which would silently re-permit the weakening it was meant to
// gate — or, in the other direction, silently drop a review that exists.
func TestExceptions_MalformedLedgerIsRefused(t *testing.T) {
	dir := precedenceFixture(t, map[string]string{"core/base": corePack}, `{"exceptions": "not-a-list"}`)
	_, err := LoadPolicySet(dir, nil)
	if err == nil {
		t.Fatal("a malformed exception ledger was read as an empty one")
	}
	if !strings.Contains(err.Error(), "not a recognised ledger") {
		t.Fatalf("error does not name the ledger: %v", err)
	}
}

// TestPrecedence_ShippedTreeIsASingleCorePack: the checked-in tree declares its
// tier and resolves through the same precedence path, so the feature is not
// skipped because the tree happens to be small.
func TestPrecedence_ShippedTreeIsASingleCorePack(t *testing.T) {
	set, err := LoadPolicySet(repoPackDir(t), knownToolsFor(t))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	apps := set.Applications()
	if len(apps) == 0 {
		t.Fatal("no applications reported; the assertions below would be vacuous")
	}
	for _, app := range apps {
		if app.Tier != "core" {
			t.Fatalf("rule %s resolved at tier %q, want core", app.RuleID, app.Tier)
		}
		if app.Weakened {
			t.Fatalf("rule %s is reported weakened with no other pack present", app.RuleID)
		}
	}
	if len(set.Conflicts()) != 0 {
		t.Fatalf("single-pack tree reported %d per-tool conflicts", len(set.Conflicts()))
	}
}

// precedenceDigestFor computes the rule-set digest a fixture will produce, so an
// exception ledger can bind to it without the test hand-computing a hash the
// implementation would then be free to change.
func precedenceDigestFor(t *testing.T, packs map[string]string) string {
	t.Helper()
	dir := tempPackDir(t, packs)
	digest, err := PolicyRuleSetDigest(dir, nil)
	if err != nil {
		// The fixture has no ledger yet, so it cannot resolve — that is exactly
		// the two-step the digest helper exists for. A failure here means the
		// helper does not serve the case it was written for.
		t.Fatalf("rule-set digest: %v", err)
	}
	if digest == "" {
		t.Fatal("rule-set digest is empty; an exception could not bind to it")
	}
	return digest
}

// tempPackDir writes packs plus a consistent lock, and no exception ledger.
func tempPackDir(t *testing.T, packs map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range packs {
		writePack(t, dir, name, body)
	}
	writeLockFor(t, dir, "")
	return dir
}
