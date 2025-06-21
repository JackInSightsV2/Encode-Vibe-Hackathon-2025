package config

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// PredefinedTestScenarios contains predefined test scenarios for different environments
type PredefinedTestScenarios struct {
	scenarios map[string]*TestScenario
}

// TestScenario represents a complete test scenario
type TestScenario struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Environment string                 `json:"environment"`
	Categories  []string               `json:"categories"`
	Tests       []TestDefinition       `json:"tests"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

// TestDefinition defines a specific test to run
type TestDefinition struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Category    string                 `json:"category"`
	Critical    bool                   `json:"critical"`
	Timeout     time.Duration          `json:"timeout"`
	Retry       int                    `json:"retry"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// NewPredefinedTestScenarios creates predefined test scenarios
func NewPredefinedTestScenarios() *PredefinedTestScenarios {
	pts := &PredefinedTestScenarios{
		scenarios: make(map[string]*TestScenario),
	}
	
	// Register all predefined scenarios
	pts.registerDevelopmentScenario()
	pts.registerProductionScenario()
	pts.registerMinimalScenario()
	pts.registerFullScenario()
	pts.registerSecurityFocusedScenario()
	pts.registerPerformanceScenario()
	
	return pts
}

// GetScenario returns a specific test scenario
func (pts *PredefinedTestScenarios) GetScenario(name string) (*TestScenario, error) {
	scenario, exists := pts.scenarios[name]
	if !exists {
		return nil, fmt.Errorf("scenario '%s' not found", name)
	}
	return scenario, nil
}

// ListScenarios returns all available scenarios
func (pts *PredefinedTestScenarios) ListScenarios() []string {
	scenarios := make([]string, 0, len(pts.scenarios))
	for name := range pts.scenarios {
		scenarios = append(scenarios, name)
	}
	return scenarios
}

// registerDevelopmentScenario registers tests for development environment
func (pts *PredefinedTestScenarios) registerDevelopmentScenario() {
	pts.scenarios["development"] = &TestScenario{
		Name:        "Development Environment Tests",
		Description: "Basic tests suitable for local development",
		Environment: "development",
		Categories:  []string{"server", "providers", "database"},
		Tests: []TestDefinition{
			{
				ID:       "dev_server_port",
				Name:     "Server Port Availability",
				Category: "server",
				Critical: true,
				Timeout:  5 * time.Second,
			},
			{
				ID:       "dev_local_provider",
				Name:     "Local Provider Connectivity",
				Category: "providers",
				Critical: false,
				Timeout:  10 * time.Second,
				Parameters: map[string]interface{}{
					"provider_types": []string{"local"},
				},
			},
			{
				ID:       "dev_database",
				Name:     "Development Database",
				Category: "database",
				Critical: true,
				Timeout:  5 * time.Second,
				Parameters: map[string]interface{}{
					"expected_type": "sqlite",
				},
			},
		},
	}
}

// registerProductionScenario registers tests for production environment
func (pts *PredefinedTestScenarios) registerProductionScenario() {
	pts.scenarios["production"] = &TestScenario{
		Name:        "Production Environment Tests",
		Description: "Comprehensive tests for production deployment",
		Environment: "production",
		Categories:  []string{"server", "providers", "database", "security", "metrics"},
		Tests: []TestDefinition{
			{
				ID:       "prod_server_config",
				Name:     "Production Server Configuration",
				Category: "server",
				Critical: true,
				Timeout:  5 * time.Second,
				Parameters: map[string]interface{}{
					"require_ssl": true,
					"min_port":    1024,
				},
			},
			{
				ID:       "prod_provider_auth",
				Name:     "Provider Authentication",
				Category: "providers",
				Critical: true,
				Timeout:  10 * time.Second,
				Retry:    2,
				Parameters: map[string]interface{}{
					"require_api_keys": true,
					"test_endpoints":   true,
				},
			},
			{
				ID:       "prod_database_pool",
				Name:     "Database Connection Pool",
				Category: "database",
				Critical: true,
				Timeout:  15 * time.Second,
				Parameters: map[string]interface{}{
					"expected_type":    "postgresql",
					"min_connections":  5,
					"max_connections":  100,
				},
			},
			{
				ID:       "prod_rate_limiting",
				Name:     "Rate Limiting Configuration",
				Category: "security",
				Critical: true,
				Timeout:  5 * time.Second,
				Parameters: map[string]interface{}{
					"require_enabled": true,
					"test_storage":    true,
				},
			},
			{
				ID:       "prod_cors_policy",
				Name:     "CORS Security Policy",
				Category: "security",
				Critical: true,
				Timeout:  5 * time.Second,
				Parameters: map[string]interface{}{
					"disallow_wildcard": true,
				},
			},
			{
				ID:       "prod_metrics_storage",
				Name:     "Metrics Storage Backend",
				Category: "metrics",
				Critical: false,
				Timeout:  10 * time.Second,
			},
		},
	}
}

