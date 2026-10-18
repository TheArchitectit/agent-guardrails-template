package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/thearchitectit/guardrail-mcp/internal/api"
	"github.com/thearchitectit/guardrail-mcp/internal/audit"
	"github.com/thearchitectit/guardrail-mcp/internal/auth"
	"github.com/thearchitectit/guardrail-mcp/internal/cache"
	"github.com/thearchitectit/guardrail-mcp/internal/config"
	"github.com/thearchitectit/guardrail-mcp/internal/controlplane"
	"github.com/thearchitectit/guardrail-mcp/internal/database"
	"github.com/thearchitectit/guardrail-mcp/internal/evidence"
	"github.com/thearchitectit/guardrail-mcp/internal/guardrails"
	mcpServer "github.com/thearchitectit/guardrail-mcp/internal/mcp"
	"github.com/thearchitectit/guardrail-mcp/internal/notifications"
	"github.com/thearchitectit/guardrail-mcp/internal/secretbox"
	"github.com/thearchitectit/guardrail-mcp/internal/team"
	"github.com/thearchitectit/guardrail-mcp/internal/validation"
	"github.com/thearchitectit/guardrail-mcp/internal/web"
)

// Version information - set by ldflags during build
var (
	version   = "v3.7.1"
	buildTime = "unknown"
	gitCommit = "unknown"
)

