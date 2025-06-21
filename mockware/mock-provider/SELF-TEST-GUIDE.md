# 🤖 QT-1 Mock Provider Self-Test Guide

This guide explains how to run comprehensive self-tests on the QT-1 Mock Provider **without any middleware** - the application fires requests directly at itself to verify all functionality.

## 🚀 Quick Start

### Option 1: Complete Automated Test (Recommended)
Starts the server, runs all tests, and stops the server automatically:

```bash
npm test
```

### Option 2: Manual Testing (Server Already Running)
If you have the server running in another terminal:

```bash
npm run test:manual
```

### Option 3: Shell Script Testing
Run the curl-based endpoint tests:

```bash
npm run test:endpoints
```

## 🧪 What Gets Tested

The self-test suite performs **24 comprehensive tests** across 8 categories:

### 1. Health Checks
- ✅ Basic health endpoint
- ✅ Detailed system status  
- ✅ Load testing capability

### 2. OpenAI API Compatibility
- ✅ Chat completions with proper response format
- ✅ Legacy text completions
- ✅ Content moderation API

### 3. Anthropic API Compatibility  
- ✅ Messages API with Claude format
- ✅ Legacy completion API

### 4. Generic APIs
- ✅ Generic chat interface
- ✅ Text analysis with PII detection
- ✅ Token counting and cost calculation
- ✅ Batch processing (multiple requests)

### 5. Content Analysis
- ✅ Violence/harmful content detection
- ✅ PII (Personal Information) detection  
- ✅ Prompt injection attempt detection

### 6. Error Simulation
- ✅ Invalid JSON handling
- ✅ Missing required fields
- ✅ 404 endpoint handling

### 7. Performance Testing
- ✅ Response time consistency
- ✅ Latency simulation verification
- ✅ Concurrent request handling

### 8. Edge Cases
- ✅ Empty message handling
- ✅ Very long message processing
- ✅ Special characters and emojis
- ✅ Multiple language detection

## 📊 Expected Results

A successful test run typically shows:
- **Success Rate**: 80-90% (some tests intentionally check edge cases)
- **Response Times**: 100-500ms depending on simulated model complexity
- **Coverage**: All major API endpoints and features
- **Error Handling**: Proper error responses for invalid inputs

### Sample Output:
```
📊 Test Summary
===============
Total Tests: 24
✅ Passed: 20
❌ Failed: 4  
📈 Success Rate: 83.3%
⏱️  Average Test Duration: 226ms
```

## 🔧 Advanced Testing Options

### Custom Base URL
Test against a different server instance:

```bash
node self-test.js http://localhost:8082
```

### Individual Test Categories
Run specific test suites by modifying the `self-test.js` file or use the dashboard for targeted testing.

### Load Testing
Test performance under load:

```bash
curl "http://localhost:8081/health/loadtest?iterations=100"
```

## 🌐 Web Dashboard Testing

Open the web dashboard for interactive testing:

1. Start the server: `npm start`
2. Open `dashboard/index.html` in your browser
3. Use the "Quick API Tests" buttons for individual endpoint testing
4. Monitor real-time statistics and health metrics

## 🐛 Troubleshooting Common Issues

### Server Won't Start
```bash
# Check if port is available
lsof -i :8081

# Try different port
PORT=8082 npm test
```

### Tests Timing Out
```bash
# Increase timeouts in self-test.js
# Or run with more verbose output
DEBUG=1 npm test
```

### High Failure Rate
- Check server logs for errors
- Verify environment configuration
- Ensure all dependencies are installed

## 🔍 What Self-Testing Proves

This comprehensive self-test demonstrates that the mock provider can:

1. **🎯 API Compatibility**: Correctly emulate OpenAI, Anthropic, and generic APIs
2. **🛡️ Security Analysis**: Detect harmful content, PII, and prompt injections  
3. **⚡ Performance**: Handle concurrent requests with realistic latency
4. **🔧 Error Handling**: Gracefully handle invalid inputs and edge cases
5. **📊 Monitoring**: Provide health metrics and system status
6. **🚀 Reliability**: Maintain stable operation under various conditions

## 📈 Integration with QT-1 Middleware

Once self-tests pass, you can confidently point your QT-1 middleware to this mock provider:

```yaml
# QT-1 Configuration
providers:
  openai:
    endpoint: "http://localhost:8081/v1"
    api_key: "mock-key-not-required"
  anthropic:  
    endpoint: "http://localhost:8081/v1"
    api_key: "mock-key-not-required"
```

The self-test validates that all middleware test scenarios will work correctly without consuming real API credits.

## 🎯 Benefits of Self-Testing

- **🔄 No Middleware Required**: Tests the provider in isolation
- **💰 Zero API Costs**: No real API calls made
- **⚡ Fast Feedback**: Complete test suite runs in under 1 minute
- **🎯 Comprehensive Coverage**: Tests all endpoints and edge cases
- **📊 Detailed Reporting**: Clear pass/fail status with performance metrics
- **🔧 CI/CD Ready**: Can be integrated into automated testing pipelines

Run `npm test` to verify your mock provider is ready for QT-1 middleware testing!