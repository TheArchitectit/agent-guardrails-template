// Versioned migration job (Spec 18 R18-01..R18-06).
//
// This is the one-shot schema-mutation owner. It is deliberately NOT part of
// application startup (R18-01): the server refuses to serve until this job has
// recorded a compatible schema version (see internal/database/schema_version.go).
//
// Usage:
//
//	MIGRATION_DSN_FILE=/run/secrets/migration_dsn go run scripts/run_migrations.go
//	MIGRATION_DATABASE_URL=...                 go run scripts/run_migrations.go --verify
//
// The database URL is read from a protected secret source ONLY: a 0600 file
// named by MIGRATION_DSN_FILE (preferred; a mounted secret), or the
// MIGRATION_DATABASE_URL environment variable (documented migration fallback).
// It is NEVER accepted on the command line and NEVER logged (Spec 18 Sec. 4.1,
// Spec 17 R17-08).
//
// Guarantees:
//   - R18-02 ordered manifest: numeric NN_name.up.sql files, ascending order,
//     validated for duplicates, gaps, invalid names and missing .down.sql pairs.
//   - R18-03 atomic idempotency: each version and its ledger row (version, name,
//     SHA-256 checksum, applied_at) commit in ONE transaction; a rerun is a
//     no-op that re-verifies checksums; a checksum mismatch is a hard failure
//     and is never auto-repaired.
//   - R18-04 serialized execution: a bounded PostgreSQL advisory lock is held
//     for the whole run; contention/timeout is a failure, not a no-op.
//   - R18-05 version truth: the ledger is authoritative; the run records the
//     numeric version, name and checksum the server's compatibility check reads.
//   - R18-06 deliberate rollback: .down.sql files are never executed here.
//     Rollback is an explicit, operator-driven action, out of this job's path.
//
// The migration content is executed as a single Exec per version so dollar-
// quoted function bodies ($$...$$) and comments are handled by the server, not
// by a hand-rolled splitter (the previous splitter silently dropped any
// statement chunk that began with a SQL comment — see the fix note in
// docs/specs/18-migration-startup-and-readiness.md).
package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// migrationLockKey is a documented, stable advisory-lock namespace derived
// from the schema this job owns. It must not change without a migration plan.
const migrationLockKey int64 = 8412523045123456789

// lockTimeout bounds the wait for the advisory lock (R18-04). A timeout is a
// failed job, never a successful no-op.
const lockTimeout = 30 * time.Second

// defaultMigrationsDir is resolved relative to the module root at runtime.
const defaultMigrationsDir = "internal/database/migrations"

// migrationFileName matches an immutable, sortable, uniquely numbered file:
// exactly three or more digits, an underscore, a name, and the .up.sql suffix.
var migrationFileName = regexp.MustCompile(`^([0-9]{3,})_([A-Za-z0-9_]+)\.up\.sql$`)

// requiredTables is the post-apply effect check (R18-03 non-vacuity): a run
// that records versions but creates no tables is a failure, not a success.
var requiredTables = []string{
	"prevention_rules",
	"failure_registry",
	"file_reads",
	"task_attempts",
	"uncertainty_tracking",
	"production_code_tracking",
}

type migration struct {
	version int
	name    string
	file    string // absolute path
}

func main() {
	log.SetFlags(0)

	verifyOnly := false
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--verify":
			verifyOnly = true
		case "--help", "-h":
			usage()
			return
		default:
			// A DSN on the command line is forbidden (R18-04/Sec. 4.1). Refuse
			// loudly rather than risk leaking a credential into argv/logs.
			log.Fatalf("refusing argument %q: the database URL must come from MIGRATION_DSN_FILE or MIGRATION_DATABASE_URL, never argv", arg)
		}
	}

	dsn, source, err := resolveDSN()
	if err != nil {
		log.Fatalf("migration job: %v", err)
	}
	_ = source // never logged

	migrationsDir, err := resolveMigrationsDir()
	if err != nil {
		log.Fatalf("migration job: %v", err)
	}

	manifest, err := loadManifest(migrationsDir)
	if err != nil {
		log.Fatalf("migration manifest invalid: %v", err)
	}
	log.Printf("manifest: %d migrations (%03d..%03d) from %s", len(manifest), manifest[0].version, manifest[len(manifest)-1].version, migrationsDir)

	if verifyOnly {
		if err := runVerify(dsn, manifest); err != nil {
			log.Fatalf("verify failed: %v", err)
		}
		log.Println("verify OK: manifest valid and recorded checksums match")
		return
	}

	if err := runApply(dsn, manifest); err != nil {
		log.Fatalf("migration failed: %v", err)
	}
	log.Println("migrations applied and verified successfully")
}

