package database

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeLedger struct {
	versions []string
	err      error
}

func (f *fakeLedger) AppliedVersions(ctx context.Context) ([]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.versions, nil
}

// TestSchemaCompatibilityBelowTargetRefusesStartup covers the negative:
// a database whose recorded schema is below the binary's declared minimum
// must fail the startup check.
func TestSchemaCompatibilityBelowTargetRefusesStartup(t *testing.T) {
	tests := []struct {
		name     string
		versions []string
	}{
		{"single old version", []string{"001_create_tables"}},
		{"one below target", []string{"017_add_budget_tables"}},
		{"mixed history below target", []string{"001_create_tables", "005_add_file_reads_table", "012_add_scope_tracking"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckSchemaCompatibility(context.Background(), &fakeLedger{versions: tt.versions}, MinRequiredSchemaVersion)
			if err == nil {
				t.Fatalf("schema below target %d must refuse startup, got nil error", MinRequiredSchemaVersion)
			}
			if !strings.Contains(err.Error(), "below the minimum required") {
				t.Errorf("error should state the version gap, got %v", err)
			}
		})
	}
}

// TestSchemaCompatibilityAtOrAboveTargetPassesStartup covers the positive:
// a database at or above the binary's declared minimum passes.
func TestSchemaCompatibilityAtOrAboveTargetPassesStartup(t *testing.T) {
	tests := []struct {
		name     string
		versions []string
	}{
		{"at target", []string{"018_add_agent_state_tables"}},
		{"full history at target", []string{"001_create_tables", "016_add_webhooks_table", "017_add_budget_tables", "018_add_agent_state_tables"}},
		{"above target", []string{"019_add_future_table"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CheckSchemaCompatibility(context.Background(), &fakeLedger{versions: tt.versions}, MinRequiredSchemaVersion); err != nil {
				t.Fatalf("schema at or above target must pass, got %v", err)
			}
		})
	}
}

// TestSchemaCompatibilityUnusableLedger covers fail-closed handling of
// missing, empty, and malformed ledgers: none may be interpreted as ready.
func TestSchemaCompatibilityUnusableLedger(t *testing.T) {
	tests := []struct {
		name     string
		ledger   SchemaLedger
		errParts []string
	}{
		{
			name:     "empty ledger",
			ledger:   &fakeLedger{versions: []string{}},
			errParts: []string{"ledger is empty", "migration job"},
		},
		{
			name:     "ledger unreadable",
			ledger:   &fakeLedger{err: errors.New("database down")},
			errParts: []string{"database down"},
		},
		{
			name:     "malformed version row",
			ledger:   &fakeLedger{versions: []string{"001_create_tables", "not-a-version"}},
			errParts: []string{"malformed schema version"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckSchemaCompatibility(context.Background(), tt.ledger, MinRequiredSchemaVersion)
			if err == nil {
				t.Fatal("unusable ledger must fail startup, got nil error")
			}
			for _, part := range tt.errParts {
				if !strings.Contains(err.Error(), part) {
					t.Errorf("error = %v, want containing %q", err, part)
				}
			}
		})
	}
}

// TestParseMigrationVersion covers ledger-row version parsing.
func TestParseMigrationVersion(t *testing.T) {
	tests := []struct {
		raw     string
		want    int
		wantErr bool
	}{
		{"001_create_tables", 1, false},
		{"018_add_agent_state_tables", 18, false},
		{"19", 19, false},
		{"007_schema_versioning", 7, false},
		{"", 0, true},
		{"latest", 0, true},
	}

	for _, tt := range tests {
		got, err := ParseMigrationVersion(tt.raw)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseMigrationVersion(%q) error = %v, wantErr %v", tt.raw, err, tt.wantErr)
			continue
		}
		if err == nil && got != tt.want {
			t.Errorf("ParseMigrationVersion(%q) = %d, want %d", tt.raw, got, tt.want)
		}
	}
}

// TestMinRequiredSchemaVersionMatchesMigrations pins the declared minimum to
// the highest numbered migration shipped with the binary.
func TestMinRequiredSchemaVersionMatchesMigrations(t *testing.T) {
	const expected = 18
	if MinRequiredSchemaVersion != expected {
		t.Errorf("MinRequiredSchemaVersion = %d, want %d; update it when migrations change", MinRequiredSchemaVersion, expected)
	}
}
