# QT-1 - Enterprise AI Middleware & Governance Platform

A comprehensive, production-ready AI middleware system built with Go that provides intelligent routing, advanced security, content moderation, optimization, and real-time monitoring for AI applications. Features enterprise-grade authentication, multi-layer protection, and responsible AI governance.

## 🚀 Quick Start

### Prerequisites
- Go 1.19+ 
- Node.js 18+ (for frontend)
- SQLite (included)
- PowerShell (Windows) or Bash (Linux/macOS)

### Backend Setup

1. **Navigate to backend directory:**
   ```powershell
   cd backend
   ```

2. **Build the middleware:**
   ```powershell
   go build -buildvcs=false -o qt1-middleware.exe .
   ```

3. **Configure the system:**
   Edit `backend/config.yaml` to configure providers, security settings, and moderation rules.

4. **Run the middleware:**
   ```powershell
   .\qt1-middleware.exe
   ```

The middleware will start on `localhost:8080` by default with full functionality enabled.

### Frontend Setup

```bash
cd frontend
npm install
npm run dev
```

## 🏗️ Architecture Overview

### Core Components

- **Authentication System** - JWT, OAuth2, RBAC, session management
- **Security Middleware** - Multi-tier rate limiting, DDoS protection, IP filtering
- **AI Provider Management** - Intelligent routing, health monitoring, circuit breakers
- **Content Moderation** - Multi-layer pipeline with AI and rule-based filtering
- **Metrics & Analytics** - Real-time collection, time-series storage, dashboards
- **Rule Optimization** - Genetic algorithms, A/B testing, performance tuning
- **WebSocket System** - Real-time communication, live monitoring
- **Configuration Management** - Hot-reloading, validation, version control

## 🛡️ Security & Protection Features

### **Authentication & Authorization**
- ✅ **JWT-based authentication** with secure token generation and validation
- ✅ **Role-based access control (RBAC)** - Admin, Operator, User roles with granular permissions
- ✅ **OAuth2 integration** - GitHub, Google, Microsoft provider support
- ✅ **API key authentication** - Service-to-service secure communication
- ✅ **Session management** - Secure session handling with timeout and cleanup
- ✅ **User management** - Registration, login, logout, profile management
- ✅ **WebSocket authentication** - Secure real-time connections
- ✅ **Password policies** - Secure hashing, complexity requirements, change tracking
- ✅ **Device tracking** - Anomaly detection for suspicious login patterns
- ✅ **Multi-factor authentication** - Enhanced security for admin accounts

### **Advanced Protection Systems**
- ✅ **Multi-tier rate limiting** - Token bucket and sliding window algorithms
  - Global, per-IP, per-user, and per-endpoint rate limits
  - Automatic scaling and burst handling
  - Memory-efficient cleanup and optimization
- ✅ **DDoS protection** - Spike detection, circuit breakers, adaptive throttling
  - Real-time traffic analysis and anomaly detection
  - Automatic blocking with configurable thresholds
  - Geographic and behavioral pattern analysis
- ✅ **IP protection** - Geoblocking, reputation scoring, automatic threat response
  - Country/region-based blocking
  - VPN/Proxy/Tor detection and filtering
  - Threat intelligence integration
  - Real-time reputation updates
- ✅ **Kill switch system** - Emergency blocking for users, sessions, or entire system
- ✅ **Security headers** - CORS, CSP, HSTS, and other security policies
- ✅ **Input validation** - Comprehensive request sanitization and validation

## 🤖 AI Content Moderation Engine

### **Multi-Layer Moderation Pipeline**
- ✅ **OpenAI moderation integration** - Commercial-grade content filtering
- ✅ **PII detection and masking** - Automated personally identifiable information protection
  - Email addresses, phone numbers, SSNs, credit cards
  - Configurable masking patterns and sensitivity levels
  - Real-time detection with minimal latency impact
- ✅ **Regex-based filtering** - Custom pattern matching with weighted scoring
- ✅ **Rule-based scoring** - Configurable thresholds and actions
- ✅ **Caching system** - High-performance result caching for repeated content
- ✅ **Real-time analytics** - Detailed moderation statistics and reporting
- ✅ **Hot configuration reloading** - Dynamic rule updates without restart
- ✅ **Rule engine** - Advanced rule parsing and execution system
- ✅ **Context awareness** - User and session-based moderation decisions

### **Moderation Features**
- Weighted layer aggregation with custom scoring algorithms
- Configurable actions: Log, Flag, Block, or Custom responses
- Severity classification: Low, Medium, High, Critical
- Context-aware processing with user and session tracking
- Performance monitoring with sub-millisecond latency tracking
- Comprehensive audit trails for compliance and analysis

## 📊 Metrics & Analytics System

