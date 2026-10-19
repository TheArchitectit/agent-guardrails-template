package guardrails

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Re-exports for content_filter.go compatibility
type Action string

const (
	ActionBlock        Action = "block"
	ActionWarn         Action = "warn"
	ActionAllow        Action = "allow"
	ActionScanAndWarn  Action = "scan_and_warn"
	ActionScanAndBlock Action = "scan_and_block"
)

// ContentFailPolicy defines the behavior when all classification backends fail.
type ContentFailPolicy string

const (
	ContentFailPolicyBlock ContentFailPolicy = "block" // Block content if classifier is unavailable
	ContentFailPolicyAllow ContentFailPolicy = "allow" // Allow content if classifier is unavailable
)

// ContentFilterConfig holds the configuration for semantic content filtering.
type ContentFilterConfig struct {
	Enabled       bool              `yaml:"enabled"`
	Backend       string            `yaml:"backend"`
	BackendConfig BackendConfig     `yaml:"backend_config"`
	Taxonomy      TaxonomyConfig    `yaml:"taxonomy"`
	Thresholds    ThresholdConfig   `yaml:"thresholds"`
	Policies      []PolicyRule      `yaml:"policies"`
	FailPolicy    ContentFailPolicy `yaml:"fail_policy"`
}

// BackendConfig holds configuration for individual classifier backends.
type BackendConfig struct {
	LlamaGuard       LlamaGuardConfig `yaml:"llama_guard"`
	OpenAIModeration OpenAIConfig     `yaml:"openai_moderation"`
	// Nemo is parsed because Sec. 3.2 documents it in the same block as the
	// implemented backends. It is NOT implemented: naming it as the selected
	// backend is refused at load rather than silently ignored, which is what
	// happened before — `backend: "nemo"` left the deployment on llama-guard
	// with nothing saying so.
	Nemo NemoConfig `yaml:"nemo"`
}

// NemoConfig configures the (unimplemented) NeMo backend.
type NemoConfig struct {
	ConfigPath string `yaml:"config_path"`
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
	for cat, th := range c.Thresholds.PerCategory {
		if th < 0 || th > 1 {
			return fmt.Errorf("content filter threshold for category %s must be 0-1", cat)
		}
	}
	for i, p := range c.Policies {
		if strings.TrimSpace(p.ID) == "" {
			return fmt.Errorf("content filter policy %d declares no id", i)
		}
		for _, d := range p.Rules {
			if !validContentAction(d.Action) {
				return fmt.Errorf("content filter policy %q rule for %s has invalid action %q", p.ID, d.Category, d.Action)
			}
		}
		for _, o := range p.Overrides {
			if !validContentAction(o.Action) {
				return fmt.Errorf("content filter policy %q override for %s has invalid action %q", p.ID, o.Category, o.Action)
			}
		}
	}
	return nil
}

// validContentAction reports whether an action is one the policy engine can
// apply. An unrecognised action previously fell through Evaluate's switch and
// left the category at its default action, so a typo silently did nothing.
func validContentAction(a Action) bool {
	switch a {
	case ActionBlock, ActionWarn, ActionAllow:
		return true
	default:
		return false
	}
}

// implementedBackends names the classifier backends this tree can actually
// construct. Sec. 3.2 also documents "nemo" and "custom"; neither exists, so
// selecting one is refused at load rather than accepted and ignored.
var implementedBackends = map[string]bool{
	"llama-guard":       true,
	"llama_guard":       true,
	"openai-moderation": true,
	"openai_moderation": true,
}

// contentFilterSectionYAML mirrors the surface CONF-02 may set. Every switch
// is a pointer so an omitted key inherits the shipped default while an explicit
// value wins — decoding straight into the struct collapses "absent" and "set to
// the zero value", which is how a file naming only `max_mb` would silently turn
// a security switch off elsewhere in this tree.
type contentFilterSectionYAML struct {
	Enabled       *bool              `yaml:"enabled"`
	Backend       string             `yaml:"backend"`
	BackendConfig *backendConfigYAML `yaml:"backend_config"`
	Taxonomy      *taxonomyYAML      `yaml:"taxonomy"`
	Thresholds    *thresholdsYAML    `yaml:"thresholds"`
	Policies      []PolicyRule       `yaml:"policies"`
	FailPolicy    string             `yaml:"fail_policy"`
}

type backendConfigYAML struct {
	LlamaGuard       *llamaGuardYAML `yaml:"llama_guard"`
	OpenAIModeration *openAIYAML     `yaml:"openai_moderation"`
	Nemo             *nemoYAML       `yaml:"nemo"`
}

type llamaGuardYAML struct {
	Model     string `yaml:"model"`
	OllamaURL string `yaml:"ollama_url"`
}

type openAIYAML struct {
	APIKeyEnv string `yaml:"api_key_env"`
}

type nemoYAML struct {
	ConfigPath string `yaml:"config_path"`
}

type taxonomyYAML struct {
	EnabledCategories []string `yaml:"enabled_categories"`
	CustomCategories  []string `yaml:"custom_categories"`
}

