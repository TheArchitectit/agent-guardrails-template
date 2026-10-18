package guardrails

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Spec 07 R07-4.2 — precedence and mandatory controls.
//
// Before this file a pack had an `enforcement` class and nothing else. There
// was no declared tier, so "core rules are mandatory by default" described a
// tree that could not express core-ness: every pack was interchangeable, and a
// second pack declaring a core rule id could lower its severity or retarget its
// tool with nothing in the resolver positioned to notice. The exception ledger
// did not exist either, so a weakening had no reviewed form — the only outcomes
// were "not possible" and "silent", and "not possible" was false.
//
// Three requirements are implemented here, and they are separable:
//
//   - R07-4.2.1: a pack declares its tier (core, domain, overlay), and the
//     declaration must agree with the pack's own name, so a domain policy
//     cannot be filed as core or the reverse. Precedence follows the tier,
//     never discovery order. A higher tier may strengthen a core rule freely;
//     lowering its severity or retargeting its tool is refused unless an
//     allowlisted, unexpired exception names it.
//   - R07-4.2.2: additive rules compose deterministically; where more than one
//     effective rule governs a tool, the STRONGER enforcement applies. There is
//     no branch that resolves a conflict to the weaker action, because that
//     would make a conflicting pack a way to shed a control.
//   - R07-4.2.3: every exception records the affected rule, scope, reason,
//     approving identity, expiry and the digest of the rule set it produced. An
//     expired exception, an exception permitting no real weakening, or one
//     bound to a different set is refused: a stale or unused exception is a
//     permanent hole recorded as diligence.

// tierRank orders pack tiers. A higher rank overrides a lower one.
var tierRank = map[string]int{"core": 0, "domain": 1, "overlay": 2}

// enforcementRank orders the severity classes the pack schema permits.
// R07-4.2.2 resolves a per-tool conflict by taking the maximum, so a conflict
// can only ever strengthen the effective control.
var enforcementRank = map[string]int{"advisory": 0, "checked": 1, "blocking": 2}

// EnforcementRank exposes the severity ordering the pack schema publishes.
func EnforcementRank(class string) int { return enforcementRank[class] }

// Exception is one reviewed, allowlisted weakening of a core rule (R07-4.2.1 /
// R07-4.2.3). Every field is required: an exception missing its approver,
// scope, reason or expiry is not a review, it is a comment.
type Exception struct {
	Rule string `json:"rule"`
	// Pack is the pack that WEAKENS the core rule, not the pack that declares
	// the core rule: the exception is permission for this override.
	Pack     string `json:"pack"`
	Scope    string `json:"scope"`
	Reason   string `json:"reason"`
	Approver string `json:"approver"`
	Expires  string `json:"expires"`
	// ResultingRuleSetDigest binds the approval to the exact effective rule set
	// it produced. A set that changed after approval makes the exception stale:
	// the approval was for a different composition.
	ResultingRuleSetDigest string `json:"resulting_rule_set_digest"`
}

// RuleApplication records how one rule id resolved across tiers. It is reported
// on the diagnostic surface so an operator can see which declaration took
// effect, what the mandatory baseline was, and whether an exception was needed.
type RuleApplication struct {
	RuleID     string   `json:"rule_id"`
	Pack       string   `json:"pack"`
	Tier       string   `json:"tier"`
	DeclaredBy []string `json:"declared_by"`
	Base       string   `json:"base_enforcement"`
	Effective  string   `json:"effective_enforcement"`
	// Weakened is true when the effective declaration lowers or retargets a
	// rule a mandatory lower tier declared. Such an application cannot exist
	// without a matching exception.
	Weakened bool `json:"weakened"`
	// Exception is the scope of the exception that allowed the weakening.
	Exception string `json:"exception_scope,omitempty"`
}

// RuleConflict records two effective rules governing one tool and which won, so
// "the stronger action applies" is observable rather than asserted.
type RuleConflict struct {
	Tool      string `json:"tool"`
	Winner    string `json:"winner"`
	Loser     string `json:"loser"`
	Effective string `json:"effective_enforcement"`
}