### **Real-time Data Collection**
- ✅ **HTTP request metrics** - Latency, status codes, error rates, user agents
- ✅ **Moderation event tracking** - Scores, decisions, layer performance, trends
- ✅ **PII detection analytics** - Detection rates, types, masking effectiveness
- ✅ **System health monitoring** - CPU, memory, connections, performance
- ✅ **Custom metric recording** - Flexible tagging and metadata support
- ✅ **Time-series storage** - Efficient querying and data retention
- ✅ **Supabase integration** - Cloud-based metrics storage and analytics
- ✅ **Metrics aggregation** - Statistical rollups and trend analysis
- ✅ **Performance benchmarking** - SLA monitoring and alerting

### **Analytics Features**
- Real-time dashboards with live metric updates
- Historical trend analysis with configurable time ranges
- Performance benchmarking and SLA monitoring
- Anomaly detection and alerting
- Data export capabilities for external analysis
- Compliance reporting and audit trail generation

## 🔄 AI Provider Management

### **Intelligent Provider System**
- ✅ **Multi-provider support** - OpenAI, Anthropic, Local, and custom providers
- ✅ **Health monitoring** - Continuous health checks with automatic failover
- ✅ **Circuit breaker patterns** - Fault tolerance with graceful degradation
- ✅ **Intelligent routing** - Health-based load balancing and optimization
- ✅ **Configuration management** - Hot-reloading provider settings
- ✅ **Request/response transformation** - Provider-specific adapters
- ✅ **Performance optimization** - Provider-specific rate limiting and caching
- ✅ **Factory pattern** - Dynamic provider creation and registration
- ✅ **Health scheduling** - Automated health check scheduling

### **Provider Features**
- Automatic provider discovery and registration
- Real-time health status monitoring
- Latency-based routing decisions
- Cost optimization with usage tracking
- A/B testing capabilities for provider comparison
- Comprehensive provider analytics and reporting

## ⚡ AI Rule Optimization Engine

### **Advanced Optimization System**
- ✅ **Genetic algorithm optimization** - Evolutionary rule improvement
- ✅ **A/B testing framework** - Statistical validation of rule changes
- ✅ **Opik integration** - Experiment tracking and ML observability
- ✅ **Python engine integration** - Advanced algorithms and machine learning
- ✅ **Performance benchmarking** - Statistical analysis and validation
- ✅ **Traffic splitting** - Controlled experiment rollouts
- ✅ **Rollback mechanisms** - Safe deployment with automatic rollback
- ✅ **Drift detection** - Performance degradation monitoring
- ✅ **Experiment management** - Complete experiment lifecycle management

### **Optimization Features**
- Multi-objective optimization with configurable fitness functions
- Population-based search with elitism and diversity preservation
- Real-time experiment monitoring and early stopping
- Statistical significance testing and confidence intervals
- Automated rule promotion based on performance metrics
- Comprehensive experiment history and analysis

## 🌐 Real-time Communication

### **WebSocket Management**
- ✅ **Connection pooling** - Efficient connection management and scaling
- ✅ **Message routing** - Type-based handlers with permission validation
- ✅ **Real-time dashboards** - Live system monitoring and status updates
- ✅ **Live log streaming** - Real-time log viewing with filtering
- ✅ **Health broadcasting** - System status updates to connected clients
- ✅ **User permissions** - Role-based message delivery and access control
- ✅ **Connection cleanup** - Automatic cleanup and ping/pong handling
- ✅ **Message queuing** - Reliable message delivery with error handling
- ✅ **Authentication integration** - Secure WebSocket connections

## 💾 Database & Storage

### **Data Management**
- ✅ **SQLite integration** - Embedded database with full ACID compliance
- ✅ **Migration system** - Version-controlled schema evolution
- ✅ **Repository pattern** - Clean data access layer with interfaces
- ✅ **Transaction management** - Atomic operations with rollback support
- ✅ **Connection pooling** - Optimized database connections
- ✅ **Health monitoring** - Database status and performance tracking
- ✅ **Backup and restore** - Automated backup with point-in-time recovery
- ✅ **Data retention** - Configurable cleanup and archival policies
- ✅ **Database versioning** - Schema migration tracking
- ✅ **Data validation** - Comprehensive input validation and sanitization

## 📡 Comprehensive API

### **RESTful Endpoints**

#### **Core System**
- `GET /health` - System health and status
- `GET /api/status` - Detailed system information
- `GET|PUT /api/config` - Configuration management with validation
- `GET /api/logs` - Log retrieval with filtering and pagination

#### **Authentication & Users**
- `POST /auth/login` - User authentication
- `POST /auth/register` - User registration
- `POST /auth/logout` - User logout
- `GET|POST|PUT|DELETE /api/users` - User management (CRUD)
- `POST /auth/refresh` - Token refresh
- `POST /auth/oauth/{provider}` - OAuth2 authentication

