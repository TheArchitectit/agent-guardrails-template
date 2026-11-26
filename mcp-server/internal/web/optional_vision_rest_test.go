package web

import (
	"context"
	"database/sql"
	"encoding/json"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	_ "github.com/mattn/go-sqlite3"
	"github.com/thearchitectit/guardrail-mcp/internal/api"
	"github.com/thearchitectit/guardrail-mcp/internal/audit"
	"github.com/thearchitectit/guardrail-mcp/internal/auth"
	"github.com/thearchitectit/guardrail-mcp/internal/config"
	"github.com/thearchitectit/guardrail-mcp/internal/database"
	"github.com/thearchitectit/guardrail-mcp/internal/mcp"
	"github.com/thearchitectit/guardrail-mcp/internal/vision"
)

const (
	visionWriterCredential = "synthetic-vision-writer-credential"
	visionReaderCredential = "synthetic-vision-reader-credential"
	visionVerifierKey      = "tpl52-local-test-verifier-key-0123456789"
)

type visionMemoryAuditStore struct {
	mu     sync.Mutex
	events chan database.AuditEvent
}

func newVisionMemoryAuditStore() *visionMemoryAuditStore {
	return &visionMemoryAuditStore{events: make(chan database.AuditEvent, 16)}
}

func (s *visionMemoryAuditStore) Insert(_ context.Context, event *database.AuditEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events <- *event
	return nil
}

type syntheticVisionClient struct {
	mu    sync.Mutex
	calls int
}

func (c *syntheticVisionClient) ReviewImage(context.Context, string, string, []vision.Finding) (*vision.ReviewResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls++
	return &vision.ReviewResponse{
		Findings:    []vision.Finding{{Category: "fixture", Severity: "low", Description: "synthetic finding"}},
		Confidence:  1,
		RawText:     "synthetic provider response",
		ModelUsed:   "fixture-model",
		BackendUsed: "local-fixture",
	}, nil
}

func (c *syntheticVisionClient) HealthCheck(context.Context) (vision.HealthStatus, error) {
	return vision.HealthStatus{Backend: "local-fixture", Healthy: true}, nil
}

func (c *syntheticVisionClient) Calls() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls
}

func optionalVisionRegistryConfig(t *testing.T) *config.Config {
	t.Helper()
	now := time.Now()
	records := []auth.Record{
		{
			CredentialID: "fixture-writer-credential-id",
			PrincipalID:  "fixture-writer",
			Scopes:       []string{auth.ScopeRESTRead, auth.ScopeRESTWrite},
			Role:         auth.RoleDeveloper,
			Resources:    []string{"*"},
			IssuedAt:     now.Add(-time.Hour),
			ExpiresAt:    now.Add(time.Hour),
			Verifier:     auth.Digest(visionVerifierKey, visionWriterCredential),
		},
		{
			CredentialID: "fixture-reader-credential-id",
			PrincipalID:  "fixture-reader",
			Scopes:       []string{auth.ScopeRESTRead},
			Role:         auth.RoleDeveloper,
			Resources:    []string{"*"},
			IssuedAt:     now.Add(-time.Hour),
			ExpiresAt:    now.Add(time.Hour),
			Verifier:     auth.Digest(visionVerifierKey, visionReaderCredential),
		},
	}
	raw, err := json.Marshal(records)
	if err != nil {
		t.Fatal(err)
	}
	return &config.Config{
		SchemaVersion:          "1.0",
		MCPAPIKey:              "legacy-mcp-key",
		IDEAPIKey:              "legacy-ide-key",
		CredentialRegistryJSON: string(raw),
		CredentialVerifierKey:  visionVerifierKey,
	}
}

func postVisionJSON(t *testing.T, e *echo.Echo, path, credential string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(string(encoded)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+credential)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func getVision(t *testing.T, e *echo.Echo, path, credential string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+credential)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func nextVisionAudit(t *testing.T, store *visionMemoryAuditStore) database.AuditEvent {
	t.Helper()
	select {
	case event := <-store.events:
		return event
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for authorization audit event")
		return database.AuditEvent{}
	}
}

func TestOptionalVisionRegisteredRoutesAuthorizeEffectsAndRedactAudit(t *testing.T) {
	root := t.TempDir()
	imagePath := filepath.Join(root, "synthetic-local-image.png")
	file, err := os.Create(imagePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	storage, err := vision.NewStorage(filepath.Join(root, "vision.sqlite"))
	if err != nil {
		t.Fatalf("open isolated Vision SQLite fixture: %v", err)
	}
	t.Cleanup(func() { _ = storage.Close() })
	client := &syntheticVisionClient{}
	engine := vision.NewReviewEngine(client, storage, vision.ReviewEngineConfig{MaxIterations: 1})
	auditStore := newVisionMemoryAuditStore()
	auditLogger := audit.NewLoggerWithStore(16, auditStore)
	t.Cleanup(auditLogger.Stop)
	cfg := optionalVisionRegistryConfig(t)
	e := echo.New()
	e.Use(APIKeyAuth(cfg, auditLogger))
	api.NewVisionServer(engine, nil).RegisterRoutes(e.Group("/v1/vision"))

	create := postVisionJSON(t, e, "/v1/vision/review", visionWriterCredential, map[string]string{"image_path": imagePath})
	if create.Code != http.StatusOK {
		t.Fatalf("registered create route status=%d body=%s", create.Code, create.Body.String())
	}
	var report vision.Report
	if err := json.Unmarshal(create.Body.Bytes(), &report); err != nil {
		t.Fatalf("decode registered create response: %v", err)
	}
	if report.Review.ID == "" || client.Calls() != 1 {
		t.Fatalf("create did not produce a review and exactly one fixture call: id=%q calls=%d", report.Review.ID, client.Calls())
	}
	reviews, err := storage.ListReviews(10)
	if err != nil || len(reviews) != 1 {
		t.Fatalf("create effect missing in isolated SQLite: reviews=%d err=%v", len(reviews), err)
	}
	allowedCreateAudit := nextVisionAudit(t, auditStore)
	if allowedCreateAudit.Action != "POST /v1/vision/review" || allowedCreateAudit.Resource != "" {
		t.Fatalf("create audit action/resource not normalized: action=%q resource=%q", allowedCreateAudit.Action, allowedCreateAudit.Resource)
	}
	if allowedCreateAudit.Actor != "fixture-writer" || allowedCreateAudit.Status != audit.DecisionAllow {
		t.Fatalf("create audit did not record resolved principal/allow: actor=%q status=%q", allowedCreateAudit.Actor, allowedCreateAudit.Status)
	}
	serializedCreateAudit, _ := json.Marshal(allowedCreateAudit)
	if strings.Contains(string(serializedCreateAudit), imagePath) || strings.Contains(string(serializedCreateAudit), visionWriterCredential) {
		t.Fatal("create audit leaked the input path or bearer credential")
	}

	get := getVision(t, e, "/v1/vision/review/"+report.Review.ID, visionReaderCredential)
	if get.Code != http.StatusOK {
		t.Fatalf("registered report read status=%d body=%s", get.Code, get.Body.String())
	}
	allowedReadAudit := nextVisionAudit(t, auditStore)
	if allowedReadAudit.Action != "GET /v1/vision/review/:id" || allowedReadAudit.Resource != report.Review.ID {
		t.Fatalf("read audit is not normalized/scoped: action=%q resource=%q", allowedReadAudit.Action, allowedReadAudit.Resource)
	}

	beforeIterations, err := storage.ListIterations(report.Review.ID)
	if err != nil {
		t.Fatal(err)
	}
	beforeCalls := client.Calls()
	denied := postVisionJSON(t, e, "/v1/vision/review/"+report.Review.ID+"/iterate", visionReaderCredential, map[string]string{"synthetic_secret": "must-not-be-audited"})
	if denied.Code != http.StatusForbidden {
		t.Fatalf("read-only principal iteration status=%d body=%s, want 403", denied.Code, denied.Body.String())
	}
	afterIterations, err := storage.ListIterations(report.Review.ID)
	if err != nil {
		t.Fatal(err)
	}
	if client.Calls() != beforeCalls || len(afterIterations) != len(beforeIterations) {
		t.Fatalf("denied iteration caused effects: calls %d->%d iterations %d->%d", beforeCalls, client.Calls(), len(beforeIterations), len(afterIterations))
	}
	deniedAudit := nextVisionAudit(t, auditStore)
	if deniedAudit.Action != "POST /v1/vision/review/:id/iterate" || deniedAudit.Resource != report.Review.ID || deniedAudit.Status != audit.DecisionDeny {
		t.Fatalf("denial audit is not normalized/scoped: action=%q resource=%q status=%q", deniedAudit.Action, deniedAudit.Resource, deniedAudit.Status)
	}
	serializedDeniedAudit, _ := json.Marshal(deniedAudit)
	if strings.Contains(string(serializedDeniedAudit), "must-not-be-audited") || strings.Contains(string(serializedDeniedAudit), visionReaderCredential) {
		t.Fatal("denial audit leaked request payload or bearer credential")
	}
}

func TestOptionalVisionDefaultProfileDoesNotMountRoutes(t *testing.T) {
	t.Setenv("VISION_ENABLED", "")
	cfg := &config.Config{SchemaVersion: "1.0", DeploymentProfile: "local", MCPAPIKey: "legacy-mcp-key"}
	sqlDB, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "server-fixture.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	mcpServer := mcp.NewMCPServer(cfg, &database.DB{DB: sqlDB}, nil, nil, nil, nil, nil, nil)
	if mcpServer.VisionTools() != nil {
		t.Fatal("unset VISION_ENABLED must keep optional Vision tools disabled")
	}
	e := echo.New()
	if toolSet := mcpServer.VisionTools(); toolSet != nil {
		api.NewVisionServer(toolSet.Engine(), toolSet.Watcher()).RegisterRoutes(e.Group("/v1/vision"))
	}
	for _, route := range e.Routes() {
		if strings.HasPrefix(route.Path, "/v1/vision/") {
			t.Fatalf("default-disabled profile mounted an optional Vision route: %s %s", route.Method, route.Path)
		}
	}
}
