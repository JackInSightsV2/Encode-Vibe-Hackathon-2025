# 🤖 QT-1 Mockware - Complete Testing Suite

**Comprehensive testing solution for QT-1 middleware without using real API credits**

Mockware provides everything you need to thoroughly test your QT-1 middleware system:
- **Load Testing Client**: Sends attack vectors and test scenarios to your middleware
- **Universal Mock Provider**: Responds to any AI provider request format without real API costs

## 🚀 Quick Start

### One-Command Startup

```bash
# Linux/macOS
./start-mockware.sh

# Windows
start-mockware.bat

# Or with npm
npm start
```

This starts both services automatically:
- **Load Testing Client**: http://localhost:3000
- **Universal Mock Provider**: http://localhost:8081

## 📦 What's Included

### Part 1: Load Testing Client (`testing-client/`)
- **500+ Test Scenarios**: Moderation triggers, prompt injections, PII, combinations
- **4 Load Testing Patterns**: Burst, Sustained, Mixed, Spike testing
- **Real-time Monitoring**: Live test progress with Socket.IO
- **Comprehensive Reporting**: Console, HTML, and JSON reports
- **Web Dashboard**: Visual interface for test management

### Part 2: Universal Mock Provider (`mock-provider/`)
- **🎭 Universal Endpoint**: Handles ANY AI provider request format automatically  
- **🧠 Smart Detection**: Recognizes OpenAI, Anthropic, or custom formats
- **🛡️ Content Analysis**: Built-in moderation, PII, and prompt injection detection
- **⚡ Realistic Simulation**: Latency, errors, rate limiting, token counting
- **📊 Health Monitoring**: Comprehensive system status and metrics

## 🎯 Perfect for QT-1 Middleware Testing

### Test Your Security Features
- **Content Moderation**: Violence, hate speech, explicit content detection
- **PII Protection**: Email, phone, SSN, credit card detection
- **Prompt Injection Defense**: Direct, encoded, and sophisticated attack patterns
- **Rate Limiting**: High-volume request handling
- **Error Resilience**: Network failures, timeouts, API errors

### Zero API Costs
- **No Real API Calls**: All responses are intelligently mocked
- **Unlimited Testing**: Test 10,000+ requests without spending money
- **Multiple Providers**: Test OpenAI, Anthropic, and custom provider integrations

## 📊 Service Management

### Start/Stop Services
```bash
./start-mockware.sh start    # Start both services
./start-mockware.sh stop     # Stop both services  
./start-mockware.sh restart  # Restart both services
./start-mockware.sh status   # Check service status
./start-mockware.sh logs     # View live logs
```

### Individual Service Control
```bash
# Mock Provider only
cd mock-provider && npm start

# Testing Client only  
cd testing-client && npm start
```

## 🔧 Configuration

### QT-1 Middleware Setup
Point your middleware to the universal mock provider:

```yaml
# config.yaml - Works for ANY provider format
providers:
  openai:
    endpoint: "http://localhost:8081"
    api_key: "mock-key-not-used"
  anthropic:
    endpoint: "http://localhost:8081"  
    api_key: "mock-key-not-used"
  custom_provider:
    endpoint: "http://localhost:8081"
    api_key: "mock-key-not-used"
```

### Environment Configuration
Both services support environment variable configuration:

```bash
# Mock Provider
PORT=8081
RESPONSE_DELAY_MIN=100
RESPONSE_DELAY_MAX=500
ERROR_RATE=0.05

# Testing Client  
PORT=3000
DEFAULT_TARGET_URL=http://localhost:8080
MAX_CONCURRENT_TESTS=100
```

## 🧪 Testing Scenarios

### Pre-built Test Categories
1. **Content Moderation** (100+ tests)
   - Violence and threats
   - Hate speech variations
   - Explicit content
   - Self-harm indicators

2. **Prompt Injection** (150+ tests)
   - Direct instruction override
   - Encoded payloads
   - Role-playing attempts
   - System prompt extraction

