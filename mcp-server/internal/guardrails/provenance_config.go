package guardrails

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// ProvenanceConfig holds the configuration for indirect prompt injection protection.
type ProvenanceConfig struct {
	Enabled             bool                   `yaml:"enabled"`
	SourceTrustPolicies []TrustPolicy          `yaml:"source_trust_policies"`
	UntrustedOverrides  UntrustedContentConfig `yaml:"untrusted_overrides"`
	CacheTTL            int                    `yaml:"cache_ttl_hours"`
}

// UntrustedContentConfig defines overrides for untrusted content handling.
type UntrustedContentConfig struct {
	InjectionThreshold           float64 `yaml:"injection_threshold"`
	ContentFilterThreshold       float64 `yaml:"content_filter_threshold"`
	DirectiveOverrideDetection   bool    `yaml:"directive_override_detection"`
	RolePlayDetection            bool    `yaml:"role_play_detection"`
}

// DefaultProvenanceConfig returns a config with spec-matching defaults.
func DefaultProvenanceConfig() *ProvenanceConfig {
	return &ProvenanceConfig{
		Enabled: true,
		CacheTTL: 1, // 1 hour as per spec 4.1
		SourceTrustPolicies: []TrustPolicy{
			// Trusted sources first (specific patterns must precede wildcards)
			{SourcePattern: "CLAUDE.md", TrustLevel: TrustLevelTrusted, Action: ActionAllow},
			{SourcePattern: "AGENTS.md", TrustLevel: TrustLevelTrusted, Action: ActionAllow},
			{SourcePattern: "docs/**/*.md", TrustLevel: TrustLevelTrusted, Action: ActionAllow},
			{SourcePattern: "config/guardrails.yaml", TrustLevel: TrustLevelTrusted, Action: ActionAllow},
			{SourcePattern: "config/guardrails.yml", TrustLevel: TrustLevelTrusted, Action: ActionAllow},

			// API-based sources
			{SourcePattern: "github.com", TrustLevel: TrustLevelUntrusted, Action: ActionScanAndBlock},
			{SourcePattern: "api.internal.*", TrustLevel: TrustLevelTrusted, Action: ActionAllow},
			{SourcePattern: "localhost", TrustLevel: TrustLevelTrusted, Action: ActionAllow},

			// File-based sources are untrusted by default (wildcards after specific patterns)
			{SourcePattern: "*.json", TrustLevel: TrustLevelUntrusted, Action: ActionScanAndWarn},
			{SourcePattern: "*.yaml", TrustLevel: TrustLevelUntrusted, Action: ActionScanAndWarn},
			{SourcePattern: "*.yml", TrustLevel: TrustLevelUntrusted, Action: ActionScanAndWarn},
			{SourcePattern: "*.md", TrustLevel: TrustLevelUntrusted, Action: ActionScanAndWarn},
			{SourcePattern: "*.txt", TrustLevel: TrustLevelUntrusted, Action: ActionScanAndWarn},
			{SourcePattern: "*.go", TrustLevel: TrustLevelUntrusted, Action: ActionScanAndWarn},
			{SourcePattern: "*.py", TrustLevel: TrustLevelUntrusted, Action: ActionScanAndWarn},
			{SourcePattern: "*.js", TrustLevel: TrustLevelUntrusted, Action: ActionScanAndWarn},
			{SourcePattern: "*.ts", TrustLevel: TrustLevelUntrusted, Action: ActionScanAndWarn},
			{SourcePattern: "*.html", TrustLevel: TrustLevelUntrusted, Action: ActionScanAndWarn},
			{SourcePattern: "*.css", TrustLevel: TrustLevelUntrusted, Action: ActionScanAndWarn},
		},
		UntrustedOverrides: UntrustedContentConfig{
			InjectionThreshold:           0.5,
			ContentFilterThreshold:       0.5,
			DirectiveOverrideDetection:   true,
			RolePlayDetection:            true,
		},
	}
}

