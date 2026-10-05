package database

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
)

// MinRequiredSchemaVersion is the minimum applied schema version this binary
// requires before it will serve traffic (Spec 18 §2/§3, R18-05). It matches
// the highest numbered migration shipped in internal/database/migrations; a
// database below this version must be brought up by the one-shot migration
// job before the server starts.
const MinRequiredSchemaVersion = 18

// SchemaLedger abstracts the migration ledger so the compatibility decision
// can be tested without a live database.
type SchemaLedger interface {
	// AppliedVersions returns the raw version identifiers recorded in the
	// schema_migrations ledger.
	AppliedVersions(ctx context.Context) ([]string, error)
}

// SQLSchemaLedger reads the schema_migrations ledger from PostgreSQL.
type SQLSchemaLedger struct {
	db *sql.DB
}

// NewSQLSchemaLedger returns a ledger backed by the given database handle.
func NewSQLSchemaLedger(db *sql.DB) *SQLSchemaLedger {
	return &SQLSchemaLedger{db: db}
}

// AppliedVersions reads all recorded version identifiers from the
// schema_migrations table. A missing table is an error, not an empty list:
// it means the migration job has never run against this database.
func (l *SQLSchemaLedger) AppliedVersions(ctx context.Context) ([]string, error) {
	rows, err := l.db.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("cannot read schema_migrations ledger: %w", err)
	}
	defer rows.Close()

	var versions []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("cannot scan schema_migrations row: %w", err)
		}
		versions = append(versions, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("schema_migrations ledger read failed: %w", err)
	}
	return versions, nil
}

// ParseMigrationVersion extracts the numeric version from a ledger row
// identifier ("018_add_agent_state_tables" → 18).
func ParseMigrationVersion(version string) (int, error) {
	digits := version
	if idx := strings.IndexAny(version, "_-. "); idx >= 0 {
		digits = version[:idx]
	}
	v, err := strconv.Atoi(strings.TrimSpace(digits))
	if err != nil {
		return 0, fmt.Errorf("malformed schema version %q: %w", version, err)
	}
	return v, nil
}

// EvaluateSchemaCompatibility is the pure compatibility decision (Spec 18
// §3 step 3): the ledger must record at least one applied migration and the
// highest applied version must be at or above minVersion. It never repairs,
// renumbers, or interprets an empty/ambiguous ledger as success.
func EvaluateSchemaCompatibility(versions []string, minVersion int) error {
	if len(versions) == 0 {
		return fmt.Errorf(
			"schema compatibility check failed: schema_migrations ledger is empty; run the versioned migration job before starting the server (minimum required schema version %d)",
			minVersion)
	}

	highest := -1
	for _, raw := range versions {
		v, err := ParseMigrationVersion(raw)
		if err != nil {
			return fmt.Errorf("schema compatibility check failed: %w", err)
		}
		if v > highest {
			highest = v
		}
	}

	if highest < minVersion {
		return fmt.Errorf(
			"schema compatibility check failed: database schema version %d is below the minimum required version %d; run the versioned migration job before starting the server",
			highest, minVersion)
	}
	return nil
}

// CheckSchemaCompatibility verifies the recorded schema version is at or
// above the minimum required by this binary. A database below the target
// (or an unreadable/empty ledger) refuses startup (Spec 18 R18-05/R18-07).
func CheckSchemaCompatibility(ctx context.Context, ledger SchemaLedger, minVersion int) error {
	versions, err := ledger.AppliedVersions(ctx)
	if err != nil {
		return fmt.Errorf("schema compatibility check failed: %w", err)
	}
	return EvaluateSchemaCompatibility(versions, minVersion)
}
