package guardrails

import (
	"bytes"
	"fmt"
	"maps"
	"os"
	"sort"
	"sync"

	"github.com/dlclark/regexp2"
	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
)

// InjectionConfig holds the YAML-mirrored configuration for the injection guard.
type InjectionConfig struct {
	Enabled               bool              `yaml:"enabled"`
	L1Enabled             bool              `yaml:"l1_enabled"`
	L1Patterns            map[string]string `yaml:"l1_patterns"`
	L2Enabled             bool              `yaml:"l2_enabled"`
	PerplexityThreshold   float64           `yaml:"perplexity_threshold"`
	L3Enabled             bool              `yaml:"l3_enabled"`
	ClassifierBackend     string            `yaml:"classifier_backend"`
	ClassifierModelPath   string            `yaml:"classifier_model_path"`
	ClassifierThreshold   float64           `yaml:"classifier_threshold"`
	L4Enabled             bool              `yaml:"l4_enabled"`
	LLMSelfCheckModel     string            `yaml:"llm_self_check_model"`
	LLMSelfCheckThreshold float64           `yaml:"llm_self_check_threshold"`
	FailPolicy            string            `yaml:"fail_policy"`
	SourcePolicies        map[string]string `yaml:"source_policies"`
	BlocklistPaths        []string          `yaml:"blocklist_paths"`
	CustomPatterns        []string          `yaml:"custom_patterns"`
}

// DefaultInjectionConfig returns a config with spec-matching defaults.
func DefaultInjectionConfig() *InjectionConfig {
	return &InjectionConfig{
		Enabled:               true,
		L1Enabled:             true,
		L1Patterns:            DefaultPatterns(),
		L2Enabled:             true,
		PerplexityThreshold:   0.85,
		L3Enabled:             true,
		ClassifierBackend:     "llama-guard",
		ClassifierModelPath:   "",
		ClassifierThreshold:   0.7,
		L4Enabled:             false,
		LLMSelfCheckModel:     "",
		LLMSelfCheckThreshold: 0.5,
		FailPolicy:            "block",
		SourcePolicies: map[string]string{
			"user":         "warn",
			"tool_output":  "block",
			"file_content": "block",
			"api_response": "block",
		},
		BlocklistPaths: []string{},
		CustomPatterns: []string{},
	}
}

// DefaultPatterns returns the built-in injection pattern categories.
func DefaultPatterns() map[string]string {
	return map[string]string{
		"directive_override":   `(?i)(ignore\s+(all\s+)?previous\s+instructions|disregard\s+(your|all)\s+(instructions|rules)|forget\s+(your|all)\s+(instructions|rules))`,
		"role_play":            `(?i)(you\s+are\s+now\s+a|act\s+as\s+(if\s+)?you\s+(are|have)\s+no\s+rules|pretend\s+you\s+(are|have)\s+no)`,
		"encoding_bypass":      `(?i)(base64\s*decode|rot13|unicode\s+escape|\\u[0-9a-fA-F]{4})`,
		"context_manipulation": `(?i)(the\s+system\s+message\s+(actually\s+)?says|override\s+the\s+system\s+prompt|replace\s+your\s+instructions)`,
		"privilege_escalation": `(?i)(run\s+(this\s+)?as\s+root|execute\s+with\s+sudo|bypass\s+(the\s+)?permission|elevated\s+privileges)`,
		"data_exfiltration":    `(?i)(print\s+(your|the)\s+system\s+prompt|reveal\s+(your|the)\s+instructions|show\s+(me\s+)?(your|the)\s+system\s+(prompt|instructions)|output\s+your\s+system\s+prompt)`,
	}
}

// SourcePolicy returns the policy for a given source type.
func (c *InjectionConfig) SourcePolicy(source string) string {
	if p, ok := c.SourcePolicies[source]; ok {
		return p
	}
	return c.FailPolicy
}