func usage() {
	fmt.Fprint(os.Stderr, `usage: run_migrations [--verify]

Applies pending numbered migrations from internal/database/migrations in
ascending order under a bounded advisory lock. --verify validates the manifest
and recorded checksums without mutating the database.

The database URL is read from MIGRATION_DSN_FILE (a 0600 file, preferred) or
MIGRATION_DATABASE_URL. It is never accepted on the command line.
`)
}

// resolveDSN returns the database URL from a protected secret source.
func resolveDSN() (dsn, source string, err error) {
	if path := strings.TrimSpace(os.Getenv("MIGRATION_DSN_FILE")); path != "" {
		info, statErr := os.Stat(path)
		if statErr != nil {
			return "", "", fmt.Errorf("MIGRATION_DSN_FILE %q is not readable: %w", path, statErr)
		}
		if info.Mode().Perm()&0o077 != 0 {
			return "", "", fmt.Errorf("MIGRATION_DSN_FILE %q is group/world-accessible (mode %04o); require 0600", path, info.Mode().Perm())
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return "", "", fmt.Errorf("cannot read MIGRATION_DSN_FILE %q: %w", path, readErr)
		}
		v := strings.TrimSpace(string(raw))
		if v == "" {
			return "", "", fmt.Errorf("MIGRATION_DSN_FILE %q is empty", path)
		}
		return v, "file", nil
	}

	if v := strings.TrimSpace(os.Getenv("MIGRATION_DATABASE_URL")); v != "" {
		return v, "env", nil
	}

	return "", "", errors.New("no database URL: set MIGRATION_DSN_FILE (preferred) or MIGRATION_DATABASE_URL")
}

// resolveMigrationsDir finds the migrations directory regardless of CWD.
func resolveMigrationsDir() (string, error) {
	candidates := []string{defaultMigrationsDir, filepath.Join("mcp-server", defaultMigrationsDir)}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			abs, _ := filepath.Abs(c)
			return abs, nil
		}
	}
	return "", fmt.Errorf("migrations directory %q not found (run from the repository root or mcp-server/)", defaultMigrationsDir)
}

// loadManifest reads, validates and orders the migration manifest (R18-02).
func loadManifest(dir string) ([]migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("cannot read migrations directory: %w", err)
	}

	byVersion := map[int]migration{}
	downSeen := map[int]bool{}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".down.sql") {
			base := strings.TrimSuffix(name, ".down.sql")
			if v, ok := parseLeadingVersion(base); ok {
				downSeen[v] = true
			}
			continue
		}
		m := migrationFileName.FindStringSubmatch(name)
		if m == nil {
			if strings.HasSuffix(name, ".up.sql") {
				return nil, fmt.Errorf("invalid migration file name %q: expected NNN_name.up.sql", name)
			}
			// Other .sql files (e.g. verify_tables.sql) are helpers, not migrations.
			continue
		}
		version, convErr := strconv.Atoi(m[1])
		if convErr != nil {
			return nil, fmt.Errorf("invalid version in %q: %w", name, convErr)
		}
		if _, dup := byVersion[version]; dup {
			return nil, fmt.Errorf("duplicate migration version %d", version)
		}
		byVersion[version] = migration{version: version, name: m[2], file: filepath.Join(dir, name)}
	}

	if len(byVersion) == 0 {
		return nil, errors.New("no migration files found: an empty manifest is a failure, not a no-op")
	}

	versions := make([]int, 0, len(byVersion))
	for v := range byVersion {
		versions = append(versions, v)
	}
	sort.Ints(versions)

	// Release policy: contiguous ascending versions with a paired down file.
	expected := 1
	for _, v := range versions {
		if v != expected {
			return nil, fmt.Errorf("manifest gap: expected version %d but found %d", expected, v)
		}
		if !downSeen[v] {
			return nil, fmt.Errorf("migration %03d has no paired .down.sql (rollback script missing)", v)
		}
		expected++
	}

	ordered := make([]migration, 0, len(versions))
	for _, v := range versions {
		ordered = append(ordered, byVersion[v])
	}
	return ordered, nil
}