3. **PII Detection** (80+ tests)
   - Email addresses
   - Phone numbers
   - Social Security Numbers
   - Credit card numbers

4. **Combination Attacks** (120+ tests)
   - Multi-vector scenarios
   - Misspelled variations
   - Unicode encoding
   - Social engineering

5. **Performance Testing** (50+ tests)
   - High-volume bursts
   - Sustained load
   - Memory usage patterns
   - Response time consistency

### Custom Test Scenarios
Add your own test cases by editing:
- `testing-client/lib/scenarios/` - Test message collections
- `mock-provider/responses/templates/` - Response templates

## 📈 Monitoring & Analytics

### Real-time Dashboards
- **Testing Dashboard**: http://localhost:3000
- **Provider Dashboard**: `mock-provider/dashboard/index.html`

### Metrics Tracked
- Request/response rates
- Error detection accuracy  
- Content analysis results
- Performance benchmarks
- System resource usage

### Reporting Formats
- **Console**: Real-time colored output
- **HTML**: Detailed visual reports
- **JSON**: Data for analysis tools
- **Logs**: Structured logging for debugging

## 🐳 Docker Deployment

### Quick Docker Start
```bash
docker-compose up -d
```

### Individual Services
```bash
# Mock Provider
cd mock-provider && docker build -t qt1-mock-provider .
docker run -p 8081:8081 qt1-mock-provider

# Testing Client
cd testing-client && docker build -t qt1-testing-client .
docker run -p 3000:3000 qt1-testing-client
```

## 🔍 Self-Testing

### Verify Mock Provider
```bash
cd mock-provider && npm test
```

### Test Load Client
```bash
cd testing-client && npm test
```

### Full System Test
```bash
npm run test:all
```

## 📁 Directory Structure

```
Mockware/
├── start-mockware.sh          # Main startup script (Linux/macOS)
├── start-mockware.bat         # Main startup script (Windows)
├── package.json               # Root package configuration
├── docker-compose.yml         # Docker orchestration
├── README.md                  # This file
│
├── testing-client/            # Load Testing Client (Part 1)
│   ├── server.js              # Express server with Socket.IO
│   ├── public/                # Web dashboard
│   ├── lib/                   # Test generators and load runners
│   └── scenarios/             # 500+ test scenarios
│
├── mock-provider/             # Universal Mock Provider (Part 2)
│   ├── server.js              # Express server
│   ├── routes/                # Universal endpoint
│   ├── lib/                   # Response generation & analysis
│   ├── responses/             # Templates & personalities
│   └── dashboard/             # Monitoring interface
│
└── logs/                      # Service logs (created at runtime)
```

## 🎯 Use Cases

### Development Testing
- Validate middleware security features
- Test error handling and resilience
- Performance benchmark under load
- Integration testing without API costs

### QA/Staging Testing  
- Comprehensive security validation
- Load testing before production
- Provider failover testing
- End-to-end workflow validation

### Production Readiness
- Stress testing with realistic scenarios
- Security posture validation
- Performance optimization
- Disaster recovery testing

## 🤝 Contributing

### Adding New Test Scenarios
1. Edit `testing-client/lib/scenarios/`
2. Add new message patterns
3. Include misspellings and variations
4. Test with the load runner

### Improving Mock Responses
1. Edit `mock-provider/responses/templates/`
2. Add new response patterns
3. Update personality configurations
4. Test with self-test suite

## 📞 Support

- **Documentation**: See individual README files in each directory
- **Issues**: Check logs in `logs/` directory
- **Health Checks**: Access `/health` endpoints for diagnostics
- **Self-Tests**: Run `npm test` in each service directory

## 🎉 Ready to Test!

1. **Start Mockware**: `./start-mockware.sh`
2. **Configure QT-1**: Point to `http://localhost:8081`
3. **Run Tests**: Access http://localhost:3000
4. **Monitor Results**: Watch real-time dashboards

Your QT-1 middleware is now ready for comprehensive testing without any API costs! 🚀