// ToPipelineConfig projects this YAML/env-shaped configuration onto the
// nested PipelineConfig the pipeline actually reads.
//
// This conversion is the defect this method fixes: InjectionConfig existed with
// an l1_enabled/l2_enabled/l3_enabled surface, a classifier_backend, blocklist
// paths and custom patterns, and NOTHING converted it. The server built its
// pipeline from DefaultPipelineConfig() and never loaded an InjectionConfig at
// all, so an operator who set l3_enabled: false (or supplied extra patterns, or
// pointed at a blocklist) changed nothing — the settings were inert prose in a
// struct. A test now drives this conversion so the two shapes cannot drift
// again.
func (c *InjectionConfig) ToPipelineConfig() PipelineConfig {
	if c == nil {
		return DefaultPipelineConfig()
	}

	patterns := make([]string, 0, len(c.CustomPatterns)+len(c.L1Patterns))
	// Order the map-derived patterns deterministically so a config diff is
	// reproducible; append operator-supplied custom patterns after.
	categories := make([]string, 0, len(c.L1Patterns))
	for category := range c.L1Patterns {
		categories = append(categories, category)
	}
	sort.Strings(categories)
	for _, category := range categories {
		patterns = append(patterns, c.L1Patterns[category])
	}
	patterns = append(patterns, c.CustomPatterns...)

	// An enabled L1 with no patterns is a matcher that can never fire. Fall
	// back to the built-in set rather than shipping a vacuous layer.
	if c.L1Enabled && len(patterns) == 0 {
		patterns = DefaultPatternList()
	}

	sourcePolicies := make(map[Source]FailPolicy, len(c.SourcePolicies))
	for source, policy := range c.SourcePolicies {
		sourcePolicies[Source(source)] = FailPolicy(policy)
	}

	cfg := PipelineConfig{
		Enabled:    c.Enabled,
		FailPolicy: FailPolicy(c.FailPolicy),
		Layers: LayersConfig{
			PatternMatching: PatternMatchingConfig{
				Enabled:        c.L1Enabled,
				BlocklistPaths: c.BlocklistPaths,
				CustomPatterns: patterns,
			},
			Perplexity: PerplexityConfig{
				Enabled:   c.L2Enabled,
				Threshold: c.PerplexityThreshold,
			},
			Classifier: ClassifierConfig{
				Enabled:   c.L3Enabled,
				Backend:   c.ClassifierBackend,
				ModelPath: c.ClassifierModelPath,
				Threshold: c.ClassifierThreshold,
			},
			LLMSelfCheck: LLMSelfCheckConfig{
				Enabled:   c.L4Enabled,
				Model:     c.LLMSelfCheckModel,
				Threshold: c.LLMSelfCheckThreshold,
			},
		},
		SourcePolicies: sourcePolicies,
	}
	if cfg.FailPolicy == "" {
		cfg.FailPolicy = FailPolicyBlock
	}
	return cfg
}

// BlocklistManager handles loading and hot-reloading of blocklist files.
type BlocklistManager struct {
	mu       sync.RWMutex
	paths    []string
	patterns map[string]string // category -> regex
	watcher  *fsnotify.Watcher
	onReload func(map[string]string)
	stopCh   chan struct{}
}

// NewBlocklistManager creates a blocklist manager for the given file paths.
func NewBlocklistManager(paths []string, onReload func(map[string]string)) (*BlocklistManager, error) {
	bm := &BlocklistManager{
		paths:    paths,
		patterns: make(map[string]string),
		onReload: onReload,
		stopCh:   make(chan struct{}),
	}

	if err := bm.loadAll(); err != nil {
		return nil, fmt.Errorf("blocklist load: %w", err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("fsnotify watcher: %w", err)
	}
	bm.watcher = watcher

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			watcher.Add(p)
		}
	}

	go bm.watchLoop()
	return bm, nil
}

// Patterns returns the current set of blocklist patterns.
func (bm *BlocklistManager) Patterns() map[string]string {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	result := make(map[string]string, len(bm.patterns))
	maps.Copy(result, bm.patterns)
	return result
}

// Close stops the file watcher.
func (bm *BlocklistManager) Close() error {
	close(bm.stopCh)
	if bm.watcher != nil {
		return bm.watcher.Close()
	}
	return nil
}

func (bm *BlocklistManager) loadAll() error {
	merged := make(map[string]string)
	for _, path := range bm.paths {
		pats, err := loadBlocklistFile(path)
		if err != nil {
			return fmt.Errorf("load %s: %w", path, err)
		}
		maps.Copy(merged, pats)
	}
	bm.mu.Lock()
	bm.patterns = merged
	bm.mu.Unlock()
	return nil
}

func (bm *BlocklistManager) watchLoop() {
	for {
		select {
		case <-bm.stopCh:
			return
		case event, ok := <-bm.watcher.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				if err := bm.loadAll(); err == nil && bm.onReload != nil {
					bm.onReload(bm.Patterns())
				}
			}
		case <-bm.watcher.Errors:
			// watcher error; continue
		}
	}
}