// registerMinimalScenario registers minimal required tests
func (pts *PredefinedTestScenarios) registerMinimalScenario() {
	pts.scenarios["minimal"] = &TestScenario{
		Name:        "Minimal Configuration Tests",
		Description: "Bare minimum tests to ensure basic functionality",
		Environment: "any",
		Categories:  []string{"server", "providers"},
		Tests: []TestDefinition{
			{
				ID:       "min_server_start",
				Name:     "Server Can Start",
				Category: "server",
				Critical: true,
				Timeout:  5 * time.Second,
			},
			{
				ID:       "min_provider_exists",
				Name:     "At Least One Provider",
				Category: "providers",
				Critical: true,
				Timeout:  5 * time.Second,
			},
		},
	}
}

// registerFullScenario registers all possible tests
func (pts *PredefinedTestScenarios) registerFullScenario() {
	pts.scenarios["full"] = &TestScenario{
		Name:        "Full Test Suite",
		Description: "Complete test suite covering all features",
		Environment: "any",
		Categories:  []string{"server", "providers", "database", "security", "metrics", "moderation"},
		Tests: []TestDefinition{
			// Server tests
			{
				ID:       "full_server_port",
				Name:     "Server Port Configuration",
				Category: "server",
				Critical: true,
				Timeout:  5 * time.Second,
			},
			{
				ID:       "full_server_host",
				Name:     "Server Host Resolution",
				Category: "server",
				Critical: true,
				Timeout:  5 * time.Second,
			},
			// Provider tests
			{
				ID:       "full_provider_connectivity",
				Name:     "All Provider Connectivity",
				Category: "providers",
				Critical: true,
				Timeout:  30 * time.Second,
			},
			{
				ID:       "full_provider_models",
				Name:     "Provider Model Validation",
				Category: "providers",
				Critical: false,
				Timeout:  10 * time.Second,
			},
			{
				ID:       "full_provider_health",
				Name:     "Provider Health Checks",
				Category: "providers",
				Critical: false,
				Timeout:  20 * time.Second,
			},
			// Database tests
			{
				ID:       "full_db_connection",
				Name:     "Database Connection",
				Category: "database",
				Critical: true,
				Timeout:  10 * time.Second,
			},
			{
				ID:       "full_db_permissions",
				Name:     "Database Permissions",
				Category: "database",
				Critical: true,
				Timeout:  10 * time.Second,
			},
			{
				ID:       "full_db_migrations",
				Name:     "Database Migrations",
				Category: "database",
				Critical: false,
				Timeout:  30 * time.Second,
			},
			// Security tests
			{
				ID:       "full_security_headers",
				Name:     "Security Headers",
				Category: "security",
				Critical: true,
				Timeout:  5 * time.Second,
			},
			{
				ID:       "full_input_validation",
				Name:     "Input Validation Rules",
				Category: "security",
				Critical: true,
				Timeout:  5 * time.Second,
			},
			{
				ID:       "full_ddos_protection",
				Name:     "DDoS Protection",
				Category: "security",
				Critical: false,
				Timeout:  10 * time.Second,
			},
			// Moderation tests
			{
				ID:       "full_moderation_layers",
				Name:     "Moderation Layers",
				Category: "moderation",
				Critical: false,
				Timeout:  15 * time.Second,
			},
		},
	}
}

