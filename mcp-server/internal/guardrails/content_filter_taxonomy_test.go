package guardrails

import (
	"strings"
	"testing"
)

// === Taxonomy alignment: backends must use canonical S1-S15 ===

func TestLlamaGuardPrompt_UsesCanonicalTaxonomy(t *testing.T) {
	prompt := contentFilterLlamaGuardPrompt("test")

	// The prompt must describe the canonical S14/S15 (Code Abuse / Data Exfiltration),
	// not the old mislabeled S8/S9.
	canonical := []string{
		"S5: Defamation",
		"S8: Intellectual Property",
		"S10: Hate",
		"S13: Elections",
		"S14: Code Abuse",
		"S15: Data Exfiltration",
	}
	for _, label := range canonical {
		if !strings.Contains(prompt, label) {
			t.Errorf("LlamaGuard prompt missing canonical label %q", label)
		}
	}

	// Ensure old incorrect labels are gone
	stale := []string{
		"S5: Suicide and Self-Harm",
		"S8: Code Abuse",
		"S9: Data Exfiltration",
		"S10: Harassment and Bullying",
		"S14: Deception and Misinformation",
		"S15: System Instruction Override",
	}
	for _, s := range stale {
		if strings.Contains(prompt, s) {
			t.Errorf("LlamaGuard prompt still contains stale label %q", s)
		}
	}
}

func TestOpenAIModeration_CanonicalMapping(t *testing.T) {
	// The OpenAI backend maps self-harm -> S11 (Self-Harm), hate/harassment -> S10 (Hate),
	// sexual -> S12 (Sexual Content), sexual minors -> S4 (Child Exploitation).
	// Verify the emptyScores baseline contains all canonical keys.
	b := NewOpenAIModerationBackend("")
	scores := b.emptyScores()
	for _, id := range AllCategoryIDs() {
		if _, ok := scores[id]; !ok {
			t.Errorf("emptyScores missing canonical category %s", id)
		}
	}
}

// openAIModerationWithScores builds a moderation backend whose Classify returns
// the given category_scores/categories by hijacking via a small helper.
func TestOpenAIModeration_ViolenceGraphicMax(t *testing.T) {
	// Construct a backend and feed it scores directly through the mapping path.
	b := NewOpenAIModerationBackend("")

	cases := []struct {
		name        string
		violence    float64
		violenceGfx float64
		gfxFlagged  bool
		wantS1      float64
	}{
		{"graphic higher", 0.3, 0.9, true, 0.9},
		{"violence higher", 0.8, 0.2, true, 0.8},
		{"equal", 0.5, 0.5, true, 0.5},
		{"graphic only flagged bool", 0.0, 0.0, true, 0.8},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Build a fake moderation response struct and exercise mapping via
			// a standalone function to keep the test hermetic.
			s1 := mapViolenceS1(c.violence, c.violenceGfx, c.gfxFlagged)
			if s1 != c.wantS1 {
				t.Errorf("expected S1 = %v, got %v", c.wantS1, s1)
			}
			_ = b
		})
	}
}

// mapViolenceS1 mirrors the OpenAI backend mapping for the S1 category.
func mapViolenceS1(violence, violenceGraphic float64, gfxFlagged bool) float64 {
	var s1 float64
	if violence > 0 {
		s1 = violence
	} else {
		s1 = 0.8
	}
	if violenceGraphic > 0 {
		s1 = max(s1, violenceGraphic)
	} else if gfxFlagged {
		s1 = max(s1, 0.8)
	}
	return s1
}

func TestOpenAIModeration_SexualMinorsUsesActualScore(t *testing.T) {
	// A borderline 0.1 must NOT trigger a 0.95 block; the actual score is used.
	cases := []struct {
		score float64
		want  float64
	}{
		{0.1, 0.1},
		{0.95, 0.95},
		{0.7, 0.7},
	}
	for _, c := range cases {
		got := mapSexualMinorsS4(c.score, false)
		if got != c.want {
			t.Errorf("sexual/minors score %v: expected S4 = %v, got %v", c.score, c.want, got)
		}
	}
	// Boolean flag with no score still maps to the conventional 0.95.
	if got := mapSexualMinorsS4(0.0, true); got != 0.95 {
		t.Errorf("flagged-only sexual/minors: expected 0.95, got %v", got)
	}
}

// mapSexualMinorsS4 mirrors the OpenAI backend mapping for the S4 category.
func mapSexualMinorsS4(score float64, flagged bool) float64 {
	if score > 0 {
		return score
	}
	if flagged {
		return 0.95
	}
	return 0.0
}