// loadBlocklistFile reads a blocklist file: one regex per line, # comments.
func loadBlocklistFile(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	lines := splitLines(string(data))
	for i, line := range lines {
		line = trimComment(line)
		if line == "" {
			continue
		}
		// Validate the regex compiles
		if _, err := regexp2.Compile(line, 0); err != nil {
			return nil, fmt.Errorf("line %d: invalid regex: %w", i+1, err)
		}
		cat := fmt.Sprintf("blocklist_L%d", i+1)
		result[cat] = line
	}
	return result, nil
}

// injectionConfigFile is the accepted on-disk shape for one configuration
// file. It carries TWO forms because the tree documented one and implemented
// the other:
//
//   - the flat form the InjectionConfig struct tags declare
//     (`l1_enabled`, `blocklist_paths`, `custom_patterns`, …), and
//   - the nested `layers:` form Spec 01 Sec. 3.2 documents
//     (`layers.pattern_matching.enabled`, `layers.perplexity.threshold`, …).
//
// The loader used to unmarshal only the flat form. Every key of the
// documented YAML therefore landed on a struct with no matching field, YAML
// ignored it, and the resulting InjectionConfig had L1/L2/L3 all false — a
// configured, syntactically valid file that silently disabled the detector.
// Both forms are accepted here and an unknown key is an ERROR, so a shape
// mistake can never again read as "all layers off".
type injectionConfigFile struct {
	InjectionDefense *injectionDefenseYAML `yaml:"injection_defense"`
}

// injectionDefenseYAML is the `injection_defense:` section in either form.
// Pointers distinguish "absent" from "explicitly false": an omitted switch
// must inherit the shipped default, while an explicit false must win.
type injectionDefenseYAML struct {
	Enabled        *bool             `yaml:"enabled"`
	FailPolicy     string            `yaml:"fail_policy"`
	SourcePolicies map[string]string `yaml:"source_policies"`

	// Flat form.
	L1Enabled             *bool             `yaml:"l1_enabled"`
	L1Patterns            map[string]string `yaml:"l1_patterns"`
	L2Enabled             *bool             `yaml:"l2_enabled"`
	PerplexityThreshold   *float64          `yaml:"perplexity_threshold"`
	L3Enabled             *bool             `yaml:"l3_enabled"`
	ClassifierBackend     string            `yaml:"classifier_backend"`
	ClassifierModelPath   string            `yaml:"classifier_model_path"`
	ClassifierThreshold   *float64          `yaml:"classifier_threshold"`
	L4Enabled             *bool             `yaml:"l4_enabled"`
	LLMSelfCheckModel     string            `yaml:"llm_self_check_model"`
	LLMSelfCheckThreshold *float64          `yaml:"llm_self_check_threshold"`
	BlocklistPaths        []string          `yaml:"blocklist_paths"`
	CustomPatterns        []string          `yaml:"custom_patterns"`

	// Nested form (Spec 01 Sec. 3.2).
	Layers *injectionLayersYAML `yaml:"layers"`
}

type injectionLayersYAML struct {
	PatternMatching *patternMatchingYAML `yaml:"pattern_matching"`
	Perplexity      *perplexityYAML      `yaml:"perplexity"`
	Classifier      *classifierYAML      `yaml:"classifier"`
	LLMSelfCheck    *llmSelfCheckYAML    `yaml:"llm_self_check"`
}

// blocklistRef is the documented `- path: "..."` entry. A bare string is also
// accepted because the flat form used that shape.
type blocklistRef struct {
	Path string `yaml:"path"`
}

type blocklistPathsYAML []string

// UnmarshalYAML accepts each entry either as `- path: file` (documented) or
// as `- file` (flat form). A `path:` entry with no path is refused: an empty
// blocklist path would load no patterns while looking configured.
func (b *blocklistPathsYAML) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.SequenceNode {
		return fmt.Errorf("blocklists must be a list of paths")
	}
	out := make([]string, 0, len(node.Content))
	for i, item := range node.Content {
		switch item.Kind {
		case yaml.ScalarNode:
			var s string
			if err := item.Decode(&s); err != nil {
				return err
			}
			out = append(out, s)
		case yaml.MappingNode:
			var ref blocklistRef
			if err := item.Decode(&ref); err != nil {
				return fmt.Errorf("blocklists[%d]: %w", i, err)
			}
			if ref.Path == "" {
				return fmt.Errorf("blocklists[%d] declares no path", i)
			}
			out = append(out, ref.Path)
		default:
			return fmt.Errorf("blocklists[%d] must be a path string or a {path: ...} entry", i)
		}
	}
	*b = out
	return nil
}