// registerSecurityFocusedScenario registers security-focused tests
func (pts *PredefinedTestScenarios) registerSecurityFocusedScenario() {
	pts.scenarios["security"] = &TestScenario{
		Name:        "Security-Focused Tests",
		Description: "Comprehensive security validation",
		Environment: "any",
		Categories:  []string{"security", "providers"},
		Tests: []TestDefinition{
			{
				ID:       "sec_rate_limits",
				Name:     "Rate Limiting Rules",
				Category: "security",
				Critical: true,
				Timeout:  10 * time.Second,
				Parameters: map[string]interface{}{
					"test_all_endpoints": true,
					"verify_limits":      true,
				},
			},
			{
				ID:       "sec_cors_strict",
				Name:     "Strict CORS Policy",
				Category: "security",
				Critical: true,
				Timeout:  5 * time.Second,
				Parameters: map[string]interface{}{
					"disallow_wildcard":     true,
					"require_https_origins": true,
				},
			},
			{
				ID:       "sec_input_sanitization",
				Name:     "Input Sanitization",
				Category: "security",
				Critical: true,
				Timeout:  10 * time.Second,
				Parameters: map[string]interface{}{
					"test_xss":           true,
					"test_sql_injection": true,
					"test_path_traversal": true,
				},
			},
			{
				ID:       "sec_api_keys",
				Name:     "API Key Security",
				Category: "providers",
				Critical: true,
				Timeout:  5 * time.Second,
				Parameters: map[string]interface{}{
					"check_key_strength":   true,
					"detect_placeholders":  true,
					"require_encryption":   true,
				},
			},
			{
				ID:       "sec_ip_protection",
				Name:     "IP Protection Rules",
				Category: "security",
				Critical: false,
				Timeout:  10 * time.Second,
				Parameters: map[string]interface{}{
					"test_geoblocking":    true,
					"test_ip_reputation":  true,
				},
			},
			{
				ID:       "sec_prompt_injection",
				Name:     "Prompt Injection Protection",
				Category: "security",
				Critical: true,
				Timeout:  5 * time.Second,
				Parameters: map[string]interface{}{
					"verify_patterns":    true,
					"test_effectiveness": true,
				},
			},
		},
	}
}

// registerPerformanceScenario registers performance-focused tests
func (pts *PredefinedTestScenarios) registerPerformanceScenario() {
	pts.scenarios["performance"] = &TestScenario{
		Name:        "Performance Tests",
		Description: "Configuration optimization for performance",
		Environment: "any",
		Categories:  []string{"providers", "database", "metrics"},
		Tests: []TestDefinition{
			{
				ID:       "perf_provider_timeouts",
				Name:     "Provider Timeout Configuration",
				Category: "providers",
				Critical: false,
				Timeout:  10 * time.Second,
				Parameters: map[string]interface{}{
					"max_timeout":       120 * time.Second,
					"warn_high_timeout": true,
				},
			},
			{
				ID:       "perf_connection_pools",
				Name:     "Connection Pool Sizing",
				Category: "database",
				Critical: false,
				Timeout:  10 * time.Second,
				Parameters: map[string]interface{}{
					"check_pool_size":     true,
					"warn_oversized_pool": true,
				},
			},
			{
				ID:       "perf_cache_config",
				Name:     "Cache Configuration",
				Category: "metrics",
				Critical: false,
				Timeout:  5 * time.Second,
				Parameters: map[string]interface{}{
					"check_cache_size":   true,
					"check_ttl_settings": true,
				},
			},
			{
				ID:       "perf_rate_limit_algo",
				Name:     "Rate Limiting Algorithm",
				Category: "security",
				Critical: false,
				Timeout:  5 * time.Second,
				Parameters: map[string]interface{}{
					"recommend_algorithm": true,
					"check_burst_size":    true,
				},
			},
		},
	}
}

// RunScenario executes a specific test scenario
func RunScenario(config *Config, scenarioName string) (*TestResult, error) {
	scenarios := NewPredefinedTestScenarios()
	scenario, err := scenarios.GetScenario(scenarioName)
	if err != nil {
		return nil, err
	}
	
	// Create a custom tester for the scenario
	tester := NewConfigTester(config)
	
	// Create test result
	result := &TestResult{
		Success:  true,
		Tests:    []TestCase{},
		TestedAt: time.Now(),
	}
	
	// Run each test in the scenario
	for _, testDef := range scenario.Tests {
		testCase := runTestDefinition(tester, testDef)
		result.Tests = append(result.Tests, testCase)
		
		switch testCase.Status {
		case "passed":
			result.PassedCount++
		case "failed":
			result.FailedCount++
			if testDef.Critical {
				result.Success = false
			}
		case "skipped":
			result.SkippedCount++
		}
	}
	
	result.Summary = fmt.Sprintf("Scenario '%s': %d passed, %d failed, %d skipped",
		scenarioName, result.PassedCount, result.FailedCount, result.SkippedCount)
	
	return result, nil
}