func main() {
	// CLI flags
	var (
		showVersion   = flag.Bool("version", false, "Show version information")
		showHealth    = flag.Bool("health-check", false, "Run health check and exit")
		healthTimeout = flag.Duration("health-timeout", 5*time.Second, "Health check timeout")
	)
	flag.Parse()

	// Show version and exit
	if *showVersion {
		fmt.Printf("Guardrail MCP Server\n")
		fmt.Printf("  Version:   %s\n", version)
		fmt.Printf("  Build Time: %s\n", buildTime)
		fmt.Printf("  Git Commit: %s\n", gitCommit)
		fmt.Printf("  Go Version: %s\n", runtime.Version())
		os.Exit(0)
	}

	// Health check mode for container health checks
	if *showHealth {
		if err := runHealthCheck(*healthTimeout); err != nil {
			fmt.Fprintf(os.Stderr, "Health check failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Health check passed")
		os.Exit(0)
	}

	// Load configuration first to get log level
	cfg, err := config.Load()
	if err != nil {
		// Use default logger for initial error
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Setup structured logging with configured level
	setLogLevel(cfg.LogLevel)

	slog.Info("Starting Guardrail MCP Server",
		"version", version,
		"build_time", buildTime,
		"git_commit", gitCommit,
		"config_schema", cfg.SchemaVersion,
	)

	// Effective deployment profile (Spec 17 R17-01): named, explicit, and
	// validated against the listener binds below.
	profile, err := config.NormalizeDeploymentProfile(cfg.DeploymentProfile)
	if err != nil {
		slog.Error("Invalid deployment profile", "error", err)
		os.Exit(1)
	}
	slog.Info("Deployment profile selected", "profile", profile)

	// Start pprof server if enabled (for debugging). Spec 17 R17-09: the
	// endpoint is loopback-only and time-boxed; it refuses to start without a
	// bounded window and stops serving once that window passes.
	if cfg.PProfEnabled {
		go startPProfServer(cfg)
	}

	// Initialize audit logger
	auditLogger := audit.NewLogger(1000)

	// Connect to database
	db, err := database.New(cfg)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Schema compatibility gate (Spec 18 R18-05/R18-07): refuse to start,
	// before opening any listener, when the database is below the schema
	// version this binary requires. The migration job owns schema changes;
	// startup never applies them.
	schemaCtx, schemaCancel := context.WithTimeout(context.Background(), cfg.DBConnectTimeout)
	if err := database.CheckSchemaCompatibility(schemaCtx, database.NewSQLSchemaLedger(db.DB), database.MinRequiredSchemaVersion); err != nil {
		schemaCancel()
		slog.Error("Startup aborted: database schema is not compatible with this binary; run the versioned migration job before starting the server", "error", err)
		os.Exit(1)
	}
	schemaCancel()

	// Start database metrics collector
	dbMetricsCollector := database.NewMetricsCollector(db, 15*time.Second)
	dbMetricsCollector.Start()
	defer dbMetricsCollector.Stop()

	// Connect to Redis
	redisClient, err := cache.New(cfg)
	if err != nil {
		slog.Error("Failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	// Create web server
	webServer := web.NewServer(cfg, db, redisClient, auditLogger, version)

	// Create validation engine
	ruleStore := database.NewRuleStore(db)
	fileReadStore := database.NewFileReadStore(db)
	taskAttemptStore := database.NewTaskAttemptStore(db)
	validationEngine := validation.NewValidationEngine(ruleStore, redisClient,
		validation.WithFileReadStore(fileReadStore),
		validation.WithTaskAttemptStore(taskAttemptStore),
	)

	// Create MCP server
	haltEventStore := database.NewHaltEventStore(db)
	mcpSrv := mcpServer.NewMCPServer(cfg, db, redisClient, auditLogger, validationEngine, fileReadStore, taskAttemptStore, haltEventStore)

	// Team data at rest (SEC-007 / Spec 17 R17-06). The key was documented in the
	// secret inventory and the encryption manager existed, but nothing ever
	// constructed it: team assignment JSON was written plaintext at 0644 while
	// the inventory claimed an AES-GCM key protected it. Build it here, once, and
	// stop at startup if a configured key cannot be used rather than writing
	// plaintext after logging a warning nobody reads.
	teamEncryption, err := team.NewEncryptionManager(cfg.TeamEncryptionKey)
	if err != nil {
		slog.Error("team encryption key is unusable", "error", err)
		os.Exit(3)
	}
	mcpSrv.SetTeamEncryption(teamEncryption)
	if teamEncryption.Enabled() {
		slog.Info("team data encrypted at rest", "key_source", teamKeySource(cfg))
	} else {
		slog.Warn("team data is NOT encrypted at rest: TEAM_ENCRYPTION_KEY is unset",
			"profile", cfg.DeploymentProfile,
			"detail", "assignments are written as plaintext JSON; set TEAM_ENCRYPTION_KEY (or TEAM_ENCRYPTION_KEY_FILE) to seal them")
	}

	// Share the process-wide credential registry with the MCP surface so a
	// revocation reloaded in place denies both surfaces (Spec 16 R16-09). A
	// configured-but-unloadable registry is nil here; the MCP Serve path then
	// fails closed at startup exactly as before.
	mcpSrv.SetCredentialRegistry(webServer.CredentialRegistry())

	// Initialize the guardrails engine. It is built UNCONDITIONALLY because the
	// L1 pattern-matching layer (Spec 01) needs no model backend: gating the
	// whole engine on OLLAMA_URL/OPENAI_API_KEY left prompt-injection detection
	// unavailable in every deployment without an LLM, even though the detector
	// shipped. The optional semantic backends are added when configured; the
	// content filter fails closed when none is, so an absent backend blocks
	// classification rather than reporting content safe (P5/D-18).
	ollamaURL := os.Getenv("OLLAMA_URL")
	engine := guardrails.NewEngine(nil, slog.Default())
	if ollamaURL != "" {
		model := os.Getenv("OLLAMA_MODEL")
		if model == "" {
			model = "llama-guard-3"
		}
		engine.AddContentFilterBackend(guardrails.NewLlamaGuardBackend(model, ollamaURL))
		slog.Info("Guardrails engine initialized", "backend", "llama-guard", "model", model)

		// Attach the SAME Ollama deployment as the L3 injection classifier.
		// Before this, NewOllamaClassifier had no caller anywhere in the tree:
		// L3 was unreachable in every deployment, and the pipeline fell back to
		// a no-op backend that answered "safe" for text it never evaluated.
		// The content filter and the classifier now share one configured
		// backend instead of the classifier being dead code.
		classifierCfg := guardrails.ClassifierConfig{
			Enabled:   true,
			Backend:   "llama-guard",
			ModelPath: model,
			Endpoint:  ollamaURL,
			Threshold: 0.7,
		}
		if err := engine.EnableInjectionClassifier(
			guardrails.NewOllamaClassifier(classifierCfg, slog.Default()), classifierCfg); err != nil {
			// Not fatal: L1 still runs and the result reports L3 as absent,
			// which is honest. Log loudly because the operator asked for it.
			slog.Error("failed to enable L3 injection classifier", "error", err)
		} else {
			slog.Info("Injection L3 classifier enabled", "backend", "ollama", "model", model)
		}
	}
	if cfg.OpenAIAPIKey != "" {
		engine.AddContentFilterBackend(guardrails.NewOpenAIModerationBackend(cfg.OpenAIAPIKey))
		slog.Info("Guardrails engine initialized", "backend", "openai-moderation")
	}
	mcpSrv.SetGuardrailsEngine(engine)

	// Multi-agent safety chains (Spec 04). Built from the SAME engine, so an
	// agent-output validation runs the injection pipeline and content filter
	// the rest of the server uses; a registry built from a second engine would
	// let two detectors disagree about the same text. Scope patterns come from
	// the deployment: an agent's declared scope is a property of the work, not
	// of the binary.
	multiAgentRegistry, err := guardrails.NewMultiAgentRegistry(engine, splitScopePatterns(os.Getenv("MULTI_AGENT_SCOPE_PATTERNS")))
	if err != nil {
		// Not fatal: the three tools then fail closed with "not configured"
		// rather than reporting a chain that never ran as passed.
		slog.Error("multi-agent safety chains unavailable", "error", err)
	} else {
		mcpSrv.SetMultiAgentRegistry(multiAgentRegistry)
		slog.Info("multi-agent safety chains initialized", "chains", multiAgentRegistry.Chains())
	}

	// Runtime policy-pack consumer (Spec 07 R07-4.4). The CI gate proves the
	// checked-in lock record is honest; it cannot make the running server obey
	// the pack. Before this block there was no runtime consumer at all: the
	// engine resolved policy from its own rule set, the effective digest was
	// exposed nowhere, and no enforcement decision was attributable to a pack
	// rule. Activation is candidate-first and atomic — a set that fails to
	// resolve or validate leaves the process reporting "not configured" rather
	// than silently serving an empty-but-active policy.
	if cfg.PolicyPacksDir != "" {
		policyPacks := guardrails.NewPolicyPackManager()

		// Spec 07 R07-4.3.4 — signature verification is enabled by CONFIGURING a
		// trusted key set, and the set is read out of band from a restricted
		// file. A configured but unreadable, malformed, empty or expired set is
		// a hard startup error: continuing would either skip verification
		// silently (turning an enabled control off) or fail every pack and look
		// like the packs were unsigned.
		activate := policyPacks.ActivateFrom
		if cfg.PolicyPackTrustedKeysFile != "" {
			trusted, terr := guardrails.LoadTrustedKeySet(cfg.PolicyPackTrustedKeysFile)
			if terr != nil {
				slog.Error("policy pack trusted key set is unusable; refusing to verify packs",
					"path", cfg.PolicyPackTrustedKeysFile, "error", terr)
				mcpSrv.SetPolicyPacksError(fmt.Errorf("policy pack trusted key set: %w", terr))
				goto policyPacksDone
			}
			slog.Info("policy pack signature verification enabled",
				"trusted_keys", trusted.Len(), "key_ids", trusted.KeyIDs())
			activate = func(dir string, known map[string]bool) error {
				return policyPacks.ActivateFromVerified(dir, known, trusted)
			}
		}

		if err := activate(cfg.PolicyPacksDir, mcpSrv.RegisteredToolNames()); err != nil {
			// Not fatal: refusing to start would make a policy typo an outage.
			// The server keeps running with no active pack, and the diagnostic
			// resource reports "failed" with the reason so the gap is visible
			// instead of looking like an enforced policy.
			slog.Error("policy pack set failed to resolve; no pack is active", "dir", cfg.PolicyPacksDir, "error", err)
			mcpSrv.SetPolicyPacks(policyPacks)
		} else {
			mcpSrv.SetPolicyPacks(policyPacks)
			active := policyPacks.Active()
			slog.Info("active policy set",
				"dir", cfg.PolicyPacksDir,
				"effective_digest", active.EffectiveDigest(),
				"resolver_version", active.ResolverVersion(),
				"packs", len(active.Packs()),
				"rules", len(active.Rules()),
			)
		}
	} else {
		slog.Info("POLICY_PACKS_DIR is not set; no policy pack is enforced (guardrail://policy-packs reports not_configured)")
	}
policyPacksDone:

	// Control-plane composition (Spec 08 R08-4.1). The manifest declares which
	// components this deployment claims to compose; it declares NO state, so
	// loading it cannot make anything look active. A manifest that fails to
	// resolve is reported through guardrail://control-plane as "failed" rather
	// than treated as "not configured" — the two are different conditions and an
	// operator needs to tell them apart. This is not fatal for the same reason
	// the policy pack is not: a manifest typo must not be an outage.
	if cfg.ControlPlaneManifest != "" {
		manifest, err := controlplane.LoadManifest(cfg.ControlPlaneManifest)
		if err != nil {
			slog.Error("control-plane manifest failed to resolve; nothing is composed under it",
				"path", cfg.ControlPlaneManifest, "error", err)
			mcpSrv.SetControlPlaneError(cfg.ControlPlaneManifest, err)
		} else {
			mcpSrv.SetControlPlane(manifest, cfg.ControlPlaneManifest)
			slog.Info("control-plane composition resolved",
				"path", cfg.ControlPlaneManifest,
				"manifest_digest", manifest.Digest(),
				"profile", manifest.Profile,
				"components", len(manifest.Components),
				"mandatory", len(manifest.MandatoryIDs()),
			)

			// Spec 08 R08-4.4.2 .. R08-4.4.4: adoption, promotion and rollback
			// posture. A state file is optional; when unset the posture is
			// in-memory and every start is observe-only, which is the safe
			// default but loses a reviewed promotion across a restart. A
			// malformed state file is logged and left observe-only rather than
			// fatal: the composition still runs, and the failure is reported
			// through the resource with the reason.
			gov, govErr := controlplane.OpenGovernanceWithManifest(manifest, cfg.ControlPlaneStateFile)
			if govErr != nil {
				slog.Error("control-plane governance state is unreadable; staying observe-only",
					"path", cfg.ControlPlaneStateFile, "error", govErr)
			} else {
				mcpSrv.SetGovernance(gov)
				slog.Info("control-plane governance loaded",
					"mode", gov.Mode(),
					"state_file", cfg.ControlPlaneStateFile,
					"records", len(gov.Records()),
				)
			}
		}
	} else {
		slog.Info("CONTROL_PLANE_MANIFEST is not set; no composition is evaluated (guardrail://control-plane reports not_configured)")
	}

	// Spec 08 R08-4.2.4: readiness must report per-mandatory-component state and
	// return not-ready when a mandatory component is unavailable. Installed only
	// when a composition is declared, so a deployment that never opted in is not
	// blocked by a requirement it did not take on — while a declared
	// composition, including one that failed to resolve, gates readiness.
	if cfg.ControlPlaneManifest != "" {
		webServer.SetControlPlaneProbe(func() *web.ControlPlaneReadiness {
			rd := mcpSrv.ControlPlaneReadiness(context.Background())
			if rd == nil {
				return nil
			}
			out := &web.ControlPlaneReadiness{
				Status: string(rd.Status),
				Ready:  rd.Ready,
				Unmet:  rd.Unmet,
			}
			out.Mandatory = make([]web.ControlPlaneComponent, 0, len(rd.Mandatory))
			for _, m := range rd.Mandatory {
				out.Mandatory = append(out.Mandatory, web.ControlPlaneComponent{
					ID:        m.ID,
					State:     string(m.State),
					Available: m.Available,
					Exercised: m.Exercised,
					Reason:    m.Reason,
				})
			}
			return out
		})
	}

	// Outbound evidence identity (Spec 15 gr-xp-02). The instance key is
	// deployment-supplied, because a key minted at startup would present a new
	// identity on every restart and no peer could hold a trust record for it.
	// Config.Validate already refused an unusable or (outside `local`)
	// missing key, so a failure here can only be a parse that Validate did not
	// see — refuse to start rather than serve artifacts with no provenance.
	if cfg.EvidencePrivateKey != "" {
		key, err := config.ParseEvidenceKey(cfg.EvidencePrivateKey)
		if err != nil {
			slog.Error("evidence signing key is unusable; refusing to start", "error", err)
			os.Exit(1)
		}
		signer, err := evidence.NewInstance(cfg.EvidenceProducerID, key)
		if err != nil {
			slog.Error("evidence signing identity is unusable; refusing to start", "error", err)
			os.Exit(1)
		}
		mcpSrv.SetEvidenceSigner(signer, cfg.EvidenceAudience)
		slog.Info("Outbound evidence signing enabled (Spec 15 gr-xp-02)",
			"producer_id", signer.ProducerID(), "audience", cfg.EvidenceAudience, "key_id", signer.KeyID())

		// Spec 08 R08-4.3.6: retain the acceptance evidence a promotion rests
		// on. Validate already refused a configured store with no window or,
		// outside `local`, no signing identity, so a failure here is a real
		// filesystem or policy problem and stops startup: a store that silently
		// did not open would promote a deployment whose evidence exists only in
		// the caller's memory.
		if cfg.ControlPlaneEvidenceDir != "" && mcpSrv.Governance() != nil {
			store, err := controlplane.NewEvidenceStore(controlplane.StoreConfig{
				Dir:        cfg.ControlPlaneEvidenceDir,
				Retention:  cfg.ControlPlaneEvidenceRetention,
				MaxRecords: cfg.ControlPlaneEvidenceMaxRecords,
				Signer:     signer,
				Audit: func(ev controlplane.Event) {
					slog.Info("control-plane evidence store",
						"action", ev.Action,
						"evaluation_id", ev.EvaluationID,
						"reason", ev.Reason,
						"age", ev.Age,
					)
					if mcpSrv.AuditLogger() != nil {
						mcpSrv.AuditLogger().LogConfigChange(context.Background(), auth.Caller{
							PrincipalID: "system", CredentialID: "control-plane", Role: auth.RoleAdministrator,
						}, "guardrail://control-plane/evidence", ev.Action)
					}
				},
			})
			if err != nil {
				slog.Error("control-plane evidence store is unusable; refusing to start", "error", err)
				os.Exit(1)
			}
			mcpSrv.Governance().SetEvidenceStore(store)
			mcpSrv.SetEvidenceStore(store)
			slog.Info("control-plane evidence retention enabled (Spec 08 R08-4.3.6)",
				"dir", cfg.ControlPlaneEvidenceDir,
				"retention", cfg.ControlPlaneEvidenceRetention.String(),
				"max_records", cfg.ControlPlaneEvidenceMaxRecords,
				"key_id", store.SignerKeyID())
		}
	} else {
		slog.Warn("EVIDENCE_PRIVATE_KEY is not set; MCP verdicts carry no signed provenance (permitted for the local profile only)")
	}

	if ollamaURL == "" && cfg.OpenAIAPIKey == "" {
		slog.Info("No semantic content-filter backend configured (OLLAMA_URL and OPENAI_API_KEY unset); " +
			"guardrail_classify_content and guardrail_check_policy will fail closed. " +
			"Prompt-injection detection (L1 pattern matching) is active regardless.")
	}

	// Initialize webhook notifications. Webhook signing material is encrypted
	// at rest (Spec 17 R17-06) when WEBHOOK_ENCRYPTION_KEY is configured; a
	// nonlocal deployment is required to set it by the config gate, so a nil
	// sealer can only occur on the local posture.
	webhookStore := database.NewWebhookStore(db)
	if key := cfg.WebhookEncryptionKey; key != "" {
		sealer, err := secretbox.New(key)
		if err != nil {
			slog.Error("webhook encryption key is unusable; refusing to start", "error", err)
			os.Exit(1)
		}
		webhookStore = webhookStore.WithSealer(sealer)
		slog.Info("Webhook signing material encrypted at rest (Spec 17 R17-06)")
	} else {
		slog.Warn("WEBHOOK_ENCRYPTION_KEY is not set; webhook signing material is stored unencrypted (permitted for the local profile only)")
	}
	webhookDispatcher := notifications.NewDispatcher(webhookStore)
	mcpSrv.SetWebhookStore(webhookStore)
	mcpSrv.SetWebhookDispatcher(webhookDispatcher)
	slog.Info("Webhook notifications initialized")

	// Register vision HTTP routes on the web server if vision is enabled
	if vt := mcpSrv.VisionTools(); vt != nil {
		visionGroup := webServer.Echo().Group("/v1/vision")
		visionServer := api.NewVisionServer(vt.Engine(), vt.Watcher())
		visionServer.RegisterRoutes(visionGroup)
		slog.Info("Vision HTTP API mounted", "prefix", "/v1/vision")
	}

	// Deployment profile binding gate (Spec 17 R17-01/R17-02): re-validate
	// the actual bind addresses against the selected profile immediately
	// before starting listeners; a mismatch exits nonzero with no app socket.
	if err := cfg.ValidateDeploymentBinding(cfg.WebBindAddr()); err != nil {
		slog.Error("Startup aborted: web listener bind is not valid for the deployment profile", "error", err)
		os.Exit(1)
	}
	if err := cfg.ValidateDeploymentBinding(cfg.MCPBindAddr()); err != nil {
		slog.Error("Startup aborted: MCP listener bind is not valid for the deployment profile", "error", err)
		os.Exit(1)
	}

	// Start servers
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Revocation reload loop (Spec 16 Sec. 4 / R16-09): re-read the configured
	// registry sources periodically and swap the validated snapshot in, so a
	// revoked credential is denied on both surfaces within
	// RevocationPropagationBound without a process restart. A failed reload
	// leaves the previous snapshot in force (never falls back to legacy).
	if reg := webServer.CredentialRegistry(); reg != nil {
		sources := auth.Sources{
			InlineJSON:      cfg.CredentialRegistryJSON,
			FilePath:        cfg.CredentialRegistryFile,
			VerifierKey:     cfg.CredentialVerifierKey,
			VerifierKeyFile: cfg.CredentialVerifierKeyFile,
		}
		// Operator audit (Spec 16 R16-09 / Spec 17 R17-10): record every
		// applied snapshot replacement or verifier-key rotation as an opaque
		// audit event (key fingerprint + counts, never key material), so an
		// operator can confirm a rotation was applied and reconstruct when it
		// happened. A no-op re-read emits nothing.
		reg.SetRotationObserver(func(ev auth.RotationEvent) {
			auditLogger.LogRotation(context.Background(), ev)
		})
		reg.StartReloadLoop(ctx, sources, auth.RevocationReloadInterval, func(err error) {
			slog.Warn("credential registry reload failed; previous validated snapshot remains in force", "error", err)
		})
		slog.Info("Credential registry reload loop started", "interval", auth.RevocationReloadInterval, "bound", auth.RevocationPropagationBound)
	}

	// Retention sweep loop (Spec 08 R08-4.3.6). A configured window that nothing
	// applies is a policy nobody enforces, so the store trims immediately and
	// then on an interval until shutdown. Written as a loop rather than a
	// startup call because a process that runs for weeks would otherwise keep
	// every record it ever produced while reporting a 48h window.
	if store := mcpSrv.EvidenceStore(); store != nil {
		store.StartRetentionLoop(ctx, controlplane.RetentionSweepInterval)
		slog.Info("control-plane evidence retention sweep started",
			"interval", controlplane.RetentionSweepInterval.String(),
			"retention", store.Retention())
	}

	// Lifecycle gate (Spec 18 Sec. 5.2 / R18-12): readiness stays non-ready until
	// every ordered check passes; only then is the READY state published.
	if report := webServer.EvaluateReadiness(); report.Ready {
		webServer.SetLifecycle(web.LifecycleReady)
		slog.Info("Readiness checks passed; instance is READY")
	} else {
		slog.Error("Readiness checks failed at startup; instance remains NOT_READY", "status", report.Status)
	}

	// Start web server
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("Web server goroutine panicked", "panic", r)
				cancel()
			}
		}()
		addr := cfg.WebBindAddr()
		slog.Info("Starting web server", "addr", addr)
		if err := webServer.Start(addr); err != nil && err != http.ErrServerClosed {
			slog.Error("Web server error", "error", err)
			cancel()
		}
	}()

	// Start MCP server
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("MCP server goroutine panicked", "panic", r)
				cancel()
			}
		}()
		addr := cfg.MCPBindAddr()
		slog.Info("Starting MCP server", "addr", addr)
		if err := mcpSrv.Start(addr); err != nil && err != http.ErrServerClosed {
			slog.Error("MCP server error", "error", err)
			cancel()
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	select {
	case sig := <-quit:
		slog.Info("Shutdown signal received", "signal", sig.String())
	case <-ctx.Done():
		slog.Info("Context cancelled")
	}

	// Graceful shutdown with configurable timeout
	shutdownTimeout := cfg.ShutdownTimeout
	if shutdownTimeout == 0 {
		shutdownTimeout = 30 * time.Second
	}

	slog.Info("Initiating graceful shutdown", "timeout", shutdownTimeout)

	// Mark readiness false before drain begins (Spec 18 Sec. 5.2 / R18-12): a
	// probe arriving during shutdown must see DRAINING, never READY.
	webServer.SetLifecycle(web.LifecycleDraining)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	// Shutdown web server
	if err := webServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("Web server shutdown error", "error", err)
	}

	// Shutdown MCP server
	if err := mcpSrv.Shutdown(shutdownCtx); err != nil {
		slog.Error("MCP server shutdown error", "error", err)
	}

	// Close database connection
	if err := db.Close(); err != nil {
		slog.Error("Database close error", "error", err)
	}

	// Close Redis connection
	if err := redisClient.Close(); err != nil {
		slog.Error("Redis close error", "error", err)
	}

	webServer.SetLifecycle(web.LifecycleStopped)
	slog.Info("Server stopped gracefully")
}

