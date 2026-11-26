package web

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/thearchitectit/guardrail-mcp/internal/audit"
	"github.com/thearchitectit/guardrail-mcp/internal/config"
	"github.com/thearchitectit/guardrail-mcp/internal/database"
)

func TestValidateSelectionPayload(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantText  string
		wantError string
	}{
		{name: "canonical selection", body: `{"selection":"  x := 1\n","language":"go"}`, wantText: "  x := 1\n"},
		{name: "legacy code", body: `{"code":"old client","file_path":"a.go"}`, wantText: "old client"},
		{name: "equal dual fields", body: `{"selection":"same","code":"same"}`, wantText: "same"},
		{name: "conflicting dual fields", body: `{"selection":"new","code":"old"}`, wantError: "selection and code must match"},
		{name: "missing", body: `{}`, wantError: "selection is required"},
		{name: "empty canonical field", body: `{"selection":""}`, wantError: "selection is required"},
		{name: "explicit null is present", body: `{"selection":null,"code":"legacy"}`, wantError: "selection and code must match"},
		{name: "wrong selection type", body: `{"selection":7}`, wantError: "invalid request body"},
		{name: "wrong legacy type", body: `{"code":false}`, wantError: "invalid request body"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fields selectionPayloadFields
			if err := json.Unmarshal([]byte(tt.body), &fields); err != nil {
				t.Fatalf("decode fixture: %v", err)
			}
			got, gotError := resolveSelectionPayload(fields)
			if gotError != tt.wantError {
				t.Fatalf("error = %q, want %q", gotError, tt.wantError)
			}
			if got.Selection != tt.wantText {
				t.Fatalf("selection = %q, want exact source %q", got.Selection, tt.wantText)
			}
			if tt.name == "canonical selection" && got.Language != "go" {
				t.Fatalf("language = %q, want preserved request field", got.Language)
			}
			if tt.name == "legacy code" && got.FilePath != "a.go" {
				t.Fatalf("file_path = %q, want preserved request field", got.FilePath)
			}
		})
	}
}

func TestValidateSelectionRouteAgainstDisposablePostgres(t *testing.T) {
	databaseURL := os.Getenv("IDE_SELECTION_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set IDE_SELECTION_TEST_DATABASE_URL to a disposable ide_selection_test database")
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open disposable selection database: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	var databaseName string
	if err := db.QueryRow("SELECT current_database()").Scan(&databaseName); err != nil {
		t.Fatalf("identify selection fixture database: %v", err)
	}
	if databaseName != "ide_selection_test" {
		t.Fatalf("refusing non-fixture database %q", databaseName)
	}
	_, err = db.Exec(`CREATE TEMP TABLE prevention_rules (
		id UUID NOT NULL, rule_id TEXT NOT NULL, name TEXT NOT NULL,
		pattern TEXT NOT NULL, pattern_hash TEXT NOT NULL, message TEXT NOT NULL,
		severity TEXT NOT NULL, enabled BOOLEAN NOT NULL, document_id UUID NOT NULL,
		category TEXT NOT NULL, created_at TIMESTAMP NOT NULL, updated_at TIMESTAMP NOT NULL
	)`)
	if err != nil {
		t.Fatalf("create synthetic selection rule fixture: %v", err)
	}
	_, err = db.Exec(`INSERT INTO prevention_rules
		(id, rule_id, name, pattern, pattern_hash, message, severity, enabled, document_id, category, created_at, updated_at)
		VALUES
		('00000000-0000-4000-8000-000000000001', 'IDE-03-TEST', 'Synthetic selection fixture',
		 'selection_probe', '', 'synthetic selection matched', 'error', true,
		 '00000000-0000-4000-8000-000000000002', '', NOW(), NOW())`)
	if err != nil {
		t.Fatalf("insert synthetic selection rule fixture: %v", err)
	}

	e := echo.New()
	logger := audit.NewLogger(4)
	t.Cleanup(logger.Stop)
	server := &Server{
		ruleStore:   database.NewRuleStore(&database.DB{DB: db}),
		auditLogger: logger,
		cfg:         &config.Config{WebEnabled: false},
	}
	registerRoutes(e, server)
	selectedText := "  selection_probe\n"
	for _, tc := range []struct{ name, body string }{
		{"canonical selection", `{"selection":"  selection_probe\n","language":"go"}`},
		{"legacy code only", `{"code":"  selection_probe\n","language":"go"}`},
		{"equal dual fields", `{"selection":"  selection_probe\n","code":"  selection_probe\n","language":"go"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/ide/validate/selection", strings.NewReader(tc.body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("POST /ide/validate/selection status = %d, want 200: %s", rec.Code, rec.Body.String())
			}
			var response struct {
				Violations []ValidationViolation `json:"violations"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
				t.Fatalf("decode selection response: %v", err)
			}
			if len(response.Violations) != 1 || response.Violations[0].Match != strings.TrimSpace(selectedText) {
				t.Fatalf("violations = %#v; want exact fixture match %q", response.Violations, strings.TrimSpace(selectedText))
			}
		})
	}
	requestDir := os.Getenv("IDE_SELECTION_CLIENT_REQUEST_DIR")
	if requestDir != "" {
		entries, err := os.ReadDir(requestDir)
		if err != nil {
			t.Fatalf("read captured client requests: %v", err)
		}
		clientFiles := []string{
			"client-vscode.json",
			"client-jetbrains.json",
			"client-vim.json",
			"client-neovim.json",
		}
		expectedFiles := make(map[string]bool, len(clientFiles))
		for _, name := range clientFiles {
			expectedFiles[name] = true
		}
		foundFiles := make(map[string]bool, len(clientFiles))
		for _, entry := range entries {
			if filepath.Ext(entry.Name()) != ".json" {
				continue
			}
			if entry.IsDir() || !expectedFiles[entry.Name()] {
				t.Fatalf("unexpected captured client request entry %q", entry.Name())
			}
			foundFiles[entry.Name()] = true
		}
		for _, name := range clientFiles {
			if !foundFiles[name] {
				t.Fatalf("missing captured client request %q", name)
			}
		}
		for _, name := range clientFiles {
			body, err := os.ReadFile(filepath.Join(requestDir, name))
			if err != nil {
				t.Fatalf("read captured client request %s: %v", name, err)
			}
			t.Run("captured client "+strings.TrimSuffix(name, ".json"), func(t *testing.T) {
				req := httptest.NewRequest(http.MethodPost, "/ide/validate/selection", strings.NewReader(string(body)))
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				rec := httptest.NewRecorder()
				e.ServeHTTP(rec, req)
				if rec.Code != http.StatusOK {
					t.Fatalf("POST /ide/validate/selection status = %d, want 200: %s", rec.Code, rec.Body.String())
				}
				var response struct {
					Violations []ValidationViolation `json:"violations"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
					t.Fatalf("decode selection response: %v", err)
				}
				if len(response.Violations) != 1 || response.Violations[0].Match != strings.TrimSpace(selectedText) {
					t.Fatalf("violations = %#v; want exact fixture match %q", response.Violations, strings.TrimSpace(selectedText))
				}
			})
		}
	}
}
