package validation

import (
	"fmt"
	"regexp"
	"time"
)

// SafeRegex performs regex matching with timeout protection
func SafeRegex(pattern string, input string, timeout time.Duration) (bool, error) {
	resultChan := make(chan bool, 1)
	panicChan := make(chan interface{}, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				panicChan <- r
			}
		}()

		re, err := regexp.Compile(pattern)
		if err != nil {
			resultChan <- false
			return
		}
		resultChan <- re.MatchString(input)
	}()

	select {
	case result := <-resultChan:
		return result, nil
	case r := <-panicChan:
		return false, fmt.Errorf("regex panic: %v", r)
	case <-time.After(timeout):
		return false, fmt.Errorf("regex timeout after %v - possible ReDoS attack", timeout)
	}
}

// ValidatePattern checks if a regex pattern is valid and safe
func ValidatePattern(pattern string) error {
	// Check pattern length
	if len(pattern) > 10000 {
		return fmt.Errorf("pattern too long (max 10000 chars)")
	}

	// Try to compile
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex pattern: %w", err)
	}

	// Check for potentially dangerous patterns (nested quantifiers)
	dangerous := regexp.MustCompile(`\*\+|\+\*|\?\?|\{[^}]+\}\{[^}]+\}`)
	if dangerous.MatchString(pattern) {
		return fmt.Errorf("pattern contains potentially dangerous nested quantifiers")
	}

	// Test with simple input to ensure it works
	testInput := "test string for validation"
	_, err = SafeRegex(pattern, testInput, 100*time.Millisecond)
	if err != nil {
		return fmt.Errorf("pattern failed validation test: %w", err)
	}

	// Ensure the compiled regex is usable
	_ = re.String() // Just to use the variable

	return nil
}

// MatchPattern safely matches a pattern against input
func MatchPattern(pattern string, input string) (bool, error) {
	return SafeRegex(pattern, input, 100*time.Millisecond)
}
