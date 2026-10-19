package guardrails

import (
	"context"
	"fmt"
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

// loadPatterns compiles the configured blocklist and custom patterns into the
// matcher. It is fail-closed at the three points this pass fixed:
//
//  1. A blocklist path the operator DECLARED but that cannot be read used to be
//     logged and skipped, so the layer ran with fewer patterns than declared
//     while every scan still returned a verdict. An operator who pointed the
//     detector at a corpus and mistyped the filename had a detector silently
//     enforcing less than the review trail claimed.
//  2. A pattern that failed to COMPILE used to be dropped the same way, so a
//     malformed rule reduced coverage without failing anything.
//  3. An enabled matcher with ZERO compiled patterns was constructible, and a
//     matcher that can never fire reports Safe for text it never evaluated —
//     the same vacuity class this spec family keeps re-creating.
//
// On any error the previously compiled set is left in force: a reload must not
// be able to disarm a running control (the credential registry uses the same
// rule).
func (pm *PatternMatcher) loadPatterns() error {
	var patterns []*regexp2.Regexp

	for _, path := range pm.config.BlocklistPaths {
		lines, err := ReadBlocklistFile(path)
		if err != nil {
			return fmt.Errorf("declared blocklist %q could not be read (Spec 01 Sec. 4.2): %w", path, err)
		}
		for i, line := range lines {
			re, err := regexp2.Compile(line, regexp2.None)
			if err != nil {
				return fmt.Errorf("declared blocklist %q line %d does not compile (Spec 01 Sec. 4.2): %w", path, i+1, err)
			}
			patterns = append(patterns, re)
		}
	}

	for i, pattern := range pm.config.CustomPatterns {
		re, err := regexp2.Compile(pattern, regexp2.None)
		if err != nil {
			return fmt.Errorf("custom pattern %d does not compile (Spec 01 Sec. 3.2): %w", i, err)
		}
		patterns = append(patterns, re)
	}

	if len(patterns) == 0 {
		return fmt.Errorf("L1 pattern matching is enabled with zero compiled patterns; a matcher that can never fire reports Safe for text it never evaluated (Spec 01 Sec. 2.2)")
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

// minPerplexityTextRunes is the shortest text L2 will score.
//
// Character-level statistics on a tiny sample are sampling noise, not signal:
// measured before this floor existed, the single character "x" scored 1.000 and
// tripped the 0.85 threshold, as did "- a\n- b\n- c" (11 runes) and a 30-rune
// markdown table. L2 is the cheap-and-broad layer; a layer that answers "anomalous"
// for a three-item list teaches operators to ignore it. Below the floor L2 abstains
// (score 0, not anomalous) and the result records no L2 evaluation.
const minPerplexityTextRunes = 32

// Analyze computes a perplexity-like score for the text, in [0,1].
//
// The score MUST be bounded to [0,1]: Spec 01 Sec. 3.2 documents
// `perplexity: threshold: 0.85  # above this = suspicious`, and a score that can
// exceed 1 makes every comparison against that number meaningless. Before this
// pass the special-character branch divided by the ALPHA count rather than by the
// text length, so it was unbounded: a 30-rune markdown table scored 3.000, a
// 531-rune JSON array scored 2.492, and with L2 enabled by default (2026-10-19)
// both returned Safe=false and, for tool output, decision=block — ordinary
// structured tool output refused as an injection.
func (pa *PerplexityAnalyzer) Analyze(ctx context.Context, text string) (float64, bool) {
	runes := []rune(text)
	if len(runes) < minPerplexityTextRunes {
		return 0, false
	}

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
		// Dense, high-entropy text: encoded payloads look like this.
		score = (entropy - 5.0) / 3.0
	} else if entropy < 2.0 {
		// Degenerate repetition (a long run of one or two characters).
		score = (2.0 - entropy) / 2.0
	}

	specialCount := 0
	for _, r := range runes {
		if (r >= 33 && r <= 47) || (r >= 58 && r <= 64) {
			specialCount++
		}
	}

	// Special characters are measured as a FRACTION OF THE TEXT, not as a ratio
	// against the letter count. The threshold is documented as "above this =
	// suspicious" on a 0..1 scale, so the quantity compared to it must live on
	// that scale: 0.85 means "85% of the characters are punctuation", which no
	// ordinary prose, JSON, SQL or Markdown reaches.
	if fraction := float64(specialCount) / n; fraction > score {
		score = fraction
	}

	if score > 1 {
		score = 1
	}

	return score, score > pa.config.Threshold
}
