# Real-Time Metrics Dashboard

## Overview
Create a comprehensive real-time metrics system with live dashboards, performance monitoring, and visual analytics for system health and usage patterns.

## Priority: High
**Estimated Effort:** 2-3 days

## Technical Requirements
- [ ] Metrics collection system
- [ ] Time-series data storage
- [ ] Real-time chart components
- [ ] Performance monitoring
- [ ] Alerting system integration

## Implementation Checklist

### Metrics Collection Backend
- [ ] Create `backend/metrics/collector.go` for metric gathering
- [ ] Implement metric types:
  - [ ] Request count and latency
  - [ ] Provider response times
  - [ ] Moderation hit rates
  - [ ] Error rates and types
  - [ ] WebSocket connection counts
  - [ ] Memory and CPU usage
  - [ ] Kill switch activations
- [ ] Add metrics middleware for HTTP requests
- [ ] Implement configurable metric retention periods

### Data Storage & Aggregation
- [ ] Create in-memory time-series storage for recent data
- [ ] Implement data aggregation (1min, 5min, 1hr buckets)
- [ ] Add metric cleanup for old data
- [ ] Support for custom metric tags and labels
- [ ] Export metrics in Prometheus format (optional)

### Real-Time Dashboard Frontend
- [ ] Install chart libraries (`recharts` or `chart.js`)
- [ ] Create `MetricsDashboard.tsx` component
- [ ] Implement real-time chart components:
  - [ ] Request rate over time (line chart)
  - [ ] Response time distribution (histogram)
  - [ ] Provider health status (gauge)
  - [ ] Moderation statistics (pie chart)
  - [ ] System resource usage (area chart)
- [ ] Add time range selectors (1h, 6h, 24h, 7d)
- [ ] Implement auto-refresh with WebSocket updates

### Key Performance Indicators (KPIs)
- [ ] Request success rate (%)
- [ ] Average response time (ms)
- [ ] Requests per second (RPS)
- [ ] Error rate by type
- [ ] Moderation block rate
- [ ] Provider availability score
- [ ] System uptime percentage

### Alerting Integration
- [ ] Define alert thresholds for key metrics
- [ ] Implement alert generation logic
- [ ] Add alert notification system (email, Slack, webhook)
- [ ] Create alert management UI
- [ ] Add alert history and acknowledgment

### API Endpoints
- [ ] `GET /api/metrics/current` - Current system metrics
- [ ] `GET /api/metrics/history` - Historical metric data
- [ ] `GET /api/metrics/alerts` - Active alerts
- [ ] `POST /api/metrics/alerts/ack` - Acknowledge alerts

## Dashboard Layout
```
┌─────────────────────────────────────────────────────────────┐
│ System Health Overview                                      │
├─────────────────┬─────────────────┬─────────────────────────┤
│ Requests/sec    │ Avg Response    │ Error Rate              │
│ [123.4]         │ [45ms]          │ [0.2%]                  │
├─────────────────┴─────────────────┴─────────────────────────┤
│ Request Rate Chart (Real-time)                             │
│ [LINE CHART - 24hr view]                                   │
├─────────────────────────────────────────────────────────────┤
│ Provider Status    │ Moderation Stats                        │
│ ├ OpenAI: ✅ 99%  │ ├ Total Requests: 1,234                │
│ ├ Local: ✅ 95%   │ ├ Blocked: 23 (1.9%)                   │
│ └ Mock: ✅ 100%   │ └ Top Block Reason: Inappropriate       │
└─────────────────────────────────────────────────────────────┘
```

## Configuration Schema
```yaml
metrics:
  enabled: true
  collection_interval: 10s
  retention_period: 24h
  aggregation_buckets: ["1m", "5m", "1h"]
  alerts:
    error_rate_threshold: 5.0
    response_time_threshold: 1000
    notification_channels: ["webhook", "email"]
```

## Testing Requirements
- [ ] Unit tests for metrics collector
- [ ] Integration tests for metric aggregation
- [ ] Frontend tests for dashboard components
- [ ] Performance tests for metric collection overhead
- [ ] Load testing with high metric volume

## Acceptance Criteria
- [ ] Real-time metrics update every 10 seconds
- [ ] Dashboard shows historical data up to 24 hours
- [ ] Charts are responsive and performant with 1000+ data points
- [ ] Alerts trigger within 30 seconds of threshold breach
- [ ] Metrics collection adds <1% CPU overhead
- [ ] Dashboard loads in under 2 seconds
- [ ] All KPIs are clearly visible and actionable

## Dependencies
- [ ] Task #01 (WebSocket Integration) for real-time updates
- [ ] Enhanced logging system for metric collection

## Files to Modify/Create
- `backend/metrics/collector.go` (new)
- `backend/metrics/storage.go` (new)
- `backend/metrics/alerts.go` (new)
- `backend/api/metrics.go` (new)
- `frontend/src/components/MetricsDashboard.tsx` (new)
- `frontend/src/components/charts/` (new directory)
- `frontend/package.json` (add chart dependencies)