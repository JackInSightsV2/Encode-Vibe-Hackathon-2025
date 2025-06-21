package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"qt1-middleware/auth"
	"qt1-middleware/config"
	"qt1-middleware/database"
	"qt1-middleware/metrics"
	"qt1-middleware/middleware"
	"qt1-middleware/api"
)

var db *database.Database
var authManager *auth.AuthManager

func main() {
	// Load configuration
	configPath := "config.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}
	
	if err := config.LoadConfig(configPath); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	
	// Create logs directory
	if err := os.MkdirAll("logs", 0755); err != nil {
		log.Printf("Warning: failed to create logs directory: %v", err)
	}
	
	// Initialize metrics collector
	var metricsCollector *metrics.MetricsCollector
	var metricsConfig metrics.MetricsConfig
	if config.AppConfig.Metrics.Enabled {
		metricsConfig = metrics.MetricsConfig{
			Enabled:    config.AppConfig.Metrics.Enabled,
			Storage:    convertStorageConfig(config.AppConfig.Metrics.Storage),
			Supabase:   convertSupabaseConfig(config.AppConfig.Metrics.Supabase),
			Collection: convertCollectionConfig(config.AppConfig.Metrics.Collection),
		}
		
		var err error
		metricsCollector, err = metrics.NewMetricsCollector(metricsConfig)
		if err != nil {
			log.Fatalf("Failed to create metrics collector: %v", err)
		}
		
		// TODO: Implement real API handlers when needed
		
		ctx := context.Background()
		if err := metricsCollector.Start(ctx); err != nil {
			log.Fatalf("Failed to start metrics collector: %v", err)
		}
		
		log.Printf("Metrics collection enabled (storage: %s)", metricsConfig.Storage.Type)
	}
	
	// Set global metrics collector instance for handlers
	if metricsCollector != nil {
		api.MetricsCollectorInstance = metricsCollector
	}
	
	// Initialize database
	dbConfig := convertDatabaseConfig(config.AppConfig)
	if dbConfig.Type != "" {
		var err error
		db, err = database.NewDatabase(dbConfig)
		if err != nil {
			log.Fatalf("Failed to create database: %v", err)
		}
		
		if err := db.Connect(); err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
		
		log.Printf("Database connected successfully (type: %s)", dbConfig.Type)
		if dbConfig.Type == "sqlite" {
			log.Printf("SQLite file: %s", dbConfig.SQLiteFile)
		}
		
		// Set database instance for API handlers
		api.SetDatabase(db)

		// Initialize authentication manager
		authManager, err = auth.NewAuthManager(db)
		if err != nil {
			log.Fatalf("Failed to create authentication manager: %v", err)
		}
		log.Printf("Authentication system initialized")
	}
	
	// Initialize middleware
	proxy := middleware.NewProxy()
	
	// Set proxy instance for moderation engine access
	api.SetProxy(proxy)
	
	// Initialize rate limiting middleware
	var rateLimitMiddleware *middleware.RateLimitMiddleware
	if config.AppConfig.Security.RateLimiting.Enabled {
		rateLimitMiddleware = middleware.NewRateLimitMiddleware(&config.AppConfig.Security.RateLimiting)
		if rateLimitMiddleware != nil {
			log.Printf("Rate limiting enabled (global: %d req/s, burst: %d)", 
				config.AppConfig.Security.RateLimiting.Global.RequestsPerSecond,
				config.AppConfig.Security.RateLimiting.Global.Burst)
			
			// Set the middleware instance for API handlers
			api.SetRateLimitMiddleware(rateLimitMiddleware)
		}
	}
	
	// Initialize security middleware
	var securityMiddleware *middleware.SecurityMiddleware
	if config.AppConfig.Security.Headers.Enabled || config.AppConfig.Security.InputValidation.Enabled {
		securityMiddleware = middleware.NewSecurityMiddleware(config.AppConfig)
		log.Printf("Security middleware enabled (headers: %v, input validation: %v)", 
			config.AppConfig.Security.Headers.Enabled,
			config.AppConfig.Security.InputValidation.Enabled)
	}
	
	// Initialize security monitor for both IP and DDoS protection
	securityMonitor := middleware.NewSecurityMonitor(config.AppConfig)
	
	// Initialize IP protection middleware
	var ipProtectionMiddleware *middleware.IPProtectionMiddleware
	if config.AppConfig.Security.IPProtection.Enabled {
		ipProtectionMiddleware = middleware.NewIPProtectionMiddleware(&config.AppConfig.Security.IPProtection, securityMonitor)
		log.Printf("IP protection enabled (geoblocking: %v, reputation check: %v)", 
			config.AppConfig.Security.IPProtection.EnableGeoblocking,
			config.AppConfig.Security.IPProtection.EnableReputationCheck)
	}
	
	// Initialize DDoS protection middleware
	var ddosProtectionMiddleware *middleware.DDoSProtectionMiddleware
	if config.AppConfig.Security.DDoSProtection.Enabled {
		log.Printf("Initializing DDoS protection...")
		ddosProtectionMiddleware = middleware.NewDDoSProtectionMiddleware(&config.AppConfig.Security.DDoSProtection, securityMonitor)
		log.Printf("DDoS protection enabled (spike threshold: %d, circuit breaker: %d, throttling: %v)",
			config.AppConfig.Security.DDoSProtection.SpikeThreshold,
			config.AppConfig.Security.DDoSProtection.CircuitBreakerThreshold,
			config.AppConfig.Security.DDoSProtection.EnableThrottling)
		
		// Set the DDoS middleware instance for API handlers
		api.SetDDoSMiddleware(ddosProtectionMiddleware)
	}
	
	// Initialize WebSocket manager
	wsManager := api.NewWebSocketManager()
	api.WSManager = wsManager  // Set the global WebSocket manager
	log.Printf("WebSocket manager initialized")

	// Set proxy instance for config reloading
	// TODO: Implement real API handlers when needed
	
	// Set up routes (chat route will be configured later with metrics middleware)
	http.HandleFunc("/test-openai", proxy.TestOpenAIHandler)
	http.HandleFunc("/health", handleHealth)
	// WebSocket endpoints
	http.HandleFunc("/ws", wsManager.HandleWebSocket)
	log.Printf("WebSocket endpoint registered at /ws")
	
	// CORS middleware wrapper
	corsHandler := func(handler http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			
			handler(w, r)
		}
	}
	
	// Admin API routes - no authentication required
	log.Printf("Admin API routes available without authentication")
	
	// Moderation routes - temporarily without authentication for debugging
	http.HandleFunc("/api/moderation/stats", corsHandler(api.HandleModerationStats))
	http.HandleFunc("/api/moderation/pii-analytics", corsHandler(api.HandlePIIAnalytics))
	http.HandleFunc("/api/moderation/rules/stats", corsHandler(api.HandleModerationRuleStats))
	http.HandleFunc("/api/moderation/layers", corsHandler(api.HandleModerationLayers))
	http.HandleFunc("/api/moderation/test", corsHandler(api.HandleModerationTest))
	
	// System status endpoint
	http.HandleFunc("/api/status", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		status := map[string]interface{}{
			"status": "healthy",
			"version": "1.0.0",
			"uptime": time.Since(time.Now().Add(-time.Hour)).String(), // Placeholder
			"services": map[string]interface{}{
				"database": "connected",
				"websocket": "active",
				"moderation": "enabled",
				"metrics": "enabled",
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data": status,
		})
	}))
	
	log.Printf("Moderation API enabled without authentication (debug mode)")
	
	// Metrics routes - all unprotected, no authentication required
	if metricsCollector != nil {
		metricsHandler := metrics.NewAPIHandler(metricsCollector)
		metricsHandler.RegisterRoutes(http.NewServeMux())
		
		// All metrics routes are unprotected
		http.HandleFunc("/api/metrics", corsHandler(metricsHandler.HandleMetrics))
		http.HandleFunc("/api/metrics/summary", corsHandler(metricsHandler.HandleSummary))
		http.HandleFunc("/api/metrics/health", corsHandler(metricsHandler.HandleHealth))
		http.HandleFunc("/api/metrics/stats", corsHandler(metricsHandler.HandleStats))
		http.HandleFunc("/api/metrics/system/snapshot", corsHandler(metricsHandler.HandleSystemSnapshot))
		http.HandleFunc("/api/metrics/providers", corsHandler(metricsHandler.HandleProviderHealth))
		http.HandleFunc("/api/metrics/timeseries", corsHandler(metricsHandler.HandleTimeSeries))
		
		// Add metrics middleware to proxy routes
		http.HandleFunc("/chat", metricsHandler.RecordHTTPMetrics(http.HandlerFunc(proxy.HandleChat)).ServeHTTP)
		
		log.Printf("Metrics API endpoints registered")
	} else {
		// Register chat route without metrics
		http.HandleFunc("/chat", proxy.HandleChat)
	}
	
	// Register authentication routes
	if authManager != nil {
		// Create a new ServeMux for auth routes
		authMux := http.NewServeMux()
		authManager.RegisterRoutes(authMux)
		
		// Register auth routes with the default mux
		http.Handle("/api/auth/", corsHandler(func(w http.ResponseWriter, r *http.Request) {
			authMux.ServeHTTP(w, r)
		}))
		http.Handle("/api/oauth2/", corsHandler(func(w http.ResponseWriter, r *http.Request) {
			authMux.ServeHTTP(w, r)
		}))
		
		log.Printf("Authentication endpoints registered")
	} else {
		log.Printf("Authentication disabled - no database connection")
	}
	
	// Database routes
	if db != nil {
		// TODO: Implement database API when needed
		log.Printf("Database API disabled - implement when needed")
	}

	// All endpoints are now unprotected - no authentication required
	http.HandleFunc("/api/users", api.HandleGetUsers)
	http.HandleFunc("/api/users/create", api.HandleCreateUser)
	
	// IP Protection endpoints
	http.HandleFunc("/api/ip-protection/status", api.HandleIPProtectionStatus)
	http.HandleFunc("/api/ip-protection/config", api.HandleIPProtectionConfig)
	http.HandleFunc("/api/ip-protection/geographic-stats", api.HandleIPProtectionGeoStats)
	
	// DDoS Protection endpoints
	http.HandleFunc("/api/ddos-protection/status", api.HandleDDoSProtectionStatus)
	http.HandleFunc("/api/ddos-protection/config", api.HandleDDoSProtectionConfig)
	http.HandleFunc("/api/ddos-protection/metrics", api.HandleDDoSProtectionMetrics)
	http.HandleFunc("/api/ddos-protection/threat-analysis", api.HandleDDoSProtectionThreatAnalysis)
	http.HandleFunc("/api/ddos-protection/emergency-mode", api.HandleDDoSProtectionEmergencyMode)
	http.HandleFunc("/api/ddos-protection/circuit-breaker/reset", api.HandleDDoSProtectionCircuitBreakerReset)
	
	// Rate Limiting endpoints
	http.HandleFunc("/api/rate-limits", api.HandleRateLimitStatus)
	http.HandleFunc("/api/rate-limits/status", api.HandleRateLimitStatus)
	http.HandleFunc("/api/rate-limits/config", api.HandleRateLimitConfig)
	http.HandleFunc("/api/rate-limits/block-ip", api.HandleRateLimitBlockIP)
	http.HandleFunc("/api/rate-limits/unblock-ip", api.HandleRateLimitUnblockIP)
	
	// Rules Management endpoints
	http.HandleFunc("/api/rules", api.HandleGetRules)
	http.HandleFunc("/api/rules/create", api.HandleCreateRule)
	
	// Optimizer endpoints
	http.HandleFunc("/api/optimizer/metrics", api.HandleOptimizerStatus)
	http.HandleFunc("/api/optimizer/status", api.HandleOptimizerStatus)
	http.HandleFunc("/api/optimizer/experiments", api.HandleOptimizerExperiments)
	http.HandleFunc("/api/optimizer/drift/alerts", api.HandleOptimizerDriftAlerts)
	http.HandleFunc("/api/optimizer/rollouts", api.HandleOptimizerRollouts)
	http.HandleFunc("/api/optimizer/metrics/historical", api.HandleOptimizerHistoricalMetrics)
	
	// Provider endpoints
	http.HandleFunc("/api/providers", api.HandleGetProviders)
	
	// Configuration endpoints
	http.HandleFunc("/api/config/status", api.HandleConfigurationStatus)
	http.HandleFunc("/api/config/moderation", corsHandler(api.HandleModerationConfig))
	http.HandleFunc("/api/config/rules", corsHandler(api.HandleRulesConfig))
	http.HandleFunc("/api/config/pii", corsHandler(api.HandlePIIConfig))
	
	// Relevancy endpoints
	http.HandleFunc("/api/relevancy/config", corsHandler(api.HandleRelevancyConfig))
	http.HandleFunc("/api/relevancy/keywords", corsHandler(api.HandleRelevancyKeywords))
	http.HandleFunc("/api/relevancy/test", corsHandler(api.HandleRelevancyTest))
	http.HandleFunc("/api/relevancy/ai-provider", corsHandler(api.HandleRelevancyAIProvider))
	http.HandleFunc("/api/relevancy/ai-test", corsHandler(api.HandleRelevancyAITest))
	
	// Kill Switch endpoints
	http.HandleFunc("/api/killswitch/status", api.HandleKillSwitchStatus)
	http.HandleFunc("/api/killswitch", api.HandleKillSwitch)
	
	// Logs endpoints
	http.HandleFunc("/api/logs", api.HandleLogs)
	
	log.Printf("Additional API endpoints registered")
	
	// TODO: Implement user conversion when API package is restored
	
	// Serve React frontend static files
	distPath := "../frontend/dist"
	if _, err := os.Stat(distPath); err == nil {
		// Create a file server for the dist directory
		fs := http.Dir(distPath)
		fileServer := http.FileServer(fs)
		
		// Handle assets directory with proper MIME types
		http.Handle("/assets/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set proper MIME types based on file extension
			ext := filepath.Ext(r.URL.Path)
			switch ext {
			case ".css":
				w.Header().Set("Content-Type", "text/css; charset=utf-8")
			case ".js":
				w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
			case ".json":
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
			case ".ico":
				w.Header().Set("Content-Type", "image/x-icon")
			case ".svg":
				w.Header().Set("Content-Type", "image/svg+xml")
			}
			
			// Serve the file
			fileServer.ServeHTTP(w, r)
		}))
		
		// Handle root and SPA routing
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// Don't serve index.html for API routes
			if r.URL.Path == "/chat" || r.URL.Path == "/health" {
				http.NotFound(w, r)
				return
			}
			
			// For the root path or any non-asset path, serve index.html
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			indexPath := filepath.Join(distPath, "index.html")
			http.ServeFile(w, r, indexPath)
		})
		
		log.Printf("Serving React frontend from: %s", distPath)
	} else {
		log.Printf("Frontend dist not found at %s, serving API only", distPath)
		// Serve a simple landing page for API-only mode
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/" {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>QT-1 Middleware</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background: #f5f5f5; }
        .container { max-width: 800px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .status { color: #22c55e; font-weight: bold; }
        .endpoint { background: #f8f9fa; padding: 15px; margin: 10px 0; border-radius: 4px; border-left: 4px solid #3b82f6; }
        .method { display: inline-block; padding: 2px 8px; border-radius: 3px; color: white; font-size: 12px; font-weight: bold; margin-right: 10px; }
        .get { background: #22c55e; }
        .post { background: #f59e0b; }
        .put { background: #8b5cf6; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🤖 QT-1 Responsible AI Middleware</h1>
        <p class="status">✅ System Online</p>
        <p>High-performance AI proxy with content moderation, relevance filtering, and kill switches.</p>
        
        <h2>Available Endpoints</h2>
        <div class="endpoint">
            <span class="method post">POST</span>
            <strong>/chat</strong> - Main proxy endpoint for AI interactions
        </div>
        <div class="endpoint">
            <span class="method get">GET</span>
            <strong>/health</strong> - Health check
        </div>
        <div class="endpoint">
            <span class="method get">GET</span>
            <strong>/api/status</strong> - System status and configuration
        </div>
        <div class="endpoint">
            <span class="method get">GET</span>
            <span class="method put">PUT</span>
            <strong>/api/config</strong> - Configuration management
        </div>
        <div class="endpoint">
            <span class="method get">GET</span>
            <strong>/api/logs</strong> - Access request logs
        </div>
        <div class="endpoint">
            <span class="method get">GET</span>
            <span class="method post">POST</span>
            <strong>/api/killswitch</strong> - Manage blocked users/sessions
        </div>
        
        <h2>Quick Test</h2>
        <p>Test the chat endpoint:</p>
        <pre style="background: #1f2937; color: #f3f4f6; padding: 15px; border-radius: 4px; overflow-x: auto;">curl -X POST http://localhost:` + fmt.Sprintf("%d", config.AppConfig.Server.Port) + `/chat \
  -H "Content-Type: application/json" \
  -d '{"user_id":"test","session_id":"test","message":"Hello world"}'</pre>
        
        <p><em>Build the React frontend with <code>npm run build</code> in the frontend directory to enable the full admin interface.</em></p>
    </div>
</body>
</html>`))
		})
	}
	
	addr := fmt.Sprintf("%s:%d", config.AppConfig.Server.Host, config.AppConfig.Server.Port)
	log.Printf("QT-1 middleware starting on %s", addr)
	log.Printf("*** DEBUG VERSION WITH VERBOSE LOGGING ***")
	log.Printf("Target URL: %s", config.AppConfig.Server.TargetURL)
	log.Printf("Moderation enabled: %v", config.AppConfig.Moderation.Enabled)
	log.Printf("Relevance filtering enabled: %v", config.AppConfig.Relevance.Enabled)
	
	// Create HTTP server with security middleware chain
	var handler http.Handler = http.DefaultServeMux
	
	// Apply middlewares in reverse order (innermost first)
	if rateLimitMiddleware != nil {
		handler = rateLimitMiddleware.Handler(handler)
	}
	if securityMiddleware != nil {
		handler = securityMiddleware.Handler(handler)
	}
	if ipProtectionMiddleware != nil {
		handler = ipProtectionMiddleware.Handler(handler)
	}
	if ddosProtectionMiddleware != nil {
		handler = ddosProtectionMiddleware.Handler(handler)
	}
	
	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	
	// Start server in a goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()
	
	// Setup graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	
	// Block until signal received
	<-c
	log.Println("Shutting down gracefully...")
	
	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// Shutdown middleware components
	if rateLimitMiddleware != nil {
		log.Println("Stopping rate limiting middleware...")
		rateLimitMiddleware.Stop()
	}
	if securityMiddleware != nil {
		log.Println("Stopping security middleware...")
		securityMiddleware.Stop()
	}
	if ipProtectionMiddleware != nil {
		log.Println("IP protection middleware stopped")
		// No explicit stop method needed for IP protection middleware
	}
	if ddosProtectionMiddleware != nil {
		log.Println("Stopping DDoS protection middleware...")
		ddosProtectionMiddleware.Stop()
	}
	
	// Shutdown metrics collector
	if metricsCollector != nil {
		log.Println("Stopping metrics collector...")
		if err := metricsCollector.Stop(); err != nil {
			log.Printf("Error stopping metrics collector: %v", err)
		}
	}
	
	// Shutdown database
	if db != nil {
		log.Println("Closing database connection...")
		if err := db.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}
	
	// Shutdown HTTP server
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}
	
	log.Println("Server exited")
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	health := map[string]interface{}{
		"status":  "healthy",
		"service": "qt1-middleware",
	}
	
	// Add database health if database is initialized
	if db != nil {
		dbHealth := db.HealthCheck()
		health["database"] = dbHealth
		
		// If database is unhealthy, mark overall status as degraded
		if dbHealth.Status == "unhealthy" {
			health["status"] = "degraded"
		}
	}
	
	// Determine HTTP status code
	statusCode := http.StatusOK
	if health["status"] == "degraded" {
		statusCode = http.StatusServiceUnavailable
	}
	
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(health)
}

// Converter functions for metrics configuration
func convertStorageConfig(c config.StorageConfig) metrics.StorageConfig {
	return metrics.StorageConfig{
		Type:                   c.Type,
		RetentionHours:         c.RetentionHours,
		CleanupIntervalMinutes: c.CleanupIntervalMinutes,
		MaxMemoryMB:           c.MaxMemoryMB,
	}
}

func convertSupabaseConfig(c config.SupabaseConfig) metrics.SupabaseConfig {
	return metrics.SupabaseConfig{
		URL:   c.URL,
		Key:   c.Key,
		Table: c.Table,
	}
}

func convertCollectionConfig(c config.CollectionConfig) metrics.CollectionConfig {
	return metrics.CollectionConfig{
		HTTPRequests:     c.HTTPRequests,
		ModerationEvents: c.ModerationEvents,
		SystemHealth:     c.SystemHealth,
		PIIDetection:     c.PIIDetection,
	}
}

func convertDatabaseConfig(c *config.Config) *database.DatabaseConfig {
	return &database.DatabaseConfig{
		Type:                c.Database.Type,
		ConnectionString:    c.Database.ConnectionString,
		Host:               c.Database.Host,
		Port:               c.Database.Port,
		Name:               c.Database.Name,
		Username:           c.Database.Username,
		Password:           c.Database.Password,
		SSLMode:            c.Database.SSLMode,
		MaxConnections:     c.Database.MaxConnections,
		MaxIdleConnections: c.Database.MaxIdleConnections,
		ConnectionLifetime: c.Database.ConnectionLifetime,
		SQLiteFile:         c.Database.SQLiteFile,
		EnableWAL:          c.Database.EnableWAL,
		EnableForeignKeys:  c.Database.EnableForeignKeys,
		Migrations: database.MigrationConfig{
			Enabled:   c.Database.Migrations.Enabled,
			Directory: c.Database.Migrations.Directory,
			Table:     c.Database.Migrations.Table,
		},
		Retention: database.RetentionConfig{
			Requests: c.Database.Retention.Requests,
			Metrics:  c.Database.Retention.Metrics,
			Logs:     c.Database.Retention.Logs,
		},
	}
}