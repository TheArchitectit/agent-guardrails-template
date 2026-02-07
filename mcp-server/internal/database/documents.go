package database

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/thearchitectit/guardrail-mcp/internal/models"
)

// DocumentStore handles document database operations
type DocumentStore struct {
	db *DB
}

// NewDocumentStore creates a new document store
func NewDocumentStore(db *DB) *DocumentStore {
	return &DocumentStore{db: db}
}

// GetByID retrieves a document by ID
func (s *DocumentStore) GetByID(ctx context.Context, id uuid.UUID) (*models.Document, error) {
	var doc models.Document
	err := s.db.QueryRowContext(ctx, `
		SELECT id, slug, title, content, category, path, version, metadata, created_at, updated_at
		FROM documents
		WHERE id = $1
	`, id).Scan(
		&doc.ID, &doc.Slug, &doc.Title, &doc.Content, &doc.Category,
		&doc.Path, &doc.Version, &doc.Metadata, &doc.CreatedAt, &doc.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("document not found: %s", id)
		}
		return nil, err
	}
	return &doc, nil
}

// GetBySlug retrieves a document by slug
func (s *DocumentStore) GetBySlug(ctx context.Context, slug string) (*models.Document, error) {
	var doc models.Document
	err := s.db.QueryRowContext(ctx, `
		SELECT id, slug, title, content, category, path, version, metadata, created_at, updated_at
		FROM documents
		WHERE slug = $1
	`, slug).Scan(
		&doc.ID, &doc.Slug, &doc.Title, &doc.Content, &doc.Category,
		&doc.Path, &doc.Version, &doc.Metadata, &doc.CreatedAt, &doc.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("document not found: %s", slug)
		}
		return nil, err
	}
	return &doc, nil
}

// List retrieves documents with pagination
func (s *DocumentStore) List(ctx context.Context, category string, limit, offset int) ([]models.Document, error) {
	var args []interface{}
	query := `
		SELECT id, slug, title, content, category, path, version, metadata, created_at, updated_at
		FROM documents
		WHERE 1=1
	`

	if category != "" {
		query += ` AND category = $1`
		args = append(args, category)
	}

	query += ` ORDER BY updated_at DESC LIMIT $2 OFFSET $3`
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []models.Document
	for rows.Next() {
		var doc models.Document
		err := rows.Scan(
			&doc.ID, &doc.Slug, &doc.Title, &doc.Content, &doc.Category,
			&doc.Path, &doc.Version, &doc.Metadata, &doc.CreatedAt, &doc.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}

	return docs, rows.Err()
}

// Search performs full-text search on documents
func (s *DocumentStore) Search(ctx context.Context, query string, limit int) ([]models.Document, error) {
	// Validate and sanitize query first
	safeQuery, err := sanitizeSearchQuery(query)
	if err != nil {
		return nil, fmt.Errorf("invalid search query: %w", err)
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, slug, title, content, category, path, version, metadata, created_at, updated_at
		FROM documents
		WHERE search_vector @@ plainto_tsquery('english', $1)
		ORDER BY ts_rank(search_vector, plainto_tsquery('english', $1)) DESC
		LIMIT $2
	`, safeQuery, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []models.Document
	for rows.Next() {
		var doc models.Document
		err := rows.Scan(
			&doc.ID, &doc.Slug, &doc.Title, &doc.Content, &doc.Category,
			&doc.Path, &doc.Version, &doc.Metadata, &doc.CreatedAt, &doc.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}

	return docs, rows.Err()
}

// Create inserts a new document
func (s *DocumentStore) Create(ctx context.Context, doc *models.Document) error {
	return s.db.QueryRowContext(ctx, `
		INSERT INTO documents (slug, title, content, category, path, version, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`, doc.Slug, doc.Title, doc.Content, doc.Category, doc.Path, doc.Version, doc.Metadata,
	).Scan(&doc.ID, &doc.CreatedAt, &doc.UpdatedAt)
}

// Update updates an existing document
func (s *DocumentStore) Update(ctx context.Context, doc *models.Document) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE documents
		SET title = $1, content = $2, category = $3, path = $4, version = version + 1, metadata = $5, updated_at = NOW()
		WHERE id = $6
	`, doc.Title, doc.Content, doc.Category, doc.Path, doc.Metadata, doc.ID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("document not found: %s", doc.ID)
	}

	return nil
}

// Delete removes a document
func (s *DocumentStore) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM documents WHERE id = $1`, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("document not found: %s", id)
	}

	return nil
}

// sanitizeSearchQuery validates and sanitizes search queries
func sanitizeSearchQuery(query string) (string, error) {
	// Limit length
	if len(query) > 200 {
		return "", fmt.Errorf("query too long (max 200 chars)")
	}

	// Remove dangerous characters - only allow safe FTS operators
	// Allow: alphanumeric, spaces, - (negation), * (prefix), " (phrase), & | (AND/OR)
	safe := regexp.MustCompile(`[^a-zA-Z0-9\s\-\*"&\|]`)
	cleaned := safe.ReplaceAllString(query, "")

	// Prevent FTS operator injection
	if strings.Count(cleaned, "(") != strings.Count(cleaned, ")") {
		return "", fmt.Errorf("mismatched parentheses")
	}

	return cleaned, nil
}