// LoadProvenanceConfigFromYAML reads the indirect_injection section from YAML.
func LoadProvenanceConfigFromYAML(path string) (*ProvenanceConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var raw struct {
		IndirectInjection ProvenanceConfig `yaml:"indirect_injection"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	cfg := &raw.IndirectInjection
	if !cfg.Enabled {
		return cfg, nil
	}

	// Merge defaults for zero values
	if len(cfg.SourceTrustPolicies) == 0 {
		cfg.SourceTrustPolicies = DefaultProvenanceConfig().SourceTrustPolicies
	}
	if cfg.CacheTTL == 0 {
		cfg.CacheTTL = 1
	}
	if cfg.UntrustedOverrides.InjectionThreshold == 0 {
		cfg.UntrustedOverrides.InjectionThreshold = 0.5
	}
	if cfg.UntrustedOverrides.ContentFilterThreshold == 0 {
		cfg.UntrustedOverrides.ContentFilterThreshold = 0.5
	}

	return cfg, nil
}

// Validate checks the config for errors.
func (c *ProvenanceConfig) Validate() error {
	if c.CacheTTL < 0 {
		return errInvalidCacheTTL
	}
	for _, p := range c.SourceTrustPolicies {
		if p.SourcePattern == "" {
			return errInvalidPattern
		}
		if p.TrustLevel != TrustLevelTrusted &&
		   p.TrustLevel != TrustLevelUntrusted &&
		   p.TrustLevel != TrustLevelUnknown {
			return errInvalidTrustLevel
		}
		if p.Action != ActionAllow &&
		   p.Action != ActionScanAndWarn &&
		   p.Action != ActionScanAndBlock {
			return errInvalidAction
		}
	}
	return nil
}

// ResolveTrustForPath returns the trust level and action for a given source path.
func (c *ProvenanceConfig) ResolveTrustForPath(path string) (SourceTrustLevel, Action) {
	for _, p := range c.SourceTrustPolicies {
		if matchPattern(p.SourcePattern, path) {
			return p.TrustLevel, p.Action
		}
	}
	return TrustLevelUnknown, ActionScanAndWarn
}

// CacheDuration returns the cache TTL duration in hours.
func (c *ProvenanceConfig) CacheDuration() time.Duration {
	if c.CacheTTL <= 0 {
		return time.Hour
	}
	return time.Duration(c.CacheTTL) * time.Hour
}

// IsEnabled checks if provenance tracking is enabled.
func (c *ProvenanceConfig) IsEnabled() bool {
	return c.Enabled
}

// GetInjectionThreshold returns the threshold for injection detection.
func (c *ProvenanceConfig) GetInjectionThreshold() float64 {
	return c.UntrustedOverrides.InjectionThreshold
}

var (
	errInvalidCacheTTL   = fmt.Errorf("cache_ttl_hours must be >= 0")
	errInvalidPattern    = fmt.Errorf("source_pattern cannot be empty")
	errInvalidTrustLevel = fmt.Errorf("trust_level must be trusted, untrusted, or unknown")
	errInvalidAction     = fmt.Errorf("action must be allow, scan_and_warn, or scan_and_block")
)

// getEnvOrDefault returns the environment variable value or a default.
func getEnvOrDefault(key, defaultVal string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return defaultVal
}

// ConfigFromEnv creates a provenance config with environment overrides.
func ConfigFromEnv() *ProvenanceConfig {
	cfg := DefaultProvenanceConfig()

	if v := getEnvOrDefault("GUARDRAILS_PROVENANCE_ENABLED", ""); v != "" {
		cfg.Enabled = v == "true"
	}
	if v := getEnvOrDefault("GUARDRAILS_PROVENANCE_CACHE_TTL", ""); v != "" {
		if n := parseInt(v); n >= 0 {
			cfg.CacheTTL = n
		}
	}

	return cfg
}

// parseInt is a simple helper to parse an integer from string.
func parseInt(s string) int {
	var n int
	_, _ = fmt.Sscanf(s, "%d", &n)
	return n
}

// Ensure ConfigFromEnv is used (avoids unused import warnings)
var _ = filepath.Join
