package validation

import (
	"encoding/json"
	"os"
	"testing"
)

type auditRule struct {
	RuleID   string `json:"rule_id"`
	Pattern  string `json:"pattern"`
	Category string `json:"category"`
	Enabled  bool   `json:"enabled"`
}

// F021 audit: which destructive commands do the shipped pattern rules actually catch?
func TestDangerousCommandCorpus(t *testing.T) {
	raw, err := os.ReadFile("../../../.guardrails/prevention-rules/extracted-rules.json")
	if err != nil {
		t.Skipf("rules file not found: %v", err)
	}
	var doc struct {
		Rules []auditRule `json:"rules"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	caught := func(cmd string) bool {
		for _, r := range doc.Rules {
			if !r.Enabled {
				continue
			}
			if ok, err := MatchPattern(r.Pattern, cmd); err == nil && ok {
				return true
			}
		}
		return false
	}
	// current truth: asserted so a regression OR an improvement is noticed.
	corpus := []struct {
		cmd  string
		want bool
	}{
		{"git push --force origin main", true},
		{"git push --force-with-lease origin feat", false},
		{"git push origin feat", false},
		{"rm -rf /tmp/build", false},
		{"git reset --hard HEAD~5", true},
		{"git push -f origin main", true},
		{"git push origin +main", true},
		{"git push origin --delete main", true},
		{"git push origin :main", true},
		{"git branch -D main", true},
		{"git filter-repo --path x --invert-paths", true},
		{"git filter-branch --tree-filter 'rm x' HEAD", true},
		{"git clean -fdx", true},
		{"rm -rf /", true},
		{"rm -rf ~", true},
		{"git status", false},
		{"git log --oneline", false},
	}
	var gaps []string
	for _, c := range corpus {
		got := caught(c.cmd)
		if got != c.want {
			t.Errorf("%q: caught=%v, recorded truth=%v (update corpus and docs/reboot/F021-AUDIT.md if rules changed)", c.cmd, got, c.want)
		}
		if !got && c.want && c.cmd != "git status" && c.cmd != "git log --oneline" {
			gaps = append(gaps, c.cmd)
		}
	}
	t.Logf("destructive commands NOT matched by shipped pattern rules: %d: %q", len(gaps), gaps)
}

func TestShippedPatternsCompile(t *testing.T) {
	for _, f := range []string{"extracted-rules.json", "pattern-rules.json"} {
		raw, err := os.ReadFile("../../../.guardrails/prevention-rules/" + f)
		if err != nil {
			t.Skip(err)
		}
		var doc struct {
			Rules []auditRule `json:"rules"`
		}
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatal(err)
		}
		for _, r := range doc.Rules {
			// The loader now refuses a rule set containing an invalid pattern, so every shipped rule must validate.
			if err := ValidatePattern(r.Pattern); err != nil {
				t.Errorf("%s %s: %v", f, r.RuleID, err)
			}
		}
	}
}
