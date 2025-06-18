# QT-1 - Responsible AI Middleware

A high-performance AI middleware system built with Go that acts as a configurable proxy for AI agents, featuring content moderation, relevance filtering, kill switches, and comprehensive logging.

## 🚀 Quick Start

### Prerequisites
- Go 1.19+ 
- Node.js 18+ (for frontend)
- jq (for testing scripts)

### Backend Setup

1. **Build the middleware:**
   ```bash
   cd backend
   go build -buildvcs=false -o qt1-middleware .
   ```

2. **Configure the system:**
   Edit `backend/config.yaml` to set your target URL and moderation settings.

3. **Run the middleware:**
   ```bash
   cd backend
   ./qt1-middleware
   ```

The middleware will start on `localhost:8080` by default.

### Frontend Setup

Due to permission issues with the mounted drive, the React frontend requires manual dependency installation. The frontend code is ready in the `frontend/` directory with:

- TypeScript + React setup
- Tailwind CSS configuration  
- Vite build system
- Admin dashboard components

To run the frontend in a different environment:
```bash
cd frontend
npm install
npm run dev
```

## 🧪 Testing

Run the included test script to verify the middleware:

```bash
chmod +x test_middleware.sh
./test_middleware.sh
```

This will test:
- Health and status endpoints
- Configuration API
- Chat endpoint with moderation
- Kill switch functionality

## 🏗️ Architecture

### Backend Components

- **`main.go`** - HTTP server and routing
- **`config/`** - YAML/environment configuration system
- **`middleware/`** - Proxy logic with moderation and filtering
- **`api/`** - Admin API endpoints
- **`utils/`** - Logging and utilities

### Key Features

1. **HTTP Proxy** - Routes requests to target AI endpoints
2. **Content Moderation** - Regex-based filtering + OpenAI integration ready
3. **Relevance Filtering** - Embedding-based content relevance (OpenAI ready)
4. **Kill Switch** - Block users/sessions in real-time
5. **Comprehensive Logging** - File-based logging with JSON format
6. **Admin API** - RESTful endpoints for configuration and monitoring

### Endpoints

- `POST /chat` - Main proxy endpoint for AI interactions
- `GET /health` - Health check
- `GET /api/status` - System status and configuration
- `GET|PUT /api/config` - Configuration management
- `GET /api/logs` - Access logs
- `GET|POST /api/killswitch` - Manage blocked users/sessions

## ⚙️ Configuration

The system uses a layered configuration approach:

1. **Defaults** - Built-in sensible defaults
2. **YAML file** - `config.yaml` for main configuration  
3. **Environment variables** - Override any setting

### Key Environment Variables

- `QT1_PORT` - Server port (default: 8080)
- `QT1_HOST` - Server host (default: localhost)  
- `QT1_TARGET_URL` - Target AI endpoint URL
- `OPENAI_API_KEY` - For moderation and embeddings
- `QT1_LOG_FILE` - Log file path

## 🛡️ QT-1 Framework

This implementation follows the QT-1 Responsible AI Framework with seven core laws:

1. **Do No Harm** - Content moderation prevents harmful outputs
2. **Respect Human Autonomy** - Users maintain control
3. **Be Transparent** - All actions are logged and auditable  
4. **Accept Oversight** - Admin controls and monitoring
5. **Stay Within Scope** - Relevance filtering keeps AI on-topic
6. **Protect Privacy** - Secure logging and data handling
7. **Ensure Accountability** - Comprehensive audit trails

## 📊 Current Status

✅ **Completed:**
- Go backend with HTTP proxy
- Configuration system (YAML + env)
- Content moderation (regex-based)
- Kill switch system
- Logging infrastructure
- Admin API endpoints
- React frontend structure (needs deps install)

🚧 **In Progress:**
- OpenAI API integration for moderation
- Embedding-based relevance filtering
- SQLite logging option
- Frontend dependency installation

## 🔄 Next Steps

1. **Deploy** - Docker containerization  
2. **Enhance** - OpenAI API integration
3. **Scale** - Database backend and clustering
4. **Monitor** - Metrics and alerting
5. **Framework** - Community plugins and templates

## 📝 Logs

All requests are logged to `logs/qt1.log` in JSON format:

```json
{
  "timestamp": "2025-06-18T14:15:30Z",
  "user_id": "user123", 
  "session_id": "session456",
  "message": "Hello world",
  "action": "forwarded",
  "level": "info"
}
```

## 🤝 Contributing

This is a hackathon project following the QT-1 Responsible AI framework. The goal is to create runtime-enforceable ethical AI systems.

---

**Built for responsible AI deployment** 🤖⚡
