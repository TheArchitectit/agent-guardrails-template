package database

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// TestMigrationManifestIntegrity pins the migration manifest contract the
// one-shot migration job (Spec 18 R18-02) depends on. It is the regression
// guard for the class of defect found 2026-10-15: the old runner split each
// file into statement chunks and silently dropped any chunk whose first line
// was a SQL comment, so migration 001 was recorded as applied while creating
// zero tables (fail-open). A runner that executes a file as a unit removes that
// class; this test keeps the manifest itself honest so the unit-executed job
// cannot be fed an ambiguous or gapped manifest.
func TestMigrationManifestIntegrity(t *testing.T) {
	dir := filepath.Join("migrations")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("cannot read migrations dir: %v", err)
	}

	upName := regexp.MustCompile(`^([0-9]{3,})_([A-Za-z0-9_]+)\.up\.sql$`)
	downName := regexp.MustCompile(`^([0-9]{3,})_([A-Za-z0-9_]+)\.down\.sql$`)

	ups := map[int]string{}
	downs := map[int]bool{}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if m := upName.FindStringSubmatch(name); m != nil {
			v, _ := strconv.Atoi(m[1])
			if _, dup := ups[v]; dup {
				t.Errorf("duplicate migration version %d", v)
			}
			ups[v] = name

			// Non-vacuity: an empty migration must never be recorded.
			data, readErr := os.ReadFile(filepath.Join(dir, name))
			if readErr != nil {
				t.Fatalf("cannot read %s: %v", name, readErr)
			}
			if strings.TrimSpace(string(data)) == "" {
				t.Errorf("migration %s is empty", name)
			}
		} else if m := downName.FindStringSubmatch(name); m != nil {
			v, _ := strconv.Atoi(m[1])
			downs[v] = true
		}
	}

	if len(ups) == 0 {
		t.Fatal("no migration files found")
	}

	versions := make([]int, 0, len(ups))
	for v := range ups {
		versions = append(versions, v)
	}
	sort.Ints(versions)

	// Release policy: contiguous ascending versions, each with a paired down
	// script, and the highest version equal to the declared minimum the server
	// refuses to start below (R18-05).
	for i, v := range versions {
		if v != i+1 {
			t.Errorf("manifest gap: expected version %d, found %d", i+1, v)
		}
		if !downs[v] {
			t.Errorf("migration %03d (%s) has no paired .down.sql", v, ups[v])
		}
	}
	if highest := versions[len(versions)-1]; highest != MinRequiredSchemaVersion {
		t.Errorf("highest migration version %d does not match MinRequiredSchemaVersion %d", highest, MinRequiredSchemaVersion)
	}
}

// TestLedgerVersionRoundTrip pins that the identifier the migration job writes
// into the ledger is exactly what ParseMigrationVersion reads back, so the
// server's compatibility check (R18-05) can never silently reject a version the
// job itself recorded.
func TestLedgerVersionRoundTrip(t *testing.T) {
	for v := 1; v <= MinRequiredSchemaVersion; v++ {
		recorded := strconv.Itoa(v)
		if v < 1000 {
			recorded = pad3(v)
		}
		got, err := ParseMigrationVersion(recorded)
		if err != nil {
			t.Fatalf("ParseMigrationVersion(%q) errored: %v", recorded, err)
		}
		if got != v {
			t.Errorf("ParseMigrationVersion(%q) = %d, want %d", recorded, got, v)
		}
	}
}

func pad3(v int) string {
	s := strconv.Itoa(v)
	for len(s) < 3 {
		s = "0" + s
	}
	return s
}
