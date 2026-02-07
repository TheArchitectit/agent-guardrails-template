package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/thearchitectit/guardrail-mcp/internal/audit"
	"github.com/thearchitectit/guardrail-mcp/internal/cache"
	"github.com/thearchitectit/guardrail-mcp/internal/config"
	"github.com/thearchitectit/guardrail-mcp/internal/database"
	"github.com/thearchitectit/guardrail-mcp/internal/web"
)

func main() {
	// Setup structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("Starting Guardrail MCP Server")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Set log level based on config
	setLogLevel(cfg.LogLevel)

	// Initialize audit logger
	auditLogger := audit.NewLogger(1000)

	// Connect to database
	db, err := database.New(cfg)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Connect to Redis
	redisClient, err := cache.New(cfg)
	if err != nil {
		slog.Error("Failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	// Create web server
	webServer := web.NewServer(cfg, db, redisClient, auditLogger)

	// Start servers
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start web server
	go func() {
		addr := fmt.Sprintf("127.0.0.1:%d", cfg.WebPort)
		slog.Info("Starting web server", "addr", addr)
		if err := webServer.Start(addr); err != nil && err != http.ErrServerClosed {
			slog.Error("Web server error", "error", err)
			cancel()
		}
	}()

	// Start MCP server
	go func() {
		addr := fmt.Sprintf("127.0.0.1:%d", cfg.MCPPort)
		slog.Info("Starting MCP server", "addr", addr)
		// TODO: Start MCP SSE server
		// mcpServer := mcp.NewServer(cfg, db, redisClient, auditLogger)
		// if err := mcpServer.Start(addr); err != nil && err != http.ErrServerClosed {
		//     slog.Error("MCP server error", "error", err)
		//     cancel()
		// }
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	select {
	case <-quit:
		slog.Info("Shutdown signal received")
	case <-ctx.Done():
		slog.Info("Context cancelled")
	}

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := webServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("Web server shutdown error", "error", err)
	}

	slog.Info("Server stopped")
}

func setLogLevel(level string) {
	var slogLevel slog.Level
	switch level {
	case "debug":
		slogLevel = slog.LevelDebug
	case "info":
		slogLevel = slog.LevelInfo
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slogLevel,
	}))
	slog.SetDefault(logger)
}
