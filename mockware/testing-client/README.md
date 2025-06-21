# QT-1 Middleware Testing Client

A comprehensive load testing client for the QT-1 middleware, designed to test security features, moderation capabilities, and performance without consuming real API credits.

## Features

- **Web-based UI** for easy test configuration and monitoring
- **Real-time metrics** via WebSocket connections
- **Multiple test types**: Burst, Sustained, Mixed, and Spike patterns
- **Comprehensive test scenarios**:
  - Moderation testing (violence, hate speech, explicit content)
  - Prompt injection detection (direct, encoded, sophisticated)
  - PII detection (emails, phones, SSNs, credit cards)
  - Combination attacks and edge cases
  - DDoS simulation patterns
- **Pre-configured test profiles** for common testing scenarios
- **Detailed reporting** in HTML and JSON formats
- **Real-time progress tracking** with live charts

## Installation

```bash
cd testing-client
npm install
```

## Configuration

Edit `.env` file to configure:

```env
PORT=3001
MIDDLEWARE_URL=http://localhost:8080
MOCK_PROVIDER_URL=http://localhost:8081
REPORT_OUTPUT_DIR=./reports
MAX_CONCURRENT_REQUESTS=1000
REQUEST_TIMEOUT=30000
```

## Usage

1. Start the testing client:
   ```bash
   npm start
   ```

2. Open your browser to `http://localhost:3001`

3. Configure your test:
   - Select a pre-configured test profile or create custom settings
   - Choose test type (Burst, Sustained, Mixed, Spike)
   - Adjust request distribution percentages
   - Set middleware URL

4. Click "Start Test" and monitor real-time progress

5. View results and download reports when complete

## Test Profiles

### Quick Test
- 30-second basic functionality test
- 10 requests/second
- Mix of clean messages and basic attacks

### Aggressive Security Test
- High-intensity security testing
- 5,000 total requests at 50 RPS
- Heavy focus on malicious content

### Standard Load Test
- 5-minute sustained load test
- 50 requests/second
- Simulates normal traffic patterns

### DDoS Simulation
- Tests spike handling and rate limiting
- Variable request rates (10-1000 RPS)
- Includes large payloads and cache busters

### PII Detection Test
- Comprehensive PII testing
- Tests all PII types and obfuscation methods
- 2,000 requests at 20 RPS

### Injection Comprehensive
- All injection types including encoded and multilingual
- 3,000 requests at 30 RPS
- Tests sophisticated bypass attempts

### Edge Cases
- Unicode abuse, boundary testing, parsing attacks
- 1,500 requests at 15 RPS
- Unusual inputs and encoding tricks

## API Endpoints

- `POST /api/test/start` - Start a new test
- `GET /api/test/status/:id` - Get test status
- `GET /api/test/results/:id` - Get test results
- `GET /api/test/active` - List active tests
- `GET /api/scenarios` - Get available test scenarios
- `GET /api/profiles` - Get test profiles

## WebSocket Events

- `start-test` - Start a new test
- `stop-test` - Stop running test
- `test-started` - Test has started
- `test-progress` - Real-time progress updates
- `test-completed` - Test finished
- `test-error` - Test encountered an error

## Test Scenarios

### Moderation Testing
- Violence and threats
- Hate speech and discrimination
- Explicit content
- Harmful instructions
- Spam and phishing

### Prompt Injection
- Direct instruction overrides
- Base64/ROT13/Hex encoding
- Role manipulation
- Jailbreak attempts
- Multilingual injections

### PII Detection
- Email addresses (standard and obfuscated)
- Phone numbers (US and international)
- Social Security Numbers
- Credit card numbers
- Physical addresses

### Combination Attacks
- PII + Injection
- Violence + Misspelling
- Hate speech + Encoding
- Multi-vector attacks
- Context switching

### Edge Cases
- Unicode abuse (zero-width characters, homoglyphs)
- Boundary testing (max/min length)
- Parsing attacks (JSON, XML, HTML)
- Semantic tricks (double negatives, euphemisms)
- Timing-based attacks

## Reports

Reports are generated in multiple formats:

### Console Report
- Summary statistics
- Performance metrics
- Category breakdown
- Error details

### HTML Report
- Interactive charts
- Visual metrics
- Detailed tables
- Downloadable

### JSON Report
- Complete test data
- Response time histograms
- Machine-readable format
- Summary file

## Development

```bash
# Run in development mode
npm run dev

# Run tests
npm test
```

## Troubleshooting

1. **Connection refused**: Ensure middleware is running on configured port
2. **High error rate**: Check middleware logs for specific errors
3. **Slow performance**: Reduce concurrent requests or request rate
4. **Memory issues**: Lower MAX_CONCURRENT_REQUESTS in .env

## License

MIT