#### **Security & Protection**
- `GET|POST /api/killswitch` - Emergency blocking controls
- `GET /api/ip-protection/status` - IP protection analytics
- `GET|PUT /api/ip-protection/config` - IP protection configuration
- `GET /api/rate-limit/status` - Rate limiting statistics
- `GET|PUT /api/rate-limit/config` - Rate limiting configuration
- `GET /api/ddos-protection/status` - DDoS protection status
- `POST /api/rate-limit/block-ip` - Manual IP blocking
- `POST /api/rate-limit/unblock-ip` - IP unblocking

#### **Moderation & Analytics**
- `POST /api/moderation/test` - Test moderation rules
- `GET /api/moderation/stats` - Comprehensive moderation analytics
- `GET|POST|PUT|DELETE /api/moderation/rules` - Rule management
- `GET /api/pii/analytics` - PII detection statistics
- `GET /api/safety-cockpit/config` - Safety cockpit configuration
- `POST /api/safety-cockpit/test-connection` - Test safety configurations

#### **Provider Management**
- `GET /api/providers` - Provider status and configuration
- `GET /api/providers/{id}/health` - Provider health details
- `PUT /api/providers/{id}/config` - Provider configuration updates

#### **Optimization**
- `GET /api/optimizer/status` - Optimization engine status
- `POST /api/optimizer/experiments` - Create optimization experiments
- `GET /api/optimizer/experiments` - List experiments and results
- `GET /api/evaluator/stats` - Evaluator performance statistics
- `GET /api/evaluator/results` - Evaluation results and metrics

#### **WebSocket Endpoints**
- `WS /ws` - Development WebSocket endpoint
- `WS /ws-secure` - Production WebSocket with authentication

## ⚙️ Configuration Management

### **Layered Configuration System**
- ✅ **YAML-based configuration** - Human-readable primary configuration
- ✅ **Environment variable overrides** - Production-ready environment support
- ✅ **Hot-reloading** - Runtime configuration updates without restart
- ✅ **Schema validation** - Comprehensive validation with detailed error reporting
- ✅ **Configuration versioning** - Change tracking and rollback capability
- ✅ **Backup and restore** - Configuration backup with restore functionality
- ✅ **Audit logging** - Complete audit trail for configuration changes
- ✅ **Configuration diffing** - Compare configuration versions
- ✅ **Rule-based validation** - Custom validation rules and constraints

### **Key Configuration Areas**

#### **Security Settings**
```yaml
security:
  rate_limiting:
    enabled: true
    global:
      requests_per_second: 100
      burst: 200
  ddos_protection:
    enabled: true
    spike_threshold: 1000
  ip_protection:
    enabled: true
    enable_geoblocking: true
    blocked_countries: ["CN", "RU"]
```

#### **Moderation Configuration**
```yaml
moderation:
  enabled: true
  layers:
    - name: "openai"
      enabled: true
      weight: 0.7
    - name: "pii"
      enabled: true
      weight: 0.8
  thresholds:
    block: 0.8
    flag: 0.6
```

#### **Provider Settings**
```yaml
providers:
  - name: "openai"
    type: "openai"
    enabled: true
    api_key: "${OPENAI_API_KEY}"
    health_check_interval: "30s"
```

## 🧪 Testing & Quality Assurance

### **Comprehensive Testing Suite**
- ✅ **Integration tests** - End-to-end system validation
- ✅ **Unit tests** - Component-level testing with mocking
- ✅ **Performance benchmarks** - Load testing and performance validation
- ✅ **WebSocket testing** - Real-time communication testing
- ✅ **Configuration validation** - Schema and rule validation testing
- ✅ **Security testing** - Rate limiting, DDoS protection, and authentication tests
- ✅ **Mock providers** - Comprehensive testing scenarios and edge cases
- ✅ **API testing** - Automated API endpoint validation
- ✅ **Database testing** - Repository and migration testing

### **Development Tools**
- Performance profiling and optimization tools
- Memory leak detection and resource monitoring
- Automated test execution with CI/CD integration
- Code coverage reporting and quality metrics
- Docker containerization for consistent environments

## 📈 Monitoring & Observability

### **System Monitoring**
- ✅ **Real-time dashboards** - Live system metrics and status
- ✅ **Health monitoring** - Component health with automatic alerting
- ✅ **Performance tracking** - Latency, throughput, and resource usage
- ✅ **Error tracking** - Comprehensive error logging and analysis
- ✅ **Audit logging** - Security events and compliance tracking
- ✅ **Resource monitoring** - CPU, memory, connections, and storage
- ✅ **System metrics collection** - Comprehensive system telemetry
- ✅ **Log aggregation** - Centralized logging with search capabilities

### **Alerting & Notifications**
- Configurable alerting thresholds
- Real-time notification system
- Incident response automation
- Performance degradation detection
- Security event alerting

