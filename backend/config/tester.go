package config

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
	
	_ "github.com/lib/pq"      // PostgreSQL driver
	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// ConfigTester tests configuration for connectivity and functionality
type ConfigTester struct {
	config     *Config
	httpClient *http.Client
	testCases  map[string]TestCaseFunc
	context    context.Context
}

// TestCaseFunc is a function that performs a specific test
type TestCaseFunc func(ctx context.Context) *TestCase

// NewConfigTester creates a new configuration tester
func NewConfigTester(config *Config) *ConfigTester {
	ct := &ConfigTester{
		config: config,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: false,
				},
			},
		},
		testCases: make(map[string]TestCaseFunc),
		context:   context.Background(),
	}
	
	// Register test cases
	ct.registerTestCases()
	
	return ct
}

// Test runs all configuration tests
func (ct *ConfigTester) Test() *TestResult {
	return ct.TestWithContext(ct.context)
}

// TestWithContext runs all configuration tests with a context
func (ct *ConfigTester) TestWithContext(ctx context.Context) *TestResult {
	start := time.Now()
	
	result := &TestResult{
		Success:  true,
		Tests:    []TestCase{},
		TestedAt: time.Now(),
	}
	
	// Run all test cases
	for name, testFunc := range ct.testCases {
		testCase := testFunc(ctx)
		if testCase == nil {
			continue
		}
		
		testCase.Name = name
		result.Tests = append(result.Tests, *testCase)
		
		switch testCase.Status {
		case "passed":
			result.PassedCount++
		case "failed":
			result.FailedCount++
			result.Success = false
		case "skipped":
			result.SkippedCount++
		}
	}
	
	result.Duration = time.Since(start)
	result.Summary = ct.generateTestSummary(result)
	
	return result
}

// TestSpecific runs specific test categories
func (ct *ConfigTester) TestSpecific(categories []string) *TestResult {
	start := time.Now()
	
	result := &TestResult{
		Success:  true,
		Tests:    []TestCase{},
		TestedAt: time.Now(),
	}
	
	// Create a map for quick category lookup
	categoryMap := make(map[string]bool)
	for _, cat := range categories {
		categoryMap[cat] = true
	}
	
	// Run test cases matching the categories
	for name, testFunc := range ct.testCases {
		testCase := testFunc(ct.context)
		if testCase == nil {
			continue
		}
		
		// Check if test category matches
		if !categoryMap[testCase.Category] {
			continue
		}
		
		testCase.Name = name
		result.Tests = append(result.Tests, *testCase)
		
		switch testCase.Status {
		case "passed":
			result.PassedCount++
		case "failed":
			result.FailedCount++
			result.Success = false
		case "skipped":
			result.SkippedCount++
		}
	}
	
	result.Duration = time.Since(start)
	result.Summary = ct.generateTestSummary(result)
	
	return result
}

// registerTestCases registers all test cases
func (ct *ConfigTester) registerTestCases() {
	// Server tests
	ct.testCases["server_port_available"] = ct.testServerPort
	ct.testCases["server_host_resolves"] = ct.testServerHost
	
	// Provider tests
	ct.testCases["provider_connectivity"] = ct.testProviderConnectivity
	ct.testCases["provider_authentication"] = ct.testProviderAuth
	ct.testCases["provider_models"] = ct.testProviderModels
	
	// Database tests
	ct.testCases["database_connection"] = ct.testDatabaseConnection
	ct.testCases["database_permissions"] = ct.testDatabasePermissions
	
	// Security tests
	ct.testCases["rate_limit_storage"] = ct.testRateLimitStorage
	ct.testCases["cors_configuration"] = ct.testCORSConfig
	
	// External services tests
	ct.testCases["metrics_storage"] = ct.testMetricsStorage
	ct.testCases["supabase_connection"] = ct.testSupabaseConnection
}

// Server test cases

