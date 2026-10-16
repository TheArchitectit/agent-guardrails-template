package keymaterial

import (
	"math"
	"strings"
	"testing"
)

// TestVerifierKeyEntropyBits pins the estimator's behaviour: an all-identical
// key carries zero bits regardless of length, a short repeated pattern is
// credited only its block length, and a full-length high-alphabet key reaches
// the 256-bit bound R17-07 names.
func TestVerifierKeyEntropyBits(t *testing.T) {
	cases := []struct {
		name string
		key  string
		want float64
	}{
		{"empty", "", 0},
		{"whitespace", "   ", 0},
		{"single repeated byte", strings.Repeat("x", 64), 0},
		// A repeated 3-char block is credited its block length (3), not its
		// full 12 characters: a pattern carries no more entropy than one copy.
		{"short repeated pattern", "abc" + "abc" + "abc" + "abc", 3 * math.Log2(3)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := VerifierKeyEntropyBits(tc.key); math.Abs(got-tc.want) > 1e-9 {
				t.Fatalf("VerifierKeyEntropyBits(%q) = %v, want %v", tc.key, got, tc.want)
			}
		})
	}

	// A 64-hex-character random-looking key carries the full 256-bit bound.
	strong := "9f2c4e6a8b1d3f5a7c9e0b2d4f6a8c1e3b5d7f9a0c2e4b6d8f1a3c5e7b9d0f2a"
	if got := VerifierKeyEntropyBits(strong); got < ProductionVerifierKeyBits {
		t.Fatalf("a 64-hex key must reach the production bound, got %v", got)
	}
}

// TestValidateVerifierKeyEntropyRejectsWeak proves the gate fails closed on a
// key that passes a length check but carries no real entropy — the exact defect
// R17-07 names ("a length check is not an entropy proof").
func TestValidateVerifierKeyEntropyRejectsWeak(t *testing.T) {
	weak := []string{
		strings.Repeat("x", 64), // length 64, one distinct byte
		"short",                 // below the length floor too
		"secret-verifier-key",   // 19 chars, tiny alphabet, low bits
	}
	for _, key := range weak {
		if err := ValidateVerifierKeyEntropy(key, MinVerifierKeyBits); err == nil {
			t.Fatalf("weak key %q must be rejected at the %v-bit floor", key, MinVerifierKeyBits)
		}
	}

	// A key that clears the floor but not the production bound.
	mid := "unit-test-verifier-key-0123456789"
	if err := ValidateVerifierKeyEntropy(mid, MinVerifierKeyBits); err != nil {
		t.Fatalf("a 147-bit key must clear the %v-bit floor: %v", MinVerifierKeyBits, err)
	}
	if err := ValidateVerifierKeyEntropy(mid, ProductionVerifierKeyBits); err == nil {
		t.Fatalf("a 147-bit key must be refused at the %v-bit production bound", ProductionVerifierKeyBits)
	}

	// The error names the requirement so an operator can act on it.
	err := ValidateVerifierKeyEntropy(strings.Repeat("x", 64), MinVerifierKeyBits)
	if err == nil || !strings.Contains(err.Error(), "R17-07") {
		t.Fatalf("error must name R17-07, got %v", err)
	}
}