// PrecedenceResult is the resolved rule set: one effective declaration per rule
// id, the per-tool composition, the conflicts it resolved, and the digest the
// exception ledger binds to.
type PrecedenceResult struct {
	Applications []RuleApplication
	Rules        []PackRule
	RuleToPack   map[string]string
	ToolToRule   map[string]PackRule
	Conflicts    []RuleConflict
	Exceptions   []Exception
	Digest       string
}

// ruleSetDigest is the identity of the effective rule set: one row per rule id
// in sorted order, each [rule_id, pack, tier, effective_enforcement], encoded
// compactly. No timestamps and no map iteration, so two resolutions of the same
// inputs agree and a changed precedence changes the digest.
//
// The exception ledger is deliberately NOT part of this digest. The digest is
// what an exception binds TO; including the ledger would make the binding
// circular and let an edited reason look like a changed policy.
func ruleSetDigest(rules []RuleApplication) (string, error) {
	sorted := make([]RuleApplication, len(rules))
	copy(sorted, rules)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].RuleID < sorted[j].RuleID })

	rows := make([][]string, 0, len(sorted))
	for _, r := range sorted {
		rows = append(rows, []string{r.RuleID, r.Pack, r.Tier, r.Effective})
	}
	encoded, err := json.Marshal(rows)
	if err != nil {
		return "", fmt.Errorf("encode canonical rule set: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

type ruleDeclaration struct {
	pack *Pack
	rule PackRule
}

// resolvePolicyPrecedence turns the declared rules into the effective rule set.
//
// The effective declaration of a rule id is the one from the highest tier that
// declares it. A tie at the same tier is refused as ambiguous: two packs of
// equal standing disagreeing about one control have no deterministic winner,
// and choosing one by discovery order would make the outcome depend on the
// spelling of file names.
func resolvePolicyPrecedence(packsByName map[string]*Pack, exceptions []Exception, now time.Time) (*PrecedenceResult, error) {
	_ = now // expiry is decided in validateExceptions, against the same clock
	byID := map[string][]ruleDeclaration{}
	names := make([]string, 0, len(packsByName))
	for name := range packsByName {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		pack := packsByName[name]
		for _, rule := range pack.Rules {
			byID[rule.ID] = append(byID[rule.ID], ruleDeclaration{pack: pack, rule: rule})
		}
	}

	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	applications := make([]RuleApplication, 0, len(ids))
	for _, id := range ids {
		decls := byID[id]
		sort.Slice(decls, func(i, j int) bool {
			ri, rj := tierRank[decls[i].pack.Tier], tierRank[decls[j].pack.Tier]
			if ri != rj {
				return ri < rj
			}
			return decls[i].pack.Name < decls[j].pack.Name
		})
		top := decls[len(decls)-1]
		if len(decls) > 1 && tierRank[decls[len(decls)-2].pack.Tier] == tierRank[top.pack.Tier] {
			return nil, fmt.Errorf(
				"rule id %q is declared by two %s packs (%s and %s); equal tiers have no deterministic precedence, so the conflict is refused rather than resolved by file order (R07-4.2.2)",
				id, top.pack.Tier, decls[len(decls)-2].pack.Name, top.pack.Name)
		}

		// The baseline is the strongest declaration among the LOWER tiers: an
		// override may not lower what a mandatory tier already required. The
		// baseline starts from the lowest tier present, not from the override,
		// or a domain pack lowering core would be compared against itself and
		// every weakening would look like no change.
		base := decls[0].rule
		for _, d := range decls[:len(decls)-1] {
			if enforcementRank[d.rule.Enforcement] > enforcementRank[base.Enforcement] {
				base = d.rule
			}
		}
		hasLowerTier := len(decls) > 1

		app := RuleApplication{
			RuleID: id, Pack: top.pack.Name, Tier: top.pack.Tier,
			Base: base.Enforcement, Effective: top.rule.Enforcement,
		}
		for _, d := range decls {
			app.DeclaredBy = append(app.DeclaredBy, d.pack.Name)
		}

		// A retargeted tool is a weakening even at equal severity: the core
		// rule stops governing the tool it was written for.
		if hasLowerTier {
			app.Weakened = enforcementRank[top.rule.Enforcement] < enforcementRank[base.Enforcement] || top.rule.Tool != base.Tool
		}
		if app.Weakened {
			if exc, ok := exceptionFor(exceptions, id, top.pack.Name); ok {
				app.Exception = exc.Scope
			}
		}
		applications = append(applications, app)
	}

	digest, err := ruleSetDigest(applications)
	if err != nil {
		return nil, err
	}

	result := &PrecedenceResult{
		Applications: applications,
		RuleToPack:   map[string]string{},
		ToolToRule:   map[string]PackRule{},
		Exceptions:   exceptions,
		Digest:       digest,
	}

	// The effective rule body comes from the pack that won precedence, not from
	// whichever copy was parsed last.
	for _, app := range applications {
		pack := packsByName[app.Pack]
		for _, rule := range pack.Rules {
			if rule.ID == app.RuleID {
				result.Rules = append(result.Rules, rule)
				result.RuleToPack[rule.ID] = app.Pack
			}
		}
	}
	sort.Slice(result.Rules, func(i, j int) bool { return result.Rules[i].ID < result.Rules[j].ID })

	conflicts, err := composePerTool(result.Rules, result.ToolToRule)
	if err != nil {
		return nil, err
	}
	result.Conflicts = conflicts
	return result, nil
}

// exceptionFor finds the exception permitting a specific override. Keyed by the
// rule AND the overriding pack: an exception granted to one domain pack must not
// silently authorise a different pack to weaken the same rule.
func exceptionFor(exceptions []Exception, ruleID, pack string) (Exception, bool) {
	for _, e := range exceptions {
		if e.Rule == ruleID && e.Pack == pack {
			return e, true
		}
	}
	return Exception{}, false
}

// tierName reports the pack's tier, treating an undeclared tier as core. The
// schema refuses an undeclared tier on a parsed pack, so this only ever names a
// pack that reached resolution — but the default is still the strongest tier,
// never the weakest, because "we do not know what this pack is" must not read
// as "it may override anything".
func (d ruleDeclaration) tierName() string {
	if d.pack.Tier == "" {
		return "core"
	}
	return d.pack.Tier
}

// composePerTool applies R07-4.2.2: where more than one effective rule governs
// a tool, the strongest enforcement is the effective one and it is written into
// toolToRule. A conflict may only raise the control; there is no branch that
// lets the weaker declaration win, which is why "ambiguity" cannot silently
// resolve to the weaker action.
//
// Equal severities are ordered by rule id so the OUTCOME is deterministic. That
// is not a weaker choice: both declarations demand the same action, so the
// only thing being decided is which identity the audit record names.
func composePerTool(rules []PackRule, toolToRule map[string]PackRule) ([]RuleConflict, error) {
	byTool := map[string][]PackRule{}
	for _, rule := range rules {
		if rule.Tool == "" {
			continue
		}
		byTool[rule.Tool] = append(byTool[rule.Tool], rule)
	}
	tools := make([]string, 0, len(byTool))
	for tool := range byTool {
		tools = append(tools, tool)
	}
	sort.Strings(tools)

	var conflicts []RuleConflict
	for _, tool := range tools {
		group := byTool[tool]
		sort.Slice(group, func(i, j int) bool {
			ri, rj := enforcementRank[group[i].Enforcement], enforcementRank[group[j].Enforcement]
			if ri != rj {
				return ri > rj
			}
			return group[i].ID < group[j].ID
		})
		winner := group[0]
		toolToRule[tool] = winner
		for _, loser := range group[1:] {
			conflicts = append(conflicts, RuleConflict{
				Tool: tool, Winner: winner.ID, Loser: loser.ID,
				Effective: winner.Enforcement,
			})
		}
	}
	return conflicts, nil
}

const exceptionLedgerName = "EXCEPTIONS.json"

// loadExceptions reads the optional exception ledger. An absent file is "no
// exceptions declared"; a present-but-empty ledger states the same fact. A
// malformed or unrecognised file is an error, never an empty ledger: a typo in
// the ledger must not silently remove every review requirement.
func loadExceptions(dir string) ([]Exception, error) {
	path := filepath.Join(dir, exceptionLedgerName)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("policy exception ledger %q: %w", path, err)
	}
	var ledger struct {
		Exceptions []Exception `json:"exceptions"`
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&ledger); err != nil {
		return nil, fmt.Errorf("policy exception ledger %q is not a recognised ledger: %w", path, err)
	}
	return ledger.Exceptions, nil
}

// validateExceptions enforces R07-4.2.3 in three ordered passes, because the
// three failures are different facts and an operator fixes them differently:
//
//  1. field completeness and expiry — a record that is not a review;
//  2. the digest binding — an approval for a composition that no longer exists;
//  3. coverage both ways — a weakening with no exception (a control silently
//     shed) and an exception with no weakening (a permanent hole recorded as
//     diligence).
//
// Field checks run first so a ledger missing its approver reports that, rather
// than reporting the weakening it failed to authorise.
func validateExceptions(result *PrecedenceResult, now time.Time) error {
	if result == nil {
		return fmt.Errorf("policy precedence result is required")
	}
	expectedDigest := result.Digest

	seen := map[string]bool{}
	for _, e := range result.Exceptions {
		at := fmt.Sprintf("policy exception for rule %q in pack %q", e.Rule, e.Pack)
		for _, field := range []struct{ name, value string }{
			{"rule", e.Rule}, {"pack", e.Pack}, {"scope", e.Scope}, {"reason", e.Reason},
			{"approver", e.Approver}, {"expires", e.Expires},
			{"resulting_rule_set_digest", e.ResultingRuleSetDigest},
		} {
			if strings.TrimSpace(field.value) == "" {
				return fmt.Errorf("%s has no %s; an exception missing it is a comment, not a review (R07-4.2.3)", at, field.name)
			}
		}
		key := e.Rule + "\x00" + e.Pack
		if seen[key] {
			return fmt.Errorf("%s is declared twice; the effective weakening would be ambiguous", at)
		}
		seen[key] = true

		expires, err := time.Parse(time.RFC3339, e.Expires)
		if err != nil {
			return fmt.Errorf("%s has an expiry %q that is not RFC3339: %w", at, e.Expires, err)
		}
		if !expires.After(now) {
			return fmt.Errorf("%s expired at %s; an elapsed exception is refused rather than honoured (R07-4.2.3)", at, e.Expires)
		}
		if e.ResultingRuleSetDigest != expectedDigest {
			return fmt.Errorf(
				"%s binds rule-set digest %s but the resolved set hashes to %s; the policy changed after approval, so the exception is stale (R07-4.2.3)",
				at, e.ResultingRuleSetDigest, expectedDigest)
		}
	}

	// A weakening must be authorised. The message names what was lowered and by
	// whom, because "policy invalid" is not a message an operator can act on.
	for _, app := range result.Applications {
		if !app.Weakened {
			continue
		}
		if _, ok := exceptionFor(result.Exceptions, app.RuleID, app.Pack); ok {
			continue
		}
		if app.Base != app.Effective {
			return fmt.Errorf(
				"rule %q is lowered from %q to %q by %s %q with no allowlisted exception (R07-4.2.1)",
				app.RuleID, app.Base, app.Effective, app.Tier, app.Pack)
		}
		return fmt.Errorf(
			"rule %q is retargeted by %s %q and its core tool coverage is dropped; that requires a named exception (R07-4.2.1)",
			app.RuleID, app.Tier, app.Pack)
	}

	for _, e := range result.Exceptions {
		used := false
		for _, app := range result.Applications {
			if app.RuleID == e.Rule && app.Pack == e.Pack && app.Weakened {
				used = true
				break
			}
		}
		if !used {
			return fmt.Errorf(
				"exception for rule %q in pack %q (scope %q) permits no weakening in the resolved set; an exception nobody uses is a permanent hole recorded as diligence (R07-4.2.3)",
				e.Rule, e.Pack, e.Scope)
		}
	}
	return nil
}