type patternMatchingYAML struct {
	Enabled        *bool              `yaml:"enabled"`
	Blocklists     blocklistPathsYAML `yaml:"blocklists"`
	CustomPatterns []string           `yaml:"custom_patterns"`
}

type perplexityYAML struct {
	Enabled   *bool    `yaml:"enabled"`
	Threshold *float64 `yaml:"threshold"`
}

type classifierYAML struct {
	Enabled   *bool    `yaml:"enabled"`
	Backend   string   `yaml:"backend"`
	ModelPath string   `yaml:"model_path"`
	Threshold *float64 `yaml:"threshold"`
}

type llmSelfCheckYAML struct {
	Enabled   *bool    `yaml:"enabled"`
	Model     string   `yaml:"model"`
	Threshold *float64 `yaml:"threshold"`
}

// LoadInjectionConfigFromYAML reads the injection_defense section from YAML.
//
// It is deliberately strict. Three failure modes that used to be silent now
// stop startup instead:
//
//  1. A key the loader does not know is an ERROR (KnownFields), so a typo or
//     a shape change cannot leave a switch at its zero value.
//  2. A file with no `injection_defense` section is an ERROR: an empty file
//     is not a configuration, and treating it as one made every layer off.
//  3. Mixing the flat and nested forms is an ERROR. Two spellings of one
//     switch with different values has no correct answer, and picking one
//     would make the effective policy depend on parse order.
func LoadInjectionConfigFromYAML(path string) (*InjectionConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var raw injectionConfigFile
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&raw); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if raw.InjectionDefense == nil {
		return nil, fmt.Errorf("config %s declares no injection_defense section", path)
	}
	section := raw.InjectionDefense

	if section.Layers != nil && mixedInjectionForms(section) {
		return nil, fmt.Errorf("injection_defense mixes the flat and nested (`layers:`) forms; use one")
	}

	cfg := DefaultInjectionConfig()
	// Spec 01 Sec. 2.2: "L1-L2 run on every input", and Sec. 3.2 documents
	// `perplexity: enabled: true`. The loader previously forced L2 off so a
	// file could not silently enable a heuristic detector, but that made the
	// documented default unexpressible and left the tree with two defaults
	// that disagreed. The spec is the decision: L2 ships on, and a file may
	// still switch it off explicitly.

	if section.Enabled != nil {
		cfg.Enabled = *section.Enabled
	}
	if section.FailPolicy != "" {
		cfg.FailPolicy = section.FailPolicy
	}
	if section.SourcePolicies != nil {
		cfg.SourcePolicies = section.SourcePolicies
	}
	applyFlatInjectionFields(cfg, section)
	if section.Layers != nil {
		applyNestedInjectionFields(cfg, section.Layers)
	}

	if !cfg.Enabled {
		return cfg, nil
	}

	// An enabled L1 with no patterns is a matcher that can never fire; the
	// shipped default seeds the built-in set, and an explicit empty custom set
	// must not empty it.
	if cfg.L1Enabled && len(cfg.L1Patterns) == 0 && len(cfg.CustomPatterns) == 0 {
		cfg.L1Patterns = DefaultPatterns()
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid injection_defense config: %w", err)
	}
	return cfg, nil
}

// mixedInjectionForms reports whether the section sets a layer switch in both
// the flat and the nested form.
func mixedInjectionForms(s *injectionDefenseYAML) bool {
	l := s.Layers
	switch {
	case s.L1Enabled != nil && l.PatternMatching != nil && l.PatternMatching.Enabled != nil:
		return true
	case s.L2Enabled != nil && l.Perplexity != nil && l.Perplexity.Enabled != nil:
		return true
	case s.L3Enabled != nil && l.Classifier != nil && l.Classifier.Enabled != nil:
		return true
	case s.L4Enabled != nil && l.LLMSelfCheck != nil && l.LLMSelfCheck.Enabled != nil:
		return true
	case len(s.BlocklistPaths) > 0 && l.PatternMatching != nil && len(l.PatternMatching.Blocklists) > 0:
		return true
	case len(s.CustomPatterns) > 0 && l.PatternMatching != nil && len(l.PatternMatching.CustomPatterns) > 0:
		return true
	case s.PerplexityThreshold != nil && l.Perplexity != nil && l.Perplexity.Threshold != nil:
		return true
	case s.ClassifierThreshold != nil && l.Classifier != nil && l.Classifier.Threshold != nil:
		return true
	}
	return false
}

