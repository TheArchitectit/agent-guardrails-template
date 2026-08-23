package guardrails

import (
	"context"
	"log/slog"
	"math"
	"sync"

	"github.com/dlclark/regexp2"
)

// PatternMatcher implements L1 pattern matching detection.
type PatternMatcher struct {
	config   PatternMatchingConfig
	patterns []*regexp2.Regexp
	mu       sync.RWMutex
}

// NewPatternMatcher creates a new pattern matcher with compiled regexes.
func NewPatternMatcher(config PatternMatchingConfig) (*PatternMatcher, error) {
	pm := &PatternMatcher{config: config}
	if err := pm.loadPatterns(); err != nil {
		return nil, err
	}
	return pm, nil
}

func (pm *PatternMatcher) loadPatterns() error {
	var patterns []*regexp2.Regexp

	for _, path := range pm.config.BlocklistPaths {
		lines, err := ReadBlocklistFile(path)
		if err != nil {
			slog.Warn("Failed to read blocklist", "path", path, "error", err)
			continue
		}
		for _, line := range lines {
			re, err := regexp2.Compile(line, regexp2.None)
			if err != nil {
				slog.Warn("Failed to compile pattern", "pattern", line, "error", err)
				continue
			}
			patterns = append(patterns, re)
		}
	}

	for _, pattern := range pm.config.CustomPatterns {
		re, err := regexp2.Compile(pattern, regexp2.None)
		if err != nil {
			slog.Warn("Failed to compile custom pattern", "pattern", pattern, "error", err)
			continue
		}
		patterns = append(patterns, re)
	}

	pm.mu.Lock()
	pm.patterns = patterns
	pm.mu.Unlock()

	return nil
}

// Match checks text against all compiled patterns.
func (pm *PatternMatcher) Match(ctx context.Context, text string) (bool, []InjectionCategory, error) {
	pm.mu.RLock()
	patterns := pm.patterns
	pm.mu.RUnlock()

	var categories []InjectionCategory
	matched := false

	for _, re := range patterns {
		isMatch, err := re.MatchString(text)
		if err != nil {
			continue
		}
		if isMatch {
			matched = true
			categories = append(categories, categorizePattern(re.String()))
		}
	}

	return matched, categories, nil
}

// Reload recompiles patterns from source files.
func (pm *PatternMatcher) Reload() error {
	return pm.loadPatterns()
}

// PerplexityAnalyzer implements L2 statistical anomaly detection.
type PerplexityAnalyzer struct {
	config PerplexityConfig
}

// NewPerplexityAnalyzer creates a new perplexity analyzer.
func NewPerplexityAnalyzer(config PerplexityConfig) *PerplexityAnalyzer {
	return &PerplexityAnalyzer{config: config}
}

// Analyze computes a perplexity-like score for the text.
func (pa *PerplexityAnalyzer) Analyze(ctx context.Context, text string) (float64, bool) {
	if len(text) == 0 {
		return 0, false
	}

	runes := []rune(text)
	n := float64(len(runes))

	freq := make(map[rune]int)
	for _, r := range runes {
		freq[r]++
	}

	entropy := 0.0
	for _, count := range freq {
		p := float64(count) / n
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}

	var score float64
	if entropy > 5.0 {
		score = (entropy - 5.0) / 3.0
	} else if entropy < 2.0 {
		score = (2.0 - entropy) / 2.0
	}

	specialCount := 0
	alphaCount := 0
	for _, r := range runes {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			alphaCount++
		} else if (r >= 33 && r <= 47) || (r >= 58 && r <= 64) {
			specialCount++
		}
	}

	if alphaCount > 0 {
		specialRatio := float64(specialCount) / float64(alphaCount)
		if specialRatio > 0.5 {
			if specialRatio > score {
				score = specialRatio
			}
		}
	}

	return score, score > pa.config.Threshold
}
