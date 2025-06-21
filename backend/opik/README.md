# Opik Integration for QT-1 Middleware

This package provides integration with Opik for distributed tracing and evaluation of the QT-1 middleware's AI safety features.

## Features

- **Distributed Tracing**: Track requests through all moderation layers
- **Custom Spans**: Specialized span types for different operations
- **Batching**: Efficient batching of traces and spans
- **Evaluators**: Custom evaluators for measuring effectiveness
- **Async Processing**: Non-blocking trace collection

## Configuration

Add to your `config.yaml`:

```yaml
opik:
  enabled: true
  api_key: ""  # Set via OPIK_API_KEY environment variable
  project_name: "qt1-safety-cockpit"
  base_url: "https://api.opik.com"
  batch_size: 100
  flush_interval: "5s"
  
  tracing:
    enabled: true
    sample_rate: 1.0
    trace_moderation: true
    trace_providers: true
    trace_security: true
    
  evaluations:
    enabled: true
    run_async: true
    timeout: "10s"
    evaluators:
      - "regex_effectiveness"
      - "llm_accuracy"
      - "pii_coverage"
      - "false_positive_rate"
      - "response_time"
```

## Usage

### Basic Tracing

```go
// Initialize Opik client
opikClient, err := opik.NewOpikClient(config.AppConfig.Opik)
if err != nil {
    log.Fatal(err)
}
defer opikClient.Close()

// Create a trace for moderation
request := opik.ModerationRequest{
    ID:        requestID,
    Provider:  provider,
    Timestamp: time.Now(),
    Content:   content,
    UserID:    userID,
    SessionID: sessionID,
    IPAddress: ipAddress,
    Endpoint:  endpoint,
}

trace, err := opikClient.TraceModeration(ctx, request)
if err != nil {
    return err
}

// End trace when done
defer opikClient.EndTrace(trace, map[string]interface{}{
    "result": result,
})
```

### Creating Spans

```go
// Regex moderation span
regexSpan := opikClient.StartSpan(trace, opik.SpanTypeRegex, map[string]interface{}{
    "content": content,
})
defer regexSpan.End()

// Process and set output
result := performRegexModeration(content)
regexSpan.SetOutput(map[string]interface{}{
    "blocked":       result.Blocked,
    "matched_rules": result.MatchedRules,
    "confidence":    result.Confidence,
})

// Add metadata
regexSpan.SetMetadata(map[string]interface{}{
    "duration_ms": time.Since(start).Milliseconds(),
    "rule_count":  len(rules),
})

// Set evaluation score
regexSpan.SetScore("effectiveness", calculateEffectiveness(result))
```

### Span Types

- `SpanTypeRegex`: Regex-based moderation
- `SpanTypeLLM`: LLM-based moderation
- `SpanTypePII`: PII detection
- `SpanTypeSecurity`: Security validation
- `SpanTypeProvider`: Provider API requests
- `SpanTypeRateLimit`: Rate limiting checks
- `SpanTypeAuth`: Authentication
- `SpanTypeCache`: Cache operations

### Custom Evaluators

```go
type MyEvaluator struct {
    opik.BaseEvaluator
}

func (e *MyEvaluator) Name() string {
    return "my_custom_evaluator"
}

func (e *MyEvaluator) Evaluate(trace *opik.Trace) (float64, map[string]interface{}) {
    // Custom evaluation logic
    score := calculateScore(trace)
    details := map[string]interface{}{
        "metric1": value1,
        "metric2": value2,
    }
    return score, details
}

// Register evaluator
opikClient.RegisterEvaluator(&MyEvaluator{})
```

## Testing

### Unit Tests
```bash
go test ./backend/opik
```

### Integration Tests
```bash
go test ./backend/opik -run Integration
```

### With Real Opik API
```bash
OPIK_API_KEY=your-key go test ./backend/opik -tags=integration
```

## Best Practices

1. **Always end traces and spans**: Use `defer` to ensure cleanup
2. **Add meaningful metadata**: Include relevant context for debugging
3. **Use appropriate span types**: Choose the right span type for clarity
4. **Set evaluation scores**: Help track effectiveness over time
5. **Handle errors gracefully**: Don't let tracing errors affect main flow

## Performance Considerations

- Traces and spans are batched for efficiency
- Evaluators run asynchronously by default
- Sampling can be configured via `sample_rate`
- Flush interval can be tuned based on load

## Troubleshooting

### Traces not appearing in Opik
1. Check if Opik is enabled in config
2. Verify API key is set correctly
3. Check batch size and flush interval
4. Look for errors in logs

### High memory usage
1. Reduce batch size
2. Decrease flush interval
3. Lower sample rate for high-traffic endpoints

### Slow response times
1. Ensure evaluators are async
2. Check network latency to Opik
3. Consider reducing trace detail