package mcp

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/thearchitectit/guardrail-mcp/internal/models"
)

// generateAdvisorResponse creates a contextual response from an advisor
func generateAdvisorResponse(advisor models.Advisor, context string, filePaths []string) models.AdvisorConsultResult {
	// Base response
	result := models.AdvisorConsultResult{
		AdvisorID:    advisor.ID,
		AdvisorName:  advisor.Name,
		Alias:        advisor.Alias,
		Enforcement:  advisor.EnforcementLevel,
		PersonaVoice: advisor.PersonaVoice,
	}

	// Generate contextual advice based on advisor type
	switch advisor.ID {
	case "advisor-resilience":
		result = generateResilienceAdvice(advisor, context, filePaths, result)
	case "advisor-supply-chain":
		result = generateSupplyChainAdvice(advisor, context, filePaths, result)
	case "advisor-privacy":
		result = generatePrivacyAdvice(advisor, context, filePaths, result)
	case "advisor-api":
		result = generateAPIAdvice(advisor, context, filePaths, result)
	case "advisor-perf":
		result = generatePerfAdvice(advisor, context, filePaths, result)
	case "advisor-a11y":
		result = generateA11yAdvice(advisor, context, filePaths, result)
	case "advisor-audit":
		result = generateAuditAdvice(advisor, context, filePaths, result)
	case "advisor-cost":
		result = generateCostAdvice(advisor, context, filePaths, result)
	case "advisor-dx":
		result = generateDXAdvice(advisor, context, filePaths, result)
	default:
		result.Message = fmt.Sprintf("%s is reviewing your changes.", advisor.Name)
		result.Severity = "info"
		result.Recommendations = []string{"Review advisor documentation for specific guidance"}
	}

	return result
}

// generateResilienceAdvice creates resilience-specific advice
func generateResilienceAdvice(advisor models.Advisor, context string, filePaths []string, result models.AdvisorConsultResult) models.AdvisorConsultResult {
	result.Severity = "critical"
	result.Message = fmt.Sprintf("%s: I see changes that may affect system resilience. Let me review...", advisor.Alias)

	// Check for common resilience patterns
	hasRetry := false
	hasCircuitBreaker := false
	hasTimeout := false

	for _, path := range filePaths {
		content := readFileIfExists(path)
		if strings.Contains(content, "retry") || strings.Contains(content, "Retry") {
			hasRetry = true
		}
		if strings.Contains(content, "circuit") || strings.Contains(content, "Circuit") {
			hasCircuitBreaker = true
		}
		if strings.Contains(content, "timeout") || strings.Contains(content, "Timeout") {
			hasTimeout = true
		}
	}

	recommendations := []string{}

	if hasRetry && !hasCircuitBreaker {
		result.Severity = "critical"
		result.Message = fmt.Sprintf("%s: You have retry logic but NO CIRCUIT BREAKER. If the service is down, you'll exhaust connection pools.", advisor.Alias)
		recommendations = append(recommendations,
			"Add circuit breaker with 50% threshold",
			"Implement fallback to queue for async processing",
			"Add health check endpoint",
		)
		result.References = []string{"https://martinfowler.com/bliki/CircuitBreaker.html"}
	} else if !hasTimeout {
		result.Severity = "warning"
		result.Message = fmt.Sprintf("%s: No timeout configuration detected. This could lead to hanging requests.", advisor.Alias)
		recommendations = append(recommendations,
			"Add explicit timeout configuration",
			"Set reasonable defaults (e.g., 5s for HTTP)",
		)
	} else {
		result.Severity = "info"
		result.Message = fmt.Sprintf("%s: Good! I see timeout and retry patterns. Consider adding health checks if not present.", advisor.Alias)
		recommendations = append(recommendations,
			"Verify health check endpoints exist",
			"Consider adding bulkhead pattern for isolation",
		)
	}

	result.Recommendations = recommendations
	return result
}