## 🛡️ QT-1 Responsible AI Framework

This implementation fully realizes the QT-1 Responsible AI Framework:

1. **✅ Do No Harm** - Multi-layer content moderation with PII protection
2. **✅ Respect Human Autonomy** - User control, transparency, and consent management
3. **✅ Be Transparent** - Comprehensive logging, audit trails, and explainable decisions
4. **✅ Accept Oversight** - Admin controls, monitoring, and governance tools
5. **✅ Stay Within Scope** - Relevance filtering and context awareness
6. **✅ Protect Privacy** - Data protection, anonymization, and secure handling
7. **✅ Ensure Accountability** - Complete audit trails and responsibility tracking

## 📊 Production Status

### ✅ **Fully Implemented & Production-Ready:**

**Core Infrastructure:**
- High-performance Go backend with efficient resource management
- Comprehensive authentication and authorization system
- Multi-layer security with enterprise-grade protection
- Real-time metrics collection and analytics
- Advanced AI content moderation pipeline
- Intelligent provider management with health monitoring
- WebSocket-based real-time communication
- Complete configuration management system

**Enterprise Features:**
- Role-based access control with granular permissions
- Multi-tier rate limiting with intelligent algorithms
- DDoS protection with adaptive response
- IP-based geoblocking and threat intelligence
- PII detection and automated masking
- A/B testing and optimization engine
- Comprehensive audit logging and compliance tools
- Hot configuration reloading and validation

**Monitoring & Operations:**
- Real-time system health monitoring
- Performance analytics and benchmarking
- Automated backup and recovery systems
- Error tracking and alerting
- Resource usage optimization
- Container-ready deployment

### 🔄 **Next Steps for Enhancement:**

1. **Deployment** - Kubernetes orchestration and cloud deployment
2. **Scaling** - Horizontal scaling and load balancing
3. **Integration** - Additional AI provider integrations
4. **Analytics** - Advanced ML-based analytics and predictions
5. **Compliance** - Additional compliance frameworks and certifications

## 🚀 Getting Started

### **Basic Usage**

1. **Start the system:**
   ```powershell
   cd backend
   .\qt1-middleware.exe
   ```

2. **Access the admin panel:**
   Open `http://localhost:8080` in your browser

3. **Test the API:**
   ```powershell
   curl -X GET http://localhost:8080/health
   ```

4. **Configure providers:**
   Edit `config.yaml` or use the web interface

5. **Monitor real-time metrics:**
   Connect via WebSocket to `/ws` endpoint

### **Advanced Configuration**

See the comprehensive `config.yaml` file for detailed configuration options including security policies, moderation rules, provider settings, and optimization parameters.

### **Environment Variables**

Key environment variables for production deployment:

- `QT1_PORT` - Server port (default: 8080)
- `QT1_HOST` - Server host (default: localhost)
- `OPENAI_API_KEY` - OpenAI API key for moderation
- `QT1_CONFIG_PATH` - Configuration file path
- `QT1_LOG_LEVEL` - Logging level (debug, info, warn, error)
- `QT1_DB_PATH` - Database file path

## 📝 Logs & Audit Trail

All system activities are logged in structured JSON format:

```json
{
  "timestamp": "2025-01-28T14:15:30Z",
  "level": "info",
  "component": "moderation",
  "user_id": "user123",
  "session_id": "session456",
  "action": "content_moderated",
  "score": 0.85,
  "decision": "blocked",
  "metadata": {
    "layer": "openai",
    "confidence": 0.92,
    "processing_time": "15ms"
  }
}
```

## 🔧 Development

### **Building from Source**

```powershell
# Clone the repository
git clone <repository-url>
cd backend

# Install dependencies
go mod tidy

# Build the application
go build -buildvcs=false -o qt1-middleware.exe .

# Run tests
go test ./...

# Run with development settings
go run . config-dev.yaml
```

### **Mock Services**

The system includes comprehensive mock services for testing:

- **Mock AI providers** - Simulate OpenAI, Anthropic responses
- **Mock threat intelligence** - Test security features
- **Load testing tools** - Performance validation
- **Integration test suite** - End-to-end validation

## 🤝 Contributing

This enterprise AI middleware system implements the complete QT-1 Responsible AI framework with production-ready capabilities. All features are fully implemented and tested.

### **Feature Highlights**

- **50+ API endpoints** with comprehensive functionality
- **Multi-layer architecture** with clean separation of concerns
- **Enterprise security** with industry-standard practices
- **Real-time capabilities** with WebSocket integration
- **Advanced analytics** with time-series data collection
- **AI optimization** with genetic algorithms and A/B testing
- **Complete audit trail** for compliance and governance
- **Hot configuration** for zero-downtime updates

---

**🤖 Built for Enterprise AI Governance & Responsible Deployment ⚡**