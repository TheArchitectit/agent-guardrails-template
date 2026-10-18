package controlplane

import (
	"strings"
	"testing"
)

// R08-4.3.3 — "Secret values, raw credentials, and unnecessary user content
// MUST not be stored in an evidence bundle. Redaction rules MUST be tested."
//
// Each case is a SHAPE, not a fixture value: a rule that only catches the
// string it was written against is a rule that fails on the next credential.
func TestRedactDetailRemovesCredentialShapes(t *testing.T) {
	cases := []struct {
		name  string
		in    string
		avoid []string // none of these may survive
		keep  []string // these must survive, so redaction is not blanket deletion
	}{
		{
			name:  "bearer token",
			in:    "upstream refused: Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.payload.sig",
			avoid: []string{"eyJhbGciOiJIUzI1NiJ9", "payload.sig"},
		},
		{
			name:  "prefixed api key",
			in:    "moderation backend rejected key sk-live-9f8a7b6c5d4e3f2a1b0c",
			avoid: []string{"sk-live-9f8a7b6c5d4e3f2a1b0c"},
			keep:  []string{"moderation backend rejected key"},
		},
		{
			name:  "github token",
			in:    "clone failed with ghp_A1b2C3d4E5f6G7h8I9j0K1l2M3n4",
			avoid: []string{"ghp_A1b2C3d4E5f6G7h8I9j0K1l2M3n4"},
			keep:  []string{"clone failed with"},
		},
		{
			name:  "url userinfo",
			in:    "dial failed for postgres://app:hunter2correct@db.internal:5432/guardrails",
			avoid: []string{"hunter2correct"},
			keep:  []string{"db.internal", "postgres://"},
		},
		{
			// The value is deliberately LOW entropy. A realistic-looking hex
			// literal here is indistinguishable from a real credential to a
			// secret scanner, and the first version of this fixture turned
			// Secret Validation red (gitleaks generic-api-key, entropy 4.0).
			// The rule under test keys on the SHAPE `NAME=value`, not on the
			// value's randomness, so a placeholder exercises it exactly.
			name:  "env style assignment",
			in:    "config rejected WEBHOOK_ENCRYPTION_KEY=placeholder-not-a-real-key",
			avoid: []string{"placeholder-not-a-real-key"},
			keep:  []string{"WEBHOOK_ENCRYPTION_KEY"},
		},
		{
			name:  "lowercase secret assignment",
			in:    "db_password=hunter2 should not appear",
			avoid: []string{"hunter2"},
		},
		{
			name:  "pem private key",
			in:    "loaded key -----BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzaC1rZXk\n-----END OPENSSH PRIVATE KEY----- ok",
			avoid: []string{"b3BlbnNzaC1rZXk"},
			keep:  []string{"loaded key", "ok"},
		},
		{
			name:  "jwt",
			in:    "session token eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U leaked",
			avoid: []string{"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"},
		},
		{
			name: "benign detail is untouched",
			in:   "policy engine answered in 12ms with 4 rules loaded",
			keep: []string{"policy engine answered in 12ms with 4 rules loaded"},
		},
		{
			name: "empty detail stays empty",
			in:   "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := RedactDetail(tc.in)
			for _, bad := range tc.avoid {
				if strings.Contains(got, bad) {
					t.Fatalf("redaction left %q in %q", bad, got)
				}
			}
			for _, want := range tc.keep {
				if !strings.Contains(got, want) {
					t.Fatalf("redaction removed useful context %q: got %q", want, got)
				}
			}
		})
	}
}

// Redaction must be non-vacuous: it must actually change something. Without
// this, a no-op implementation would pass every `avoid` check above only if the
// inputs never contained the token — which they do.
func TestRedactionIsNotANoOp(t *testing.T) {
	const in = "tried key sk-live-aaaaaaaaaaaaaaaaaaaa and failed"
	if RedactDetail(in) == in {
		t.Fatalf("redaction returned the input unchanged for a credential-shaped string")
	}
}

// An over-long diagnostic is truncated explicitly, and the JSON stays valid
// UTF-8 — a cut inside a multi-byte rune would produce invalid encoding in a
// signed artifact.
func TestBoundDetailTruncatesOnRuneBoundary(t *testing.T) {
	long := strings.Repeat("á", MaxDetailLength) // 2 bytes each
	got := BoundDetail(long)
	if len(got) > MaxDetailLength+len("…[truncated]") {
		t.Fatalf("bounded detail is %d bytes, over the bound", len(got))
	}
	if !strings.HasSuffix(got, "…[truncated]") {
		t.Fatalf("truncation was not marked: %q", got[len(got)-20:])
	}
	// Must still be valid UTF-8: a mid-rune cut is a real bug, not cosmetic.
	if !isValidUTF8(got) {
		t.Fatalf("truncated detail is not valid UTF-8")
	}
	// A short detail is untouched, so the bound does not mangle normal text.
	short := "component exercised once"
	if BoundDetail(short) != short {
		t.Fatalf("short detail was modified")
	}
}

func isValidUTF8(s string) bool {
	for i := 0; i < len(s); {
		b := s[i]
		switch {
		case b < 0x80:
			i++
		case b&0xE0 == 0xC0:
			if i+1 >= len(s) {
				return false
			}
			i += 2
		case b&0xF0 == 0xE0:
			if i+2 >= len(s) {
				return false
			}
			i += 3
		case b&0xF8 == 0xF0:
			if i+3 >= len(s) {
				return false
			}
			i += 4
		default:
			return false
		}
	}
	return true
}