// runHealthCheck performs a health check against the local server
func runHealthCheck(timeout time.Duration) error {
	client := &http.Client{
		Timeout: timeout,
	}

	// Container probe contract (Spec 18 Sec. 5.2 / R18-08): the health-check mode
	// gates on readiness, not liveness alone. A live-but-unready instance must
	// fail the probe so orchestrators do not route traffic to it.
	webPort := os.Getenv("WEB_PORT")
	if webPort == "" {
		webPort = "8081"
	}

	resp, err := client.Get(fmt.Sprintf("http://localhost:%s/health/ready", webPort))
	if err != nil {
		return fmt.Errorf("readiness check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("readiness check returned status %d", resp.StatusCode)
	}

	return nil
}

// teamKeySource reports where the team at-rest key came from, without ever
// naming the value (Spec 17 R17-08). The distinction matters operationally: a
// file-sourced key is the production posture, an inline one is the migration
// path R17-06 restricts.
// splitScopePatterns parses MULTI_AGENT_SCOPE_PATTERNS into the forbidden
// actions the Four Laws scope check enforces.
//
// Comma-separated because the value is a flat list of short action names, not a
// document. Blank entries are returned as-is on purpose: the validator refuses
// them at construction (an empty pattern matches every output), which surfaces
// a malformed deployment at startup instead of at the first agent call.
func splitScopePatterns(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, strings.TrimSpace(p))
	}
	return out
}

