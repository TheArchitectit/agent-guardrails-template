package guardrails

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
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
	InjectionThreshold         float64 `yaml:"injection_threshold"`
	ContentFilterThreshold     float64 `yaml:"content_filter_threshold"`
	DirectiveOverrideDetection bool    `yaml:"directive_override_detection"`
	RolePlayDetection          bool    `yaml:"role_play_detection"`
}

// DefaultProvenanceConfig returns a config with spec-matching defaults.
func DefaultProvenanceConfig() *ProvenanceConfig {
	return &ProvenanceConfig{
		Enabled:  true,
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
			InjectionThreshold:         0.5,
			ContentFilterThreshold:     0.5,
			DirectiveOverrideDetection: true,
			RolePlayDetection:          true,
		},
	}
}

// provenanceSectionYAML mirrors the surface CONF-05 may set. Pointers make
// absence distinguishable from an explicit false.
type provenanceSectionYAML struct {
	Enabled             *bool                   `yaml:"enabled"`
	SourceTrustPolicies []TrustPolicy           `yaml:"source_trust_policies"`
	UntrustedOverrides  *untrustedOverridesYAML `yaml:"untrusted_overrides"`
	CacheTTL            *int                    `yaml:"cache_ttl_hours"`
}

type untrustedOverridesYAML struct {
	InjectionThreshold         *float64 `yaml:"injection_threshold"`
	ContentFilterThreshold     *float64 `yaml:"content_filter_threshold"`
	DirectiveOverrideDetection *bool    `yaml:"directive_override_detection"`
	RolePlayDetection          *bool    `yaml:"role_play_detection"`
}

// provenanceFile is the wrapper: Sec. 2.3 documents the block under a top-level
// `indirect_injection:` key.
type provenanceFile struct {
	IndirectInjection *provenanceSectionYAML `yaml:"indirect_injection"`
}

// LoadProvenanceConfigFromYAML reads the indirect_injection section from YAML.
//
// It is deliberately strict, for the same reasons as the Spec 01/02/03 config
// loaders: an unknown key is an ERROR (KnownFields) so a typo cannot leave a
// switch at its zero value; a file with no `indirect_injection` section is an
// ERROR; and a config that fails Validate() stops startup. Before this it
// unmarshalled with no strictness at all, so an unknown key, a bad trust level
// and an empty file all loaded clean, and the resulting invalid policy set was
// only visible if a caller remembered to call Validate().
func LoadProvenanceConfigFromYAML(path string) (*ProvenanceConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var root provenanceFile
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&root); err != nil {
		// An empty file is not a configuration. Reporting the decoder's bare
		// "EOF" would send an operator looking for a YAML syntax error in a
		// file that is simply empty, so name the actual problem.
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("config %s is empty; it declares no indirect_injection section", path)
		}
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if root.IndirectInjection == nil {
		return nil, fmt.Errorf("config %s declares no indirect_injection section", path)
	}
	section := root.IndirectInjection

	cfg := DefaultProvenanceConfig()
	if section.Enabled != nil {
		cfg.Enabled = *section.Enabled
	}
	if section.SourceTrustPolicies != nil {
		// A declared policy set REPLACES the shipped table. Merging would keep
		// running a policy the operator removed, which is the opposite of what
		// deleting it says.
		cfg.SourceTrustPolicies = section.SourceTrustPolicies
	}
	if section.CacheTTL != nil {
		cfg.CacheTTL = *section.CacheTTL
	}
	if o := section.UntrustedOverrides; o != nil {
		if o.InjectionThreshold != nil {
			cfg.UntrustedOverrides.InjectionThreshold = *o.InjectionThreshold
		}
		if o.ContentFilterThreshold != nil {
			cfg.UntrustedOverrides.ContentFilterThreshold = *o.ContentFilterThreshold
		}
		if o.DirectiveOverrideDetection != nil {
			cfg.UntrustedOverrides.DirectiveOverrideDetection = *o.DirectiveOverrideDetection
		}
		if o.RolePlayDetection != nil {
			cfg.UntrustedOverrides.RolePlayDetection = *o.RolePlayDetection
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid provenance config: %w", err)
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
