package config

import (
	"strings"
	"testing"
	"time"
)

// TestLegacyAcceptanceDeadline covers the Spec 11 Sec. 4.5.2 removal switch:
// an unset value means "not configured" (never expires), a valid RFC3339
// value parses, and a malformed value is a hard error rather than a silent
// "no deadline".
func TestLegacyAcceptanceDeadline(t *testing.T) {
	deadline := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name     string
		value    string
		wantTime time.Time
		wantSet  bool
		wantErr  bool
	}{
		{name: "unset is not configured", value: "", wantSet: false},
		{name: "whitespace is not configured", value: "   ", wantSet: false},
		{name: "valid RFC3339", value: "2026-12-01T00:00:00Z", wantTime: deadline, wantSet: true},
		{name: "valid RFC3339 with offset", value: "2026-11-30T18:00:00-06:00", wantTime: deadline, wantSet: true},
		{name: "malformed is a hard error", value: "2026-12-01", wantErr: true},
		{name: "garbage is a hard error", value: "soon", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &Config{LegacyAcceptanceUntil: tc.value}
			got, set, err := c.LegacyAcceptanceDeadline()
			if tc.wantErr {
				if err == nil {
					t.Fatalf("want error, got nil (deadline=%v set=%v)", got, set)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if set != tc.wantSet {
				t.Fatalf("set=%v want %v", set, tc.wantSet)
			}
			if tc.wantSet && !got.Equal(tc.wantTime) {
				t.Fatalf("deadline=%v want %v", got, tc.wantTime)
			}
		})
	}
}

// TestLegacyAcceptanceExpired is the testable removal-date switch required by
// Spec 11 acceptance criterion 8: an unconfigured switch never expires, a
// configured switch expires strictly after its deadline, and a malformed value
// fails closed (error) rather than reporting "not expired".
func TestLegacyAcceptanceExpired(t *testing.T) {
	deadline := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name    string
		value   string
		now     time.Time
		want    bool
		wantErr bool
	}{
		{name: "unconfigured never expires", value: "", now: deadline.AddDate(1, 0, 0), want: false},
		{name: "before deadline", value: "2026-12-01T00:00:00Z", now: deadline.Add(-time.Hour), want: false},
		{name: "at deadline is not yet expired", value: "2026-12-01T00:00:00Z", now: deadline, want: false},
		{name: "after deadline", value: "2026-12-01T00:00:00Z", now: deadline.Add(time.Second), want: true},
		{name: "malformed fails closed", value: "not-a-time", now: deadline, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &Config{LegacyAcceptanceUntil: tc.value}
			got, err := c.LegacyAcceptanceExpired(tc.now)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("want error, got nil (expired=%v)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("expired=%v want %v", got, tc.want)
			}
		})
	}
}

// TestValidateRejectsMalformedLegacyDeadline proves Validate fails startup on a
// malformed LEGACY_ACCEPTANCE_UNTIL: an unreadable deadline must never silently
// disable the retirement of the legacy keys.
func TestValidateRejectsMalformedLegacyDeadline(t *testing.T) {
	good := validMinimalConfig()
	good.LegacyAcceptanceUntil = "2026-12-01T00:00:00Z"
	if err := good.Validate(); err != nil {
		t.Fatalf("valid deadline rejected: %v", err)
	}

	unset := validMinimalConfig()
	unset.LegacyAcceptanceUntil = ""
	if err := unset.Validate(); err != nil {
		t.Fatalf("unset deadline rejected: %v", err)
	}

	bad := validMinimalConfig()
	bad.LegacyAcceptanceUntil = "2026-12-01"
	err := bad.Validate()
	if err == nil {
		t.Fatalf("malformed LEGACY_ACCEPTANCE_UNTIL accepted by Validate")
	}
	if !strings.Contains(err.Error(), "LEGACY_ACCEPTANCE_UNTIL") {
		t.Fatalf("error must name LEGACY_ACCEPTANCE_UNTIL, got %v", err)
	}
}