func (ct *ConfigTester) testServerPort(ctx context.Context) *TestCase {
	tc := &TestCase{
		Description: "Test if server port is available",
		Category:    "server",
		Status:      "passed",
	}
	
	start := time.Now()
	defer func() {
		tc.Duration = time.Since(start)
	}()
	
	// Check if port is in valid range
	if ct.config.Server.Port < 1 || ct.config.Server.Port > 65535 {
		tc.Status = "failed"
		tc.Error = fmt.Sprintf("Invalid port number: %d", ct.config.Server.Port)
		return tc
	}
	
	// Try to listen on the port
	addr := fmt.Sprintf("%s:%d", ct.config.Server.Host, ct.config.Server.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		tc.Status = "failed"
		tc.Error = fmt.Sprintf("Port %d is not available: %v", ct.config.Server.Port, err)
		return tc
	}
	listener.Close()
	
	tc.Message = fmt.Sprintf("Port %d is available", ct.config.Server.Port)
	return tc
}

func (ct *ConfigTester) testServerHost(ctx context.Context) *TestCase {
	tc := &TestCase{
		Description: "Test if server host resolves correctly",
		Category:    "server",
		Status:      "passed",
	}
	
	start := time.Now()
	defer func() {
		tc.Duration = time.Since(start)
	}()
	
	// Special cases that don't need resolution
	if ct.config.Server.Host == "localhost" || ct.config.Server.Host == "0.0.0.0" || ct.config.Server.Host == "::" {
		tc.Message = fmt.Sprintf("Host '%s' is valid", ct.config.Server.Host)
		return tc
	}
	
	// Try to resolve the host
	ips, err := net.LookupHost(ct.config.Server.Host)
	if err != nil {
		tc.Status = "failed"
		tc.Error = fmt.Sprintf("Cannot resolve host '%s': %v", ct.config.Server.Host, err)
		return tc
	}
	
	tc.Message = fmt.Sprintf("Host '%s' resolves to %v", ct.config.Server.Host, ips)
	tc.Metadata = map[string]interface{}{
		"resolved_ips": ips,
	}
	return tc
}

// Provider test cases

func (ct *ConfigTester) testProviderConnectivity(ctx context.Context) *TestCase {
	tc := &TestCase{
		Description: "Test connectivity to AI providers",
		Category:    "providers",
		Status:      "passed",
	}
	
	start := time.Now()
	defer func() {
		tc.Duration = time.Since(start)
	}()
	
	results := make(map[string]string)
	hasFailure := false
	
	// Test legacy providers
	for name, provider := range ct.config.Providers {
		if provider.BaseURL == "" {
			continue
		}
		
		// Parse and validate URL
		u, err := url.Parse(provider.BaseURL)
		if err != nil {
			results[name] = fmt.Sprintf("Invalid URL: %v", err)
			hasFailure = true
			continue
		}
		
		// For local providers, check if they're running
		if u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" {
			conn, err := net.DialTimeout("tcp", u.Host, 2*time.Second)
			if err != nil {
				results[name] = "Local service not running"
				// Don't mark as failure for local services
				continue
			}
			conn.Close()
			results[name] = "Connected"
		} else {
			// For remote providers, just validate URL format
			results[name] = "URL valid"
		}
	}
	
	// Test enhanced providers
	for name, provider := range ct.config.EnhancedProviders {
		if !provider.Enabled || provider.BaseURL == "" {
			continue
		}
		
		// Parse and validate URL
		u, err := url.Parse(provider.BaseURL)
		if err != nil {
			results[name] = fmt.Sprintf("Invalid URL: %v", err)
			hasFailure = true
			continue
		}
		
		// For local providers, check if they're running
		if u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" {
			conn, err := net.DialTimeout("tcp", u.Host, 2*time.Second)
			if err != nil {
				results[name] = "Local service not running"
				continue
			}
			conn.Close()
			results[name] = "Connected"
		} else {
			// For remote providers, just validate URL format
			results[name] = "URL valid"
		}
	}
	
	if hasFailure {
		tc.Status = "failed"
		tc.Error = "Some providers have connectivity issues"
	}
	
	tc.Message = fmt.Sprintf("Tested %d providers", len(results))
	tc.Metadata = map[string]interface{}{
		"provider_results": results,
	}
	
	return tc
}