func applyFlatInjectionFields(cfg *InjectionConfig, s *injectionDefenseYAML) {
	if s.L1Enabled != nil {
		cfg.L1Enabled = *s.L1Enabled
	}
	if s.L1Patterns != nil {
		cfg.L1Patterns = s.L1Patterns
	}
	if s.L2Enabled != nil {
		cfg.L2Enabled = *s.L2Enabled
	}
	if s.PerplexityThreshold != nil {
		cfg.PerplexityThreshold = *s.PerplexityThreshold
	}
	if s.L3Enabled != nil {
		cfg.L3Enabled = *s.L3Enabled
	}
	if s.ClassifierBackend != "" {
		cfg.ClassifierBackend = s.ClassifierBackend
	}
	if s.ClassifierModelPath != "" {
		cfg.ClassifierModelPath = s.ClassifierModelPath
	}
	if s.ClassifierThreshold != nil {
		cfg.ClassifierThreshold = *s.ClassifierThreshold
	}
	if s.L4Enabled != nil {
		cfg.L4Enabled = *s.L4Enabled
	}
	if s.LLMSelfCheckModel != "" {
		cfg.LLMSelfCheckModel = s.LLMSelfCheckModel
	}
	if s.LLMSelfCheckThreshold != nil {
		cfg.LLMSelfCheckThreshold = *s.LLMSelfCheckThreshold
	}
	if s.BlocklistPaths != nil {
		cfg.BlocklistPaths = s.BlocklistPaths
	}
	if s.CustomPatterns != nil {
		cfg.CustomPatterns = s.CustomPatterns
	}
}

func applyNestedInjectionFields(cfg *InjectionConfig, l *injectionLayersYAML) {
	if pm := l.PatternMatching; pm != nil {
		if pm.Enabled != nil {
			cfg.L1Enabled = *pm.Enabled
		}
		if pm.Blocklists != nil {
			cfg.BlocklistPaths = pm.Blocklists
		}
		if pm.CustomPatterns != nil {
			cfg.CustomPatterns = pm.CustomPatterns
		}
	}
	if p := l.Perplexity; p != nil {
		if p.Enabled != nil {
			cfg.L2Enabled = *p.Enabled
		}
		if p.Threshold != nil {
			cfg.PerplexityThreshold = *p.Threshold
		}
	}
	if c := l.Classifier; c != nil {
		if c.Enabled != nil {
			cfg.L3Enabled = *c.Enabled
		}
		if c.Backend != "" {
			cfg.ClassifierBackend = c.Backend
		}
		if c.ModelPath != "" {
			cfg.ClassifierModelPath = c.ModelPath
		}
		if c.Threshold != nil {
			cfg.ClassifierThreshold = *c.Threshold
		}
	}
	if s := l.LLMSelfCheck; s != nil {
		if s.Enabled != nil {
			cfg.L4Enabled = *s.Enabled
		}
		if s.Model != "" {
			cfg.LLMSelfCheckModel = s.Model
		}
		if s.Threshold != nil {
			cfg.LLMSelfCheckThreshold = *s.Threshold
		}
	}
}

// Validate checks the config for errors.
func (c *InjectionConfig) Validate() error {
	validPolicies := map[string]bool{"block": true, "warn": true, "log_only": true}
	if !validPolicies[c.FailPolicy] {
		return fmt.Errorf("invalid fail_policy %q: must be block, warn, or log_only", c.FailPolicy)
	}
	for src, pol := range c.SourcePolicies {
		if !validPolicies[pol] {
			return fmt.Errorf("invalid source_policy for %s: %q", src, pol)
		}
	}
	if c.L2Enabled && (c.PerplexityThreshold <= 0 || c.PerplexityThreshold > 1) {
		return fmt.Errorf("perplexity_threshold must be in (0,1], got %.2f", c.PerplexityThreshold)
	}
	if c.L3Enabled && (c.ClassifierThreshold <= 0 || c.ClassifierThreshold > 1) {
		return fmt.Errorf("classifier_threshold must be in (0,1], got %.2f", c.ClassifierThreshold)
	}
	// Validate all L1 patterns compile
	for cat, pat := range c.L1Patterns {
		if _, err := regexp2.Compile(pat, 0); err != nil {
			return fmt.Errorf("invalid L1 pattern for %s: %w", cat, err)
		}
	}
	return nil
}

// Helper functions for parsing

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func trimComment(line string) string {
	for i := 0; i < len(line); i++ {
		if line[i] == '#' {
			return line[:i]
		}
	}
	return line
}