// runTestDefinition executes a single test definition
func runTestDefinition(tester *ConfigTester, testDef TestDefinition) TestCase {
	ctx := context.Background()
	if testDef.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, testDef.Timeout)
		defer cancel()
	}
	
	// Map test ID to actual test function
	// This is a simplified implementation
	testCase := TestCase{
		Name:        testDef.Name,
		Category:    testDef.Category,
		Description: fmt.Sprintf("Test %s", testDef.ID),
		Status:      "skipped",
		Message:     "Test implementation pending",
	}
	
	// In a real implementation, you would map test IDs to actual test functions
	// For now, we'll just run the category-based tests
	categoryResult := tester.TestSpecific([]string{testDef.Category})
	if len(categoryResult.Tests) > 0 {
		// Use the first matching test result
		testCase = categoryResult.Tests[0]
	}
	
	return testCase
}

// CreateCustomScenario creates a custom test scenario
func CreateCustomScenario(name, description string, tests []TestDefinition) *TestScenario {
	categories := make(map[string]bool)
	for _, test := range tests {
		categories[test.Category] = true
	}
	
	categoryList := make([]string, 0, len(categories))
	for cat := range categories {
		categoryList = append(categoryList, cat)
	}
	
	return &TestScenario{
		Name:        name,
		Description: description,
		Environment: "custom",
		Categories:  categoryList,
		Tests:       tests,
	}
}

// ExportScenario exports a test scenario to JSON
func ExportScenario(scenario *TestScenario) ([]byte, error) {
	return json.MarshalIndent(scenario, "", "  ")
}

// ImportScenario imports a test scenario from JSON
func ImportScenario(data []byte) (*TestScenario, error) {
	var scenario TestScenario
	if err := json.Unmarshal(data, &scenario); err != nil {
		return nil, fmt.Errorf("failed to parse scenario: %w", err)
	}
	return &scenario, nil
}

// GenerateTestReport generates a detailed test report
func GenerateTestReport(result *TestResult, format string) (string, error) {
	switch format {
	case "json":
		data, err := json.MarshalIndent(result, "", "  ")
		return string(data), err
		
	case "markdown":
		return generateMarkdownReport(result), nil
		
	case "text":
		return generateTextReport(result), nil
		
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

func generateMarkdownReport(result *TestResult) string {
	var sb strings.Builder
	
	sb.WriteString("# Configuration Test Report\n\n")
	sb.WriteString(fmt.Sprintf("**Date:** %s\n", result.TestedAt.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("**Duration:** %s\n", result.Duration))
	sb.WriteString(fmt.Sprintf("**Result:** %s\n\n", result.Summary))
	
	sb.WriteString("## Test Results\n\n")
	sb.WriteString("| Test | Category | Status | Duration | Message |\n")
	sb.WriteString("|------|----------|--------|----------|----------|\n")
	
	for _, test := range result.Tests {
		status := test.Status
		if status == "failed" {
			status = "❌ " + status
		} else if status == "passed" {
			status = "✅ " + status
		} else {
			status = "⏭️ " + status
		}
		
		message := test.Message
		if test.Error != "" {
			message = test.Error
		}
		
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n",
			test.Name,
			test.Category,
			status,
			test.Duration,
			message,
		))
	}
	
	return sb.String()
}

func generateTextReport(result *TestResult) string {
	var sb strings.Builder
	
	sb.WriteString("Configuration Test Report\n")
	sb.WriteString("========================\n\n")
	sb.WriteString(fmt.Sprintf("Date: %s\n", result.TestedAt.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("Duration: %s\n", result.Duration))
	sb.WriteString(fmt.Sprintf("Result: %s\n\n", result.Summary))
	
	for _, test := range result.Tests {
		sb.WriteString(fmt.Sprintf("[%s] %s\n", strings.ToUpper(test.Status), test.Name))
		sb.WriteString(fmt.Sprintf("  Category: %s\n", test.Category))
		sb.WriteString(fmt.Sprintf("  Duration: %s\n", test.Duration))
		
		if test.Message != "" {
			sb.WriteString(fmt.Sprintf("  Message: %s\n", test.Message))
		}
		if test.Error != "" {
			sb.WriteString(fmt.Sprintf("  Error: %s\n", test.Error))
		}
		sb.WriteString("\n")
	}
	
	return sb.String()
}