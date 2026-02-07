package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"runtime"
	"time"

	"github.com/thearchitectit/guardrail-mcp/internal/config"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// DB wraps sql.DB with guardrail-specific operations
type DB struct {
	*sql.DB
}

// New creates a new database connection with connection pooling
func New(cfg *config.Config) (*DB, error) {
	db, err := sql.Open("pgx", cfg.DatabaseURL())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	// Scale based on CPU cores - need 50+ for 1000 sessions
	maxConns := 4 * runtime.NumCPU()
	if maxConns < 50 {
		maxConns = 50
	}

	db.SetMaxOpenConns(maxConns)              // 50+ for 1000 sessions
	db.SetMaxIdleConns(maxConns / 2)          // 25 idle
	db.SetConnMaxLifetime(15 * time.Minute)   // Longer for stability
	db.SetConnMaxIdleTime(5 * time.Minute)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	slog.Info("Database connected",
		"max_conns", maxConns,
		"host", cfg.DBHost,
		"database", cfg.DBName,
	)

	return &DB{db}, nil
}

// HealthCheck verifies database connectivity
func (db *DB) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return db.PingContext(ctx)
}

// Close gracefully closes the database connection
func (db *DB) Close() error {
	slog.Info("Closing database connection")
	return db.DB.Close()
}
