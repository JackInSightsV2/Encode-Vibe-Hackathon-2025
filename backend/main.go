package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"qt1-middleware/config"
	"qt1-middleware/middleware"
	"qt1-middleware/api"
)

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
	
	// Initialize middleware
	proxy := middleware.NewProxy()
	
	// Set proxy instance for config reloading
	api.ProxyInstance = proxy
	
	// Set up routes
	http.HandleFunc("/chat", proxy.HandleChat)
	http.HandleFunc("/test-openai", proxy.TestOpenAIHandler)
	http.HandleFunc("/health", handleHealth)
	
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
	
	// Admin API routes with CORS
	http.HandleFunc("/api/config", corsHandler(api.HandleConfig))
	http.HandleFunc("/api/logs", corsHandler(api.HandleLogs))
	http.HandleFunc("/api/status", corsHandler(api.HandleStatus))
	http.HandleFunc("/api/killswitch", corsHandler(api.HandleKillSwitch))
	
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
	
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy","service":"qt1-middleware"}`))
}