func (ct *ConfigTester) testProviderAuth(ctx context.Context) *TestCase {
	tc := &TestCase{
		Description: "Test provider API key configuration",
		Category:    "providers",
		Status:      "passed",
	}
	
	start := time.Now()
	defer func() {
		tc.Duration = time.Since(start)
	}()
	
	results := make(map[string]string)
	warnings := []string{}
	
	// Check legacy providers
	for name, provider := range ct.config.Providers {
		if provider.Name == "Local" || provider.Name == "Mock" {
			continue // Skip local providers
		}
		
		if provider.APIKey == "" {
			results[name] = "No API key configured"
			warnings = append(warnings, fmt.Sprintf("Provider %s has no API key", name))
		} else if len(provider.APIKey) < 20 {
			results[name] = "API key seems too short"
			warnings = append(warnings, fmt.Sprintf("Provider %s API key might be invalid", name))
		} else {
			results[name] = "API key configured"
		}
	}
	
	// Check enhanced providers
	for name, provider := range ct.config.EnhancedProviders {
		if !provider.Enabled || provider.Type == ProviderTypeLocal || provider.Type == ProviderTypeMock {
			continue
		}
		
		if provider.APIKey == "" {
			results[name] = "No API key configured"
			warnings = append(warnings, fmt.Sprintf("Provider %s has no API key", name))
		} else if len(provider.APIKey) < 20 {
			results[name] = "API key seems too short"
			warnings = append(warnings, fmt.Sprintf("Provider %s API key might be invalid", name))
		} else {
			results[name] = "API key configured"
		}
	}
	
	if len(warnings) > 0 {
		tc.Status = "failed"
		tc.Error = strings.Join(warnings, "; ")
	}
	
	tc.Message = fmt.Sprintf("Checked %d provider API keys", len(results))
	tc.Metadata = map[string]interface{}{
		"api_key_status": results,
	}
	
	return tc
}

func (ct *ConfigTester) testProviderModels(ctx context.Context) *TestCase {
	tc := &TestCase{
		Description: "Test provider model configuration",
		Category:    "providers",
		Status:      "passed",
	}
	
	start := time.Now()
	defer func() {
		tc.Duration = time.Since(start)
	}()
	
	totalModels := 0
	providerCount := 0
	
	// Count models in legacy providers
	for _, provider := range ct.config.Providers {
		if len(provider.Models) > 0 {
			totalModels += len(provider.Models)
			providerCount++
		}
	}
	
	// Count models in enhanced providers
	for _, provider := range ct.config.EnhancedProviders {
		if provider.Enabled && len(provider.Models) > 0 {
			totalModels += len(provider.Models)
			providerCount++
		}
	}
	
	if totalModels == 0 {
		tc.Status = "failed"
		tc.Error = "No models configured for any provider"
	} else {
		tc.Message = fmt.Sprintf("Found %d models across %d providers", totalModels, providerCount)
	}
	
	tc.Metadata = map[string]interface{}{
		"total_models":   totalModels,
		"provider_count": providerCount,
	}
	
	return tc
}

// Database test cases

func (ct *ConfigTester) testDatabaseConnection(ctx context.Context) *TestCase {
	tc := &TestCase{
		Description: "Test database connectivity",
		Category:    "database",
		Status:      "passed",
	}
	
	start := time.Now()
	defer func() {
		tc.Duration = time.Since(start)
	}()
	
	var db *sql.DB
	var err error
	
	// Build connection string based on database type
	switch ct.config.Database.Type {
	case "postgresql":
		connStr := ct.config.Database.ConnectionString
		if connStr == "" {
			// Build connection string from components
			connStr = fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
				ct.config.Database.Host,
				ct.config.Database.Port,
				ct.config.Database.Name,
				ct.config.Database.Username,
				ct.config.Database.Password,
				ct.config.Database.SSLMode,
			)
		}
		db, err = sql.Open("postgres", connStr)
		
	case "sqlite":
		db, err = sql.Open("sqlite3", ct.config.Database.SQLiteFile)
		
	default:
		tc.Status = "skipped"
		tc.Message = fmt.Sprintf("Unknown database type: %s", ct.config.Database.Type)
		return tc
	}
	
	if err != nil {
		tc.Status = "failed"
		tc.Error = fmt.Sprintf("Failed to open database: %v", err)
		return tc
	}
	defer db.Close()
	
	// Test connection
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	if err := db.PingContext(ctx); err != nil {
		tc.Status = "failed"
		tc.Error = fmt.Sprintf("Failed to ping database: %v", err)
		return tc
	}
	
	tc.Message = fmt.Sprintf("Successfully connected to %s database", ct.config.Database.Type)
	return tc
}

func (ct *ConfigTester) testDatabasePermissions(ctx context.Context) *TestCase {
	tc := &TestCase{
		Description: "Test database permissions",
		Category:    "database",
		Status:      "passed",
	}
	
	start := time.Now()
	defer func() {
		tc.Duration = time.Since(start)
	}()
	
	// This is a placeholder for database permission testing
	// In a real implementation, you would:
	// 1. Try to create a test table
	// 2. Insert test data
	// 3. Query the data
	// 4. Delete the test table
	
	tc.Status = "skipped"
	tc.Message = "Database permission testing not implemented yet"
	
	return tc
}

// Security test cases

func (ct *ConfigTester) testRateLimitStorage(ctx context.Context) *TestCase {
	tc := &TestCase{
		Description: "Test rate limiting storage backend",
		Category:    "security",
		Status:      "passed",
	}
	
	start := time.Now()
	defer func() {
		tc.Duration = time.Since(start)
	}()
	
	if !ct.config.Security.RateLimiting.Enabled {
		tc.Status = "skipped"
		tc.Message = "Rate limiting is disabled"
		return tc
	}
	
	storageType := ct.config.Security.RateLimiting.Storage.Type
	
	switch storageType {
	case "memory":
		tc.Message = "Using in-memory rate limit storage"
		
	case "redis":
		redisURL := ct.config.Security.RateLimiting.Storage.RedisURL
		if redisURL == "" {
			tc.Status = "failed"
			tc.Error = "Redis URL not configured"
			return tc
		}
		
		// Parse Redis URL
		u, err := url.Parse(redisURL)
		if err != nil {
			tc.Status = "failed"
			tc.Error = fmt.Sprintf("Invalid Redis URL: %v", err)
			return tc
		}
		
		// Try to connect to Redis
		conn, err := net.DialTimeout("tcp", u.Host, 2*time.Second)
		if err != nil {
			tc.Status = "failed"
			tc.Error = fmt.Sprintf("Cannot connect to Redis: %v", err)
			return tc
		}
		conn.Close()
		
		tc.Message = "Redis connection successful"
		
	default:
		tc.Status = "failed"
		tc.Error = fmt.Sprintf("Unknown storage type: %s", storageType)
	}
	
	return tc
}