func teamKeySource(cfg *config.Config) string {
	if strings.TrimSpace(cfg.TeamEncryptionKeyFile) != "" {
		return "file"
	}
	return "environment"
}

// startPProfServer starts the pprof debugging server on the declared
// loopback-only address, gated by the time-boxed access window (Spec 17
// R17-09). The handler is an explicit mux holding only the pprof endpoints
// rather than the process-wide DefaultServeMux, so the debug listener can
// never expose an unrelated handler registered elsewhere in the process.
func startPProfServer(cfg *config.Config) {
	addr := cfg.PProfBindAddr()
	if _, deadline := cfg.DebugEndpointActive(time.Now()); deadline.IsZero() {
		slog.Error("refusing to start the debug endpoint without a bounded access window (Spec 17 R17-09)", "addr", addr)
		return
	}
	slog.Info("Starting pprof server (loopback-only, time-boxed)", "addr", addr, "until", cfg.PProfUntil)

	// Dedicated mux: only the pprof handlers, never DefaultServeMux.
	// /debug/pprof/ - profile overview
	// /debug/pprof/profile - CPU profile
	// /debug/pprof/heap - heap profile
	// /debug/pprof/goroutine - goroutine profile
	dbgMux := http.NewServeMux()
	dbgMux.HandleFunc("/debug/pprof/", pprof.Index)
	dbgMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	dbgMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	dbgMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	dbgMux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	srv := &http.Server{Addr: addr, Handler: dbgMux}

	// Stop serving the moment the window closes instead of waiting for the
	// next request to notice it has expired.
	go func() {
		deadline, _, _ := cfg.PProfDeadline()
		timer := time.NewTimer(time.Until(deadline))
		defer timer.Stop()
		<-timer.C
		slog.Info("debug endpoint window closed; shutting it down (Spec 17 R17-09)", "addr", addr)
		_ = srv.Close()
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("pprof server error", "error", err)
	}
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