// generateSupplyChainAdvice creates supply chain advice
func generateSupplyChainAdvice(advisor models.Advisor, context string, filePaths []string, result models.AdvisorConsultResult) models.AdvisorConsultResult {
	result.Severity = "high"
	result.Message = fmt.Sprintf("%s: Checking your dependencies for CVEs and maintenance status...", advisor.Alias)

	// Check for package files
	hasPackageJSON := false
	hasGoMod := false
	hasRequirements := false

	for _, path := range filePaths {
		if strings.Contains(path, "package.json") {
			hasPackageJSON = true
		}
		if strings.Contains(path, "go.mod") || strings.Contains(path, "go.sum") {
			hasGoMod = true
		}
		if strings.Contains(path, "requirements.txt") {
			hasRequirements = true
		}
	}

	if hasPackageJSON || hasGoMod || hasRequirements {
		result.Message = fmt.Sprintf("%s: Dependency file changes detected. Checking for security and maintenance risks.", advisor.Alias)
		result.Recommendations = []string{
			"Run 'npm audit' or equivalent for CVE scanning",
			"Verify transitive dependencies haven't introduced new CVEs",
			"Check last commit date of new dependencies",
			"Review license compatibility",
		}
	} else {
		result.Severity = "info"
		result.Message = fmt.Sprintf("%s: No dependency changes detected in this update.", advisor.Alias)
	}

	return result
}

// generatePrivacyAdvice creates privacy-specific advice
func generatePrivacyAdvice(advisor models.Advisor, context string, filePaths []string, result models.AdvisorConsultResult) models.AdvisorConsultResult {
	result.Severity = "critical"
	result.Message = fmt.Sprintf("%s: Checking for PII and data privacy compliance...", advisor.Alias)

	// Check for PII patterns
	piiPatterns := []string{"email", "phone", "ssn", "password", "credit_card"}
	foundPII := false

	for _, path := range filePaths {
		content := readFileIfExists(path)
		for _, pattern := range piiPatterns {
			if strings.Contains(strings.ToLower(content), pattern) {
				foundPII = true
				break
			}
		}
	}

	if foundPII {
		result.Message = fmt.Sprintf("%s: PII fields detected. Ensure data minimization and consent management.", advisor.Alias)
		result.Recommendations = []string{
			"Verify only necessary PII is collected",
			"Check retention policies are defined",
			"Ensure encryption at rest and in transit",
			"Verify GDPR/CCPA compliance",
		}
	} else {
		result.Severity = "info"
		result.Message = fmt.Sprintf("%s: No obvious PII patterns detected. Continue with privacy review.", advisor.Alias)
	}

	return result
}

// generateAPIAdvice creates API-specific advice
func generateAPIAdvice(advisor models.Advisor, context string, filePaths []string, result models.AdvisorConsultResult) models.AdvisorConsultResult {
	result.Severity = "warning"
	result.Message = fmt.Sprintf("%s: Reviewing API changes for breaking changes...", advisor.Alias)

	// Check for API file patterns
	isAPIChange := false
	for _, path := range filePaths {
		if strings.Contains(path, "api") || strings.Contains(path, "endpoint") ||
			strings.Contains(path, "route") || strings.Contains(path, "handler") {
			isAPIChange = true
			break
		}
	}

	if isAPIChange {
		result.Recommendations = []string{
			"Verify no required fields were added to existing responses",
			"Check for breaking changes in URL patterns",
			"Ensure proper API versioning",
			"Update API documentation if schema changed",
		}
	} else {
		result.Severity = "info"
		result.Message = fmt.Sprintf("%s: No API-related changes detected.", advisor.Alias)
	}

	return result
}

// generatePerfAdvice creates performance-specific advice
func generatePerfAdvice(advisor models.Advisor, context string, filePaths []string, result models.AdvisorConsultResult) models.AdvisorConsultResult {
	result.Severity = "warning"
	result.Message = fmt.Sprintf("%s: Checking for performance anti-patterns...", advisor.Alias)

	// Check for query patterns
	hasQuery := false
	for _, path := range filePaths {
		content := readFileIfExists(path)
		if strings.Contains(content, "SELECT") || strings.Contains(content, "query") {
			hasQuery = true
			break
		}
	}

	if hasQuery {
		result.Recommendations = []string{
			"Check for N+1 query patterns",
			"Verify indexes exist for query patterns",
			"Consider caching for hot paths",
			"Add query timing instrumentation",
		}
	} else {
		result.Severity = "info"
		result.Message = fmt.Sprintf("%s: No database query changes detected.", advisor.Alias)
	}

	return result
}