func parseLeadingVersion(base string) (int, bool) {
	digits := base
	if idx := strings.IndexByte(base, '_'); idx >= 0 {
		digits = base[:idx]
	}
	v, err := strconv.Atoi(strings.TrimSpace(digits))
	if err != nil {
		return 0, false
	}
	return v, true
}

func fileChecksum(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("cannot read migration %q: %w", path, err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("cannot open database: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("cannot reach database: %w", err)
	}
	return db, nil
}

// ensureLedger creates the authoritative ledger if absent. The shape is stable:
// a numeric version, a name, a SHA-256 checksum and the application time.
// A pre-existing (legacy) ledger is migrated in place by adding the columns
// this job records, so an upgrade never loses history.
func ensureLedger(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT PRIMARY KEY,
			name       TEXT,
			checksum   TEXT,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			dirty      BOOLEAN NOT NULL DEFAULT FALSE
		)`)
	if err != nil {
		return fmt.Errorf("cannot create schema_migrations ledger: %w", err)
	}
	// Legacy ledgers used a descriptive text version and lacked the extra
	// columns; add them without destroying rows.
	for _, ddl := range []string{
		`ALTER TABLE schema_migrations ADD COLUMN IF NOT EXISTS name TEXT`,
		`ALTER TABLE schema_migrations ADD COLUMN IF NOT EXISTS checksum TEXT`,
		`ALTER TABLE schema_migrations ADD COLUMN IF NOT EXISTS applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`,
		`ALTER TABLE schema_migrations ADD COLUMN IF NOT EXISTS dirty BOOLEAN NOT NULL DEFAULT FALSE`,
	} {
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			return fmt.Errorf("cannot normalize schema_migrations ledger: %w", err)
		}
	}
	return nil
}

// readLedger returns the applied versions keyed by parsed numeric version,
// with the recorded checksum (empty for legacy rows).
func readLedger(ctx context.Context, db *sql.DB) (map[int]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT version, COALESCE(checksum, '') FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("cannot read schema_migrations ledger: %w", err)
	}
	defer rows.Close()

	applied := map[int]string{}
	for rows.Next() {
		var raw, checksum string
		if err := rows.Scan(&raw, &checksum); err != nil {
			return nil, fmt.Errorf("cannot scan schema_migrations row: %w", err)
		}
		v, ok := parseLeadingVersion(raw)
		if !ok {
			return nil, fmt.Errorf("malformed schema version %q in ledger", raw)
		}
		if prev, dup := applied[v]; dup && prev != checksum {
			return nil, fmt.Errorf("ambiguous ledger: version %d recorded more than once", v)
		}
		applied[v] = checksum
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("schema_migrations ledger read failed: %w", err)
	}
	return applied, nil
}

func acquireLock(ctx context.Context, db *sql.DB) (release func(), err error) {
	lockCtx, cancel := context.WithTimeout(ctx, lockTimeout)
	defer cancel()

	// A dedicated connection holds the session-level advisory lock.
	conn, err := db.Conn(lockCtx)
	if err != nil {
		return nil, fmt.Errorf("cannot acquire a connection for the advisory lock: %w", err)
	}
	if _, err := conn.ExecContext(lockCtx, `SELECT pg_advisory_lock($1)`, migrationLockKey); err != nil {
		conn.Close()
		return nil, fmt.Errorf("could not acquire the migration advisory lock within %s (another job may be running): %w", lockTimeout, err)
	}
	return func() {
		if _, err := conn.ExecContext(context.Background(), `SELECT pg_advisory_unlock($1)`, migrationLockKey); err != nil {
			log.Printf("warning: could not release the migration advisory lock: %v", err)
		}
		conn.Close()
	}, nil
}

// runVerify validates the manifest and recorded checksums without mutating
// schema state (it may create/normalize the ledger table only to read it).
func runVerify(dsn string, manifest []migration) error {
	db, err := openDB(dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	ctx := context.Background()
	if err := ensureLedger(ctx, db); err != nil {
		return err
	}
	applied, err := readLedger(ctx, db)
	if err != nil {
		return err
	}
	if len(applied) == 0 {
		return errors.New("ledger is empty: the migration job has never been applied to this database")
	}
	for _, m := range manifest {
		stored, ok := applied[m.version]
		if !ok {
			return fmt.Errorf("version %03d (%s) is not recorded in the ledger", m.version, m.name)
		}
		if stored == "" {
			log.Printf("warning: version %03d has no recorded checksum (legacy row); cannot verify content", m.version)
			continue
		}
		sum, err := fileChecksum(m.file)
		if err != nil {
			return err
		}
		if sum != stored {
			return fmt.Errorf("checksum mismatch for version %03d: recorded %s but file is %s", m.version, stored, sum)
		}
	}
	return nil
}

func runApply(dsn string, manifest []migration) error {
	db, err := openDB(dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	ctx := context.Background()

	release, err := acquireLock(ctx, db)
	if err != nil {
		return err
	}
	defer release()

	// Re-read state AFTER acquiring the lock so two jobs cannot act on stale
	// state (R18-04).
	if err := ensureLedger(ctx, db); err != nil {
		return err
	}
	applied, err := readLedger(ctx, db)
	if err != nil {
		return err
	}

	appliedCount := 0
	for _, m := range manifest {
		sum, err := fileChecksum(m.file)
		if err != nil {
			return err
		}
		if stored, ok := applied[m.version]; ok {
			if stored == "" {
				log.Printf("version %03d already recorded (legacy row, checksum unverifiable); skipping", m.version)
				continue
			}
			if stored != sum {
				return fmt.Errorf("checksum mismatch for applied version %03d: recorded %s but file is %s; refusing to auto-repair", m.version, stored, sum)
			}
			continue
		}
		if err := applyOne(ctx, db, m, sum); err != nil {
			return err
		}
		appliedCount++
	}

	if appliedCount == 0 {
		log.Println("no pending migrations; ledger already current")
	}
	return verifyTables(ctx, db)
}

// applyOne executes one migration and records its ledger row in the SAME
// transaction (R18-03). Any failure rolls back both the schema change and the
// ledger row, leaving the version unapplied.
func applyOne(ctx context.Context, db *sql.DB, m migration, checksum string) error {
	content, err := os.ReadFile(m.file)
	if err != nil {
		return fmt.Errorf("cannot read migration %03d: %w", m.version, err)
	}
	if len(strings.TrimSpace(string(content))) == 0 {
		return fmt.Errorf("migration %03d is empty: refusing to record a vacuous migration", m.version)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("cannot begin transaction for %03d: %w", m.version, err)
	}
	defer tx.Rollback()

	log.Printf("applying %03d_%s", m.version, m.name)
	if _, err := tx.ExecContext(ctx, string(content)); err != nil {
		return fmt.Errorf("migration %03d failed: %w", m.version, err)
	}

	version := fmt.Sprintf("%03d", m.version)
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO schema_migrations (version, name, checksum, dirty) VALUES ($1, $2, $3, FALSE)`,
		version, m.name, checksum); err != nil {
		return fmt.Errorf("cannot record migration %03d: %w", m.version, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("cannot commit migration %03d: %w", m.version, err)
	}
	return nil
}

// verifyTables is the effect check: the recorded schema must actually exist.
func verifyTables(ctx context.Context, db *sql.DB) error {
	var missing []string
	for _, table := range requiredTables {
		var exists bool
		err := db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.tables
				WHERE table_schema = 'public' AND table_name = $1
			)`, table).Scan(&exists)
		if err != nil {
			return fmt.Errorf("cannot check table %s: %w", table, err)
		}
		if !exists {
			missing = append(missing, table)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("post-apply verification failed: required table(s) missing: %s", strings.Join(missing, ", "))
	}
	log.Printf("verified %d required tables exist", len(requiredTables))
	return nil
}
