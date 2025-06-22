# AI Safety Cockpit: Real-Time Moderation Intelligence

## Overview
Transform QT-1 middleware into an intelligent safety observatory using Opik's tracing to create a real-time 3D visualization dashboard showing threat landscapes, moderation effectiveness, and provider performance vs safety trade-offs.

## Priority: High (Hackathon Submission)
**Estimated Effort:** 3-4 days

## Technical Requirements
- [ ] Opik SDK integration for trace collection
- [ ] Real-time streaming of moderation decisions
- [ ] 3D visualization using Three.js/D3.js
- [ ] Custom Opik evaluators for each moderation layer
- [ ] WebSocket integration for live updates
- [ ] Historical data analysis capabilities

## Implementation Checklist

### Opik Integration
- [ ] Install Opik SDK dependencies
- [ ] Create `backend/opik/` directory structure
- [ ] Implement Opik trace collector for middleware events
- [ ] Create custom spans for each moderation layer:
  - [ ] Regex pattern matching spans
  - [ ] LLM moderation evaluation spans
  - [ ] PII detection spans
  - [ ] Security validation spans
- [ ] Implement Opik scoring for moderation decisions
- [ ] Create Opik experiments for A/B testing rules

### Real-Time Data Pipeline
- [ ] Create event streaming architecture
- [ ] Implement WebSocket bridge to Opik data
- [ ] Build aggregation service for metrics
- [ ] Create time-series data collection
- [ ] Implement data buffering for visualization

### 3D Visualization Dashboard
- [ ] Create `frontend/src/components/SafetyCockpit/`
- [ ] Implement Three.js scene for 3D visualization
- [ ] Create threat landscape visualization:
  - [ ] Prompt injection attempts as red spikes
  - [ ] PII leaks as orange bubbles
  - [ ] Safe requests as green particles
- [ ] Build provider performance comparison view
- [ ] Implement real-time threat heatmap
- [ ] Create interactive navigation controls

### Custom Opik Evaluators
- [ ] Regex Pattern Effectiveness Evaluator
- [ ] LLM Moderation Accuracy Evaluator
- [ ] PII Detection Coverage Evaluator
- [ ] False Positive Rate Evaluator
- [ ] Response Time Impact Evaluator
- [ ] Security Threat Severity Evaluator

### Moderation Intelligence Features
- [ ] Pattern recognition for emerging threats
- [ ] Automated threat classification
- [ ] Provider safety scoring system
- [ ] Real-time anomaly detection
- [ ] Predictive threat analysis

### Safety Metrics & Analytics
- [ ] Safety score calculation per provider
- [ ] Moderation overhead tracking
- [ ] False positive/negative rates
- [ ] Threat type distribution
- [ ] Geographic threat mapping
- [ ] Time-based threat patterns

### Interactive Features
- [ ] Click on threats for detailed traces
- [ ] Time travel through security events
- [ ] Filter by threat type/severity
- [ ] Provider comparison mode
- [ ] Export safety reports
- [ ] Alert configuration panel

### Configuration
```yaml
opik:
  api_key: "${OPIK_API_KEY}"
  project_name: "qt1-safety-cockpit"
  
safety_cockpit:
  visualization:
    refresh_rate: 100ms
    max_points: 10000
    threat_retention: 24h
    
  evaluators:
    - name: "regex_effectiveness"
      threshold: 0.8
    - name: "llm_accuracy"
      threshold: 0.9
    - name: "pii_coverage"
      threshold: 0.95
      
  alerts:
    threat_spike_threshold: 100
    false_positive_threshold: 0.2
```

### API Endpoints
- [ ] `GET /api/safety/cockpit/stream` - Real-time event stream
- [ ] `GET /api/safety/cockpit/metrics` - Current safety metrics
- [ ] `GET /api/safety/cockpit/threats` - Threat analysis data
- [ ] `GET /api/safety/cockpit/providers` - Provider safety scores
- [ ] `POST /api/safety/cockpit/export` - Export safety report

### Testing & Validation
- [ ] Load test with simulated threats
- [ ] Verify real-time performance
- [ ] Test 3D visualization performance
- [ ] Validate Opik trace accuracy
- [ ] Security event correlation testing

## Acceptance Criteria
- [ ] Real-time 3D visualization updates < 100ms
- [ ] All moderation events traced to Opik
- [ ] Interactive threat exploration works
- [ ] Provider safety scores accurately calculated
- [ ] Custom evaluators provide actionable insights
- [ ] Dashboard handles 1000+ events/second
- [ ] Export functionality generates comprehensive reports

## Dependencies
- [ ] Opik SDK
- [ ] Three.js for 3D visualization
- [ ] D3.js for data visualization
- [ ] WebSocket infrastructure
- [ ] Existing QT-1 middleware

## Files to Create/Modify
- `backend/opik/tracer.go` (new)
- `backend/opik/evaluators.go` (new)
- `backend/api/safety_cockpit.go` (new)
- `frontend/src/components/SafetyCockpit/` (new directory)
- `frontend/src/services/opikService.ts` (new)
- `backend/config.yaml` (add Opik configuration)