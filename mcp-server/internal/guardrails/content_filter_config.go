package guardrails

import (
	"fmt"
	"log/slog"
	"os"

	"gopkg.in/yaml.v3"
)

// Re-exports for content_filter.go compatibility
type Action string

const (
	ActionBlock         Action = "block"
	ActionWarn          Action = "warn"
	ActionAllow         Action = "allow"
	ActionScanAndWarn   Action = "scan_and_warn"
	ActionScanAndBlock  Action = "scan_and_block"
)

// ContentFailPolicy defines the behavior when all classification backends fail.
type ContentFailPolicy string

const (
	ContentFailPolicyBlock ContentFailPolicy = "block" // Block content if classifier is unavailable
	ContentFailPolicyAllow ContentFailPolicy = "allow" // Allow content if classifier is unavailable
)

// ContentFilterConfig holds the configuration for semantic content filtering.
type ContentFilterConfig struct {
	Enabled       bool                 `yaml:"enabled"`
	Backend       string               `yaml:"backend"`
	BackendConfig BackendConfig        `yaml:"backend_config"`
	Taxonomy      TaxonomyConfig       `yaml:"taxonomy"`
	Thresholds    ThresholdConfig      `yaml:"thresholds"`
	Policies      []PolicyRule         `yaml:"policies"`
	FailPolicy    ContentFailPolicy    `yaml:"fail_policy"`
}

// BackendConfig holds configuration for individual classifier backends.
type BackendConfig struct {
	LlamaGuard       LlamaGuardConfig `yaml:"llama_guard"`
	OpenAIModeration OpenAIConfig     `yaml:"openai_moderation"`
}

// LlamaGuardConfig configures the Llama Guard backend.
type LlamaGuardConfig struct {
	Model     string `yaml:"model"`
	OllamaURL string `yaml:"ollama_url"`
}

// OpenAIConfig configures the OpenAI Moderation backend.
type OpenAIConfig struct {
	APIKeyEnv string `yaml:"api_key_env"`
}

// TaxonomyConfig controls which categories are evaluated.
type TaxonomyConfig struct {
	EnabledCategories []string `yaml:"enabled_categories"`
	CustomCategories  []string `yaml:"custom_categories"`
}

// ThresholdConfig defines score thresholds for classification.
type ThresholdConfig struct {
	Default     float64            `yaml:"default"`
	PerCategory map[string]float64 `yaml:"per_category"`
}

// PolicyRule defines a rule mapping a category to an action.
type PolicyRule struct {
	ID          string           `yaml:"id"`
	Description string           `yaml:"description"`
	Rules       []PolicyDetail   `yaml:"rules"`
	Overrides   []PolicyOverride `yaml:"overrides"`
}

// PolicyDetail maps a single category to an action with an optional threshold.
type PolicyDetail struct {
	Category    string  `yaml:"category"`
	Action      Action  `yaml:"action"`
	Threshold   float64 `yaml:"threshold"`
	Description string  `yaml:"description"`
}

// PolicyOverride provides context-specific overrides for a policy.
type PolicyOverride struct {
	Category       string `yaml:"category"`
	Action         Action `yaml:"action"`
	Description    string `yaml:"description"`
	ContextPattern string `yaml:"context_pattern"`
}

// DefaultFilterConfig returns a sane set of defaults for content filtering.
func DefaultFilterConfig() *ContentFilterConfig {
	return &ContentFilterConfig{
		Enabled:    true,
		Backend:    "llama-guard",
		FailPolicy: ContentFailPolicyBlock,
		Thresholds: ThresholdConfig{
			Default:     0.7,
			PerCategory: make(map[string]float64),
		},
		Taxonomy: TaxonomyConfig{
			EnabledCategories: AllCategoryIDs(),
		},
	}
}

// AllCategoryIDs returns all S1-S15 category IDs.
func AllCategoryIDs() []string {
	return []string{
		"S1", "S2", "S3", "S4", "S5", "S6", "S7", "S8",
		"S9", "S10", "S11", "S12", "S13", "S14", "S15",
	}
}

// Validate checks the config for errors.
func (c *ContentFilterConfig) Validate() error {
	if c.Thresholds.Default < 0 || c.Thresholds.Default > 1 {
		return fmt.Errorf("content filter default threshold must be 0-1")
	}
	if c.FailPolicy != ContentFailPolicyBlock && c.FailPolicy != ContentFailPolicyAllow {
		return fmt.Errorf("content filter fail_policy must be 'block' or 'allow'")
	}
	return nil
}

// LoadContentConfig loads the content filter configuration from a YAML file.
func LoadContentConfig(path string) (*ContentFilterConfig, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		slog.Info("Content config file not found, using defaults", "path", path)
		return DefaultFilterConfig(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read content config: %w", err)
	}

	cfg := DefaultFilterConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse content config: %w", err)
	}

	// Merge: set zero-value thresholds from defaults
	if cfg.Thresholds.Default == 0 {
		cfg.Thresholds.Default = 0.7
	}
	if cfg.FailPolicy == "" {
		cfg.FailPolicy = ContentFailPolicyBlock
	}

	return cfg, nil
}