func (ct *ConfigTester) testCORSConfig(ctx context.Context) *TestCase {
	tc := &TestCase{
		Description: "Test CORS configuration",
		Category:    "security",
		Status:      "passed",
	}
	
	start := time.Now()
	defer func() {
		tc.Duration = time.Since(start)
	}()
	
	if !ct.config.Security.CORS.Enabled {
		tc.Status = "skipped"
		tc.Message = "CORS is disabled"
		return tc
	}
	
	warnings := []string{}
	
	// Check allowed origins
	if len(ct.config.Security.CORS.AllowedOrigins) == 0 {
		warnings = append(warnings, "No allowed origins configured")
	} else {
		for _, origin := range ct.config.Security.CORS.AllowedOrigins {
			if origin == "*" {
				warnings = append(warnings, "Wildcard origin (*) allows requests from any domain")
			}
		}
	}
	
	// Check allowed methods
	if len(ct.config.Security.CORS.AllowedMethods) == 0 {
		warnings = append(warnings, "No allowed methods configured")
	}
	
	if len(warnings) > 0 {
		tc.Status = "passed" // Still pass but with warnings
		tc.Message = strings.Join(warnings, "; ")
	} else {
		tc.Message = "CORS configuration is valid"
	}
	
	tc.Metadata = map[string]interface{}{
		"allowed_origins": ct.config.Security.CORS.AllowedOrigins,
		"allowed_methods": ct.config.Security.CORS.AllowedMethods,
	}
	
	return tc
}

// External services test cases

func (ct *ConfigTester) testMetricsStorage(ctx context.Context) *TestCase {
	tc := &TestCase{
		Description: "Test metrics storage configuration",
		Category:    "metrics",
		Status:      "passed",
	}
	
	start := time.Now()
	defer func() {
		tc.Duration = time.Since(start)
	}()
	
	if !ct.config.Metrics.Enabled {
		tc.Status = "skipped"
		tc.Message = "Metrics collection is disabled"
		return tc
	}
	
	storageType := ct.config.Metrics.Storage.Type
	
	switch storageType {
	case "memory":
		maxMemory := ct.config.Metrics.Storage.MaxMemoryMB
		if maxMemory < 10 {
			tc.Status = "failed"
			tc.Error = "Memory limit too low for metrics storage"
		} else {
			tc.Message = fmt.Sprintf("Using in-memory storage with %dMB limit", maxMemory)
		}
		
	default:
		tc.Status = "failed"
		tc.Error = fmt.Sprintf("Unknown metrics storage type: %s", storageType)
	}
	
	return tc
}

func (ct *ConfigTester) testSupabaseConnection(ctx context.Context) *TestCase {
	tc := &TestCase{
		Description: "Test Supabase connection",
		Category:    "metrics",
		Status:      "passed",
	}
	
	start := time.Now()
	defer func() {
		tc.Duration = time.Since(start)
	}()
	
	if ct.config.Metrics.Supabase.URL == "" || ct.config.Metrics.Supabase.Key == "" {
		tc.Status = "skipped"
		tc.Message = "Supabase not configured"
		return tc
	}
	
	// Validate Supabase URL
	u, err := url.Parse(ct.config.Metrics.Supabase.URL)
	if err != nil {
		tc.Status = "failed"
		tc.Error = fmt.Sprintf("Invalid Supabase URL: %v", err)
		return tc
	}
	
	if !strings.HasSuffix(u.Host, "supabase.co") && !strings.HasSuffix(u.Host, "supabase.io") {
		tc.Status = "failed"
		tc.Error = "URL does not appear to be a valid Supabase endpoint"
		return tc
	}
	
	tc.Message = "Supabase configuration appears valid"
	tc.Metadata = map[string]interface{}{
		"supabase_host": u.Host,
		"table_name":    ct.config.Metrics.Supabase.Table,
	}
	
	return tc
}

// Helper methods

func (ct *ConfigTester) generateTestSummary(result *TestResult) string {
	if result.Success {
		return fmt.Sprintf("All tests passed (%d passed, %d skipped)",
			result.PassedCount, result.SkippedCount)
	}
	
	return fmt.Sprintf("Some tests failed (%d passed, %d failed, %d skipped)",
		result.PassedCount, result.FailedCount, result.SkippedCount)
}

// SetHTTPClient sets a custom HTTP client for testing
func (ct *ConfigTester) SetHTTPClient(client *http.Client) {
	ct.httpClient = client
}

// SetContext sets the context for test execution
func (ct *ConfigTester) SetContext(ctx context.Context) {
	ct.context = ctx
}