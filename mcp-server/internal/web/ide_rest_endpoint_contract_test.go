package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/thearchitectit/guardrail-mcp/internal/config"
)

func TestIDERestEndpointContract(t *testing.T) {
	server := &Server{
		readiness: NewReadinessEvaluator("contract-test", readyDeps()),
		version:   "contract-test",
		cfg:       &config.Config{WebEnabled: false},
	}
	server.SetLifecycle(LifecycleReady)

	e := echo.New()
	registerRoutes(e, server)
	listener := httptest.NewServer(e)
	defer listener.Close()

	client := listener.Client()
	for _, testCase := range []struct {
		name      string
		lifecycle LifecycleState
		status    int
		ready     bool
	}{
		{name: "ready web listener", lifecycle: LifecycleReady, status: http.StatusOK, ready: true},
		{name: "starting control", lifecycle: LifecycleStarting, status: http.StatusServiceUnavailable, ready: false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			server.SetLifecycle(testCase.lifecycle)
			response, err := client.Get(listener.URL + "/health/ready")
			if err != nil {
				t.Fatalf("GET /health/ready through the web listener: %v", err)
			}
			defer response.Body.Close()

			if response.StatusCode != testCase.status {
				t.Fatalf("GET /health/ready returned %d, want %d", response.StatusCode, testCase.status)
			}

			var report ReadinessReport
			if err := json.NewDecoder(response.Body).Decode(&report); err != nil {
				t.Fatalf("decode readiness response: %v", err)
			}
			if report.Ready != testCase.ready {
				t.Fatalf("readiness response says ready=%v, want %v", report.Ready, testCase.ready)
			}
		})
	}
}
