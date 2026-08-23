package guardrails

import (
	"fmt"
	"maps"
	"os"
	"sync"

	"github.com/dlclark/regexp2"
	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
)

// InjectionConfig holds the YAML-mirrored configuration for the injection guard.
type InjectionConfig struct {
	Enabled             bool                       `yaml:"enabled"`
	L1Enabled           bool                       `yaml:"l1_enabled"`
	L1Patterns          map[string]string          `yaml:"l1_patterns"`
	L2Enabled           bool                       `yaml:"l2_enabled"`
	PerplexityThreshold float64                    `yaml:"perplexity_threshold"`
	L3Enabled           bool                       `yaml:"l3_enabled"`
	ClassifierBackend   string                     `yaml:"classifier_backend"`
	ClassifierModelPath string                     `yaml:"classifier_model_path"`
	ClassifierThreshold float64                    `yaml:"classifier_threshold"`
	L4Enabled           bool                       `yaml:"l4_enabled"`
	LLMSelfCheckModel   string                     `yaml:"llm_self_check_model"`
	LLMSelfCheckThreshold float64                  `yaml:"llm_self_check_threshold"`
	FailPolicy          string                     `yaml:"fail_policy"`
	SourcePolicies      map[string]string          `yaml:"source_policies"`
	BlocklistPaths      []string                   `yaml:"blocklist_paths"`
	CustomPatterns      []string                   `yaml:"custom_patterns"`
}

// DefaultInjectionConfig returns a config with spec-matching defaults.
func DefaultInjectionConfig() *InjectionConfig {
	return &InjectionConfig{
		Enabled:             true,
		L1Enabled:           true,
		L1Patterns:          DefaultPatterns(),
		L2Enabled:           true,
		PerplexityThreshold: 0.85,
		L3Enabled:           true,
		ClassifierBackend:   "llama-guard",
		ClassifierModelPath: "",
		ClassifierThreshold: 0.7,
		L4Enabled:           false,
		LLMSelfCheckModel:   "",
		LLMSelfCheckThreshold: 0.5,
		FailPolicy:          "block",
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

// LoadInjectionConfigFromYAML reads the injection_defense section from YAML.
func LoadInjectionConfigFromYAML(path string) (*InjectionConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var raw struct {
		InjectionDefense InjectionConfig `yaml:"injection_defense"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	cfg := &raw.InjectionDefense
	if !cfg.Enabled {
		return cfg, nil
	}

	// Merge defaults for zero values
	if cfg.L1Patterns == nil {
		cfg.L1Patterns = DefaultPatterns()
	}
	if cfg.PerplexityThreshold == 0 {
		cfg.PerplexityThreshold = 0.85
	}
	if cfg.ClassifierThreshold == 0 {
		cfg.ClassifierThreshold = 0.7
	}
	if cfg.FailPolicy == "" {
		cfg.FailPolicy = "block"
	}
	if cfg.SourcePolicies == nil {
		cfg.SourcePolicies = map[string]string{
			"user":         "warn",
			"tool_output":  "block",
			"file_content": "block",
			"api_response": "block",
		}
	}

	return cfg, nil
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