type thresholdsYAML struct {
	Default     *float64           `yaml:"default"`
	PerCategory map[string]float64 `yaml:"per_category"`
}

// contentFilterFile is the wrapper shape: Sec. 3.2 documents the block under a
// top-level `content_filter:` key. Before this the loader unmarshalled the file
// directly into the flat struct, so every documented key landed on a struct
// with no matching field, YAML discarded it without complaint, and a verbatim
// copy of the spec's own example loaded with ZERO policies and the full
// S1-S15 taxonomy enabled — the same fail-open class as the Spec 01 and Spec 03
// config seams, one spec over.
type contentFilterFile struct {
	ContentFilter *contentFilterSectionYAML `yaml:"content_filter"`
}

// LoadContentConfig loads the content filter configuration from a YAML file.
//
// It is deliberately strict. Failure modes that used to be silent now stop
// startup instead:
//
//  1. An unknown key is an ERROR (KnownFields), so a typo cannot leave a
//     switch at its zero value and load clean.
//  2. Both the documented `content_filter:` wrapper and the flat struct-tag
//     form load, but not at once: two spellings of one config has no correct
//     merge order.
//  3. A file that declares no content_filter section is an ERROR: an empty
//     file is not a configuration.
//  4. `fail_policy: allow` is refused. Failing open on an unavailable
//     classifier is a security semantic, and P5/D-18 makes those
//     non-configurable.
//  5. A backend the tree cannot construct (nemo, custom, unknown) is refused
//     rather than silently ignored.
func LoadContentConfig(path string) (*ContentFilterConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read content config: %w", err)
	}

	// Detect which shape the file uses before strict decoding, so KnownFields
	// can be applied to the right struct without rejecting the other form's
	// keys as unknown.
	var probe map[string]any
	if err := yaml.Unmarshal(data, &probe); err != nil {
		return nil, fmt.Errorf("parse content config: %w", err)
	}
	_, wrapped := probe["content_filter"]
	flatKeys := false
	for _, k := range []string{"enabled", "backend", "backend_config", "taxonomy", "thresholds", "policies", "fail_policy"} {
		if _, ok := probe[k]; ok {
			flatKeys = true
			break
		}
	}
	if wrapped && flatKeys {
		return nil, fmt.Errorf("config %s declares both the `content_filter:` wrapper and top-level content-filter keys; use one", path)
	}
	if !wrapped && !flatKeys {
		return nil, fmt.Errorf("config %s declares no content_filter section", path)
	}

	var section *contentFilterSectionYAML
	if wrapped {
		var root contentFilterFile
		dec := yaml.NewDecoder(bytes.NewReader(data))
		dec.KnownFields(true)
		if err := dec.Decode(&root); err != nil {
			return nil, fmt.Errorf("parse content config: %w", err)
		}
		section = root.ContentFilter
	} else {
		var s contentFilterSectionYAML
		dec := yaml.NewDecoder(bytes.NewReader(data))
		dec.KnownFields(true)
		if err := dec.Decode(&s); err != nil {
			return nil, fmt.Errorf("parse content config: %w", err)
		}
		section = &s
	}

	cfg := DefaultFilterConfig()
	if section.Enabled != nil {
		cfg.Enabled = *section.Enabled
	}
	if section.Backend != "" {
		if !implementedBackends[section.Backend] {
			return nil, fmt.Errorf("content filter backend %q is not implemented by this tree (Sec. 3.2 documents llama-guard and openai-moderation)", section.Backend)
		}
		cfg.Backend = section.Backend
	}
	if section.FailPolicy != "" {
		fp := ContentFailPolicy(section.FailPolicy)
		if fp != ContentFailPolicyBlock && fp != ContentFailPolicyAllow {
			return nil, fmt.Errorf("content filter fail_policy %q must be 'block' or 'allow'", section.FailPolicy)
		}
		if fp == ContentFailPolicyAllow {
			return nil, fmt.Errorf("content filter fail_policy 'allow' is refused: failing open on an unavailable classifier is a fail-closed security semantic (P5/D-18)")
		}
		cfg.FailPolicy = fp
	}
	if section.Thresholds != nil {
		if section.Thresholds.Default != nil {
			cfg.Thresholds.Default = *section.Thresholds.Default
		}
		if section.Thresholds.PerCategory != nil {
			cfg.Thresholds.PerCategory = section.Thresholds.PerCategory
		}
	}
	if section.Taxonomy != nil {
		// A declared taxonomy is a declaration, not a merge: the spec's own
		// example enables a subset, so an empty list means "evaluate nothing"
		// rather than "fall back to all fifteen".
		cfg.Taxonomy.EnabledCategories = section.Taxonomy.EnabledCategories
		cfg.Taxonomy.CustomCategories = section.Taxonomy.CustomCategories
	}
	if section.Policies != nil {
		cfg.Policies = section.Policies
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid content config: %w", err)
	}
	return cfg, nil
}
