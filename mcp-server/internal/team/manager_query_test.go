package team

import "testing"

// GetTeamsByPhase previously compared team.Phase for exact equality while
// callers validated and passed the short form ("Phase 1"), so every valid
// filter returned an empty list. Both forms must now match.
func TestGetTeamsByPhase_AcceptsShortAndFullForms(t *testing.T) {
	m := &Manager{teams: map[int]Team{
		1: {ID: 1, Name: "Strategy", Phase: "Phase 1: Strategy, Governance & Planning"},
		2: {ID: 2, Name: "Architecture", Phase: "Phase 1: Strategy, Governance & Planning"},
		4: {ID: 4, Name: "Infra", Phase: "Phase 2: Platform & Foundation"},
		11: {ID: 11, Name: "SRE", Phase: "Phase 4: Validation & Hardening"},
		12: {ID: 12, Name: "Delivery", Phase: "Phase 5: Delivery & Sustainment"},
	}}

	cases := []struct {
		name    string
		phase   string
		wantIDs int
	}{
		{"short form", "Phase 1", 2},
		{"full label", "Phase 1: Strategy, Governance & Planning", 2},
		{"short form phase 2", "Phase 2", 1},
		{"short form phase 4", "Phase 4", 1},
		{"short form phase 5", "Phase 5", 1},
		{"no match", "Phase 3", 0},
		{"unrelated", "NotAPhase", 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := m.GetTeamsByPhase(tc.phase)
			if len(got) != tc.wantIDs {
				t.Fatalf("GetTeamsByPhase(%q) returned %d teams, want %d", tc.phase, len(got), tc.wantIDs)
			}
		})
	}
}

func TestPhaseMatches(t *testing.T) {
	const full = "Phase 1: Strategy, Governance & Planning"

	if !phaseMatches(full, "Phase 1") {
		t.Error("short form should match the full label")
	}
	if !phaseMatches(full, full) {
		t.Error("full label should match itself")
	}
	if phaseMatches(full, "Phase 10") {
		t.Error("Phase 10 must not match Phase 1")
	}
	if phaseMatches(full, "Phase") {
		t.Error("a bare 'Phase' prefix must not match")
	}
	if phaseMatches("Phase 2: Platform", "Phase 1") {
		t.Error("Phase 1 must not match a Phase 2 team")
	}
}