// generateA11yAdvice creates accessibility-specific advice
func generateA11yAdvice(advisor models.Advisor, context string, filePaths []string, result models.AdvisorConsultResult) models.AdvisorConsultResult {
	result.Severity = "warning"
	result.Message = fmt.Sprintf("%s: Checking for accessibility compliance...", advisor.Alias)

	// Check for UI file patterns
	isUIChange := false
	for _, path := range filePaths {
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".html" || ext == ".jsx" || ext == ".tsx" || ext == ".vue" {
			isUIChange = true
			break
		}
	}

	if isUIChange {
		result.Recommendations = []string{
			"Ensure all interactive elements have accessible labels",
			"Verify keyboard navigation works",
			"Check color contrast ratios",
			"Test with screen reader if possible",
		}
	} else {
		result.Severity = "info"
		result.Message = fmt.Sprintf("%s: No UI component changes detected.", advisor.Alias)
	}

	return result
}

// generateAuditAdvice creates audit-specific advice
func generateAuditAdvice(advisor models.Advisor, context string, filePaths []string, result models.AdvisorConsultResult) models.AdvisorConsultResult {
	result.Severity = "critical"
	result.Message = fmt.Sprintf("%s: Checking for audit logging and compliance...", advisor.Alias)

	// Check for audit patterns
	hasAuditLog := false
	for _, path := range filePaths {
		content := readFileIfExists(path)
		if strings.Contains(content, "audit") || strings.Contains(content, "log") {
			hasAuditLog = true
			break
		}
	}

	if !hasAuditLog {
		result.Message = fmt.Sprintf("%s: Changes affecting data access should include audit logging.", advisor.Alias)
		result.Recommendations = []string{
			"Add audit logging for sensitive operations",
			"Ensure logs are immutable",
			"Include user identity and timestamp in logs",
			"Verify log retention policies",
		}
	} else {
		result.Severity = "info"
		result.Message = fmt.Sprintf("%s: Audit logging detected. Verify completeness.", advisor.Alias)
	}

	return result
}

// generateCostAdvice creates cost-specific advice
func generateCostAdvice(advisor models.Advisor, context string, filePaths []string, result models.AdvisorConsultResult) models.AdvisorConsultResult {
	result.Severity = "warning"
	result.Message = fmt.Sprintf("%s: Reviewing for cost optimization opportunities...", advisor.Alias)

	// Check for infrastructure patterns
	hasInfra := false
	for _, path := range filePaths {
		if strings.Contains(path, "tf") || strings.Contains(path, "cloud") ||
			strings.Contains(path, "k8s") || strings.Contains(path, "docker") {
			hasInfra = true
			break
		}
	}

	if hasInfra {
		result.Recommendations = []string{
			"Verify instance types match workload requirements",
			"Check for unused resources",
			"Consider reserved instances for steady workloads",
			"Review auto-scaling policies",
		}
	} else {
		result.Severity = "info"
		result.Message = fmt.Sprintf("%s: No infrastructure changes detected.", advisor.Alias)
	}

	return result
}

// generateDXAdvice creates DX-specific advice
func generateDXAdvice(advisor models.Advisor, context string, filePaths []string, result models.AdvisorConsultResult) models.AdvisorConsultResult {
	result.Severity = "info"
	result.Message = fmt.Sprintf("%s: Considering developer experience impact...", advisor.Alias)

	result.Recommendations = []string{
		"Ensure changes are documented",
		"Consider impact on onboarding time",
		"Verify tooling still works as expected",
		"Update README if setup steps changed",
	}

	return result
}
