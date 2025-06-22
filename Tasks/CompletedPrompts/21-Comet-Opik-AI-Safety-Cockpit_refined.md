# AI Safety Cockpit: Real-Time Moderation Intelligence - Detailed Implementation Plan

## Overview
Transform QT-1 middleware into an intelligent safety observatory using Opik's tracing to create a real-time 3D visualization dashboard showing threat landscapes, moderation effectiveness, and provider performance vs safety trade-offs.

## Implementation Cycles

### Cycle 21A: Opik SDK Integration & Basic Tracing
**Duration:** 4-6 hours

#### Backend Implementation
1. **Install Opik SDK**
   ```bash
   go get github.com/comet-ml/opik-go
   ```

2. **Create Opik Integration Structure**
   ```
   backend/
   ├── opik/
   │   ├── client.go          # Opik client initialization
   │   ├── tracer.go          # Trace management
   │   ├── spans.go           # Span creation utilities
   │   └── config.go          # Opik configuration
   ```

3. **Implement Opik Client (`backend/opik/client.go`)**
   ```go
   package opik

   import (
       "context"
       "github.com/comet-ml/opik-go"
       "log"
   )

   type OpikClient struct {
       client    *opik.Client
       projectID string
       tracer    *opik.Tracer
   }

   func NewOpikClient(apiKey, projectName string) (*OpikClient, error) {
       client, err := opik.NewClient(opik.Config{
           APIKey:     apiKey,
           ProjectName: projectName,
       })
       if err != nil {
           return nil, err
       }

       tracer := client.Tracer("qt1-middleware")
       
       return &OpikClient{
           client:    client,
           projectID: projectName,
           tracer:    tracer,
       }, nil
   }
   ```

4. **Create Trace Wrapper (`backend/opik/tracer.go`)**
   ```go
   func (oc *OpikClient) TraceModeration(ctx context.Context, request ModerationRequest) (*opik.Trace, error) {
       trace := oc.tracer.StartTrace(ctx, "moderation_request", opik.TraceOptions{
           Input: map[string]interface{}{
               "request_id": request.ID,
               "provider":   request.Provider,
               "timestamp":  request.Timestamp,
           },
           Metadata: map[string]interface{}{
               "user_id":    request.UserID,
               "session_id": request.SessionID,
               "ip_address": request.IPAddress,
           },
       })
       
       return trace, nil
   }
   ```

5. **Update Configuration**
   ```yaml
   # backend/config.yaml
   opik:
     enabled: true
     api_key: "${OPIK_API_KEY}"
     project_name: "qt1-safety-cockpit"
     batch_size: 100
     flush_interval: "5s"
   ```

#### Testing Cycle 21A
1. **Unit Tests**
   ```go
   func TestOpikClientInitialization(t *testing.T) {
       client, err := NewOpikClient("test-key", "test-project")
       assert.NoError(t, err)
       assert.NotNil(t, client)
   }
   ```

2. **Integration Test**
   - Start mock Opik server
   - Send test moderation request
   - Verify trace creation
   - Check trace metadata

3. **Manual Testing**
   - Set up Opik account
   - Configure API key
   - Send test requests
   - Verify traces appear in Opik dashboard

### Cycle 21B: Moderation Layer Span Implementation
**Duration:** 6-8 hours

#### Backend Implementation
1. **Create Span Types (`backend/opik/spans.go`)**
   ```go
   type SpanType string

   const (
       SpanTypeRegex      SpanType = "regex_moderation"
       SpanTypeLLM        SpanType = "llm_moderation"
       SpanTypePII        SpanType = "pii_detection"
       SpanTypeSecurity   SpanType = "security_validation"
       SpanTypeProvider   SpanType = "provider_request"
   )

   func (oc *OpikClient) StartSpan(trace *opik.Trace, spanType SpanType, input interface{}) *opik.Span {
       return trace.StartSpan(string(spanType), opik.SpanOptions{
           Input: input,
           SpanKind: opik.SpanKindInternal,
       })
   }
   ```

2. **Integrate with Regex Moderation**
   ```go
   // backend/moderation/regex.go
   func (rm *RegexModerator) ModerateWithTracing(ctx context.Context, content string, span *opik.Span) (*ModerationResult, error) {
       defer span.End()
       
       startTime := time.Now()
       result := rm.Moderate(content)
       
       span.SetOutput(map[string]interface{}{
           "blocked":       result.Blocked,
           "matched_rules": result.MatchedRules,
           "confidence":    result.Confidence,
       })
       
       span.SetMetadata(map[string]interface{}{
           "duration_ms": time.Since(startTime).Milliseconds(),
           "rule_count":  len(rm.rules),
       })
       
       // Score the moderation effectiveness
       score := rm.calculateEffectivenessScore(result)
       span.SetScore("effectiveness", score)
       
       return result, nil
   }
   ```

3. **Integrate with LLM Moderation**
   ```go
   // backend/moderation/llm.go
   func (lm *LLMModerator) ModerateWithTracing(ctx context.Context, content string, span *opik.Span) (*ModerationResult, error) {
       defer span.End()
       
       span.SetMetadata(map[string]interface{}{
           "model":      lm.model,
           "provider":   lm.provider,
           "threshold":  lm.threshold,
       })
       
       result, err := lm.callLLMProvider(ctx, content)
       if err != nil {
           span.SetError(err)
           return nil, err
       }
       
       span.SetOutput(map[string]interface{}{
           "categories": result.Categories,
           "scores":     result.Scores,
           "flagged":    result.Flagged,
       })
       
       return result, nil
   }
   ```

4. **Integrate with PII Detection**
   ```go
   // backend/moderation/pii.go
   func (pd *PIIDetector) DetectWithTracing(ctx context.Context, content string, span *opik.Span) (*PIIResult, error) {
       defer span.End()
       
       detections := pd.detectPII(content)
       
       span.SetOutput(map[string]interface{}{
           "pii_found":     len(detections) > 0,
           "pii_types":     pd.getPIITypes(detections),
           "pii_count":     len(detections),
           "masked_output": pd.maskPII(content, detections),
       })
       
       // Custom evaluator for PII detection accuracy
       span.SetScore("pii_detection_accuracy", pd.calculateAccuracy(detections))
       
       return &PIIResult{
           Detections: detections,
           MaskedText: pd.maskPII(content, detections),
       }, nil
   }
   ```

#### Testing Cycle 21B
1. **Span Creation Tests**
   ```go
   func TestModerationSpans(t *testing.T) {
       trace := createTestTrace()
       
       // Test regex span
       regexSpan := opikClient.StartSpan(trace, SpanTypeRegex, testInput)
       assert.NotNil(t, regexSpan)
       
       // Test span hierarchy
       assert.Equal(t, trace.ID, regexSpan.ParentID)
   }
   ```

2. **End-to-End Moderation Test**
   - Create test request with known patterns
   - Trace through all moderation layers
   - Verify span relationships
   - Check span metadata and scores

### Cycle 21C: Real-Time Event Streaming
**Duration:** 6-8 hours

#### Backend Implementation
1. **Create Event Stream Manager (`backend/opik/stream.go`)**
   ```go
   type EventStreamManager struct {
       opikClient *OpikClient
       buffer     *ring.Buffer
       subscribers map[string]chan Event
       mu         sync.RWMutex
   }

   type Event struct {
       Type      string                 `json:"type"`
       Timestamp time.Time             `json:"timestamp"`
       TraceID   string                `json:"trace_id"`
       Data      map[string]interface{} `json:"data"`
   }

   func (esm *EventStreamManager) PublishEvent(event Event) {
       esm.mu.RLock()
       defer esm.mu.RUnlock()
       
       // Buffer for visualization
       esm.buffer.Push(event)
       
       // Notify subscribers
       for _, ch := range esm.subscribers {
           select {
           case ch <- event:
           default:
               // Non-blocking send
           }
       }
   }
   ```

2. **WebSocket Integration for Real-Time Updates**
   ```go
   // backend/api/safety_cockpit.go
   func (h *Handler) HandleCockpitStream(w http.ResponseWriter, r *http.Request) {
       conn, err := h.upgrader.Upgrade(w, r, nil)
       if err != nil {
           log.Printf("WebSocket upgrade failed: %v", err)
           return
       }
       defer conn.Close()
       
       clientID := generateClientID()
       eventChan := make(chan Event, 100)
       
       h.eventManager.Subscribe(clientID, eventChan)
       defer h.eventManager.Unsubscribe(clientID)
       
       // Send historical data
       historicalEvents := h.eventManager.GetBufferedEvents(100)
       for _, event := range historicalEvents {
           conn.WriteJSON(event)
       }
       
       // Stream real-time events
       for event := range eventChan {
           if err := conn.WriteJSON(event); err != nil {
               log.Printf("WebSocket write error: %v", err)
               break
           }
       }
   }
   ```

3. **Create Aggregation Service**
   ```go
   type AggregationService struct {
       metrics    map[string]*MetricAggregator
       timeWindow time.Duration
       mu         sync.RWMutex
   }

   func (as *AggregationService) AggregateTraceData(trace *opik.Trace) {
       as.mu.Lock()
       defer as.mu.Unlock()
       
       // Aggregate by threat type
       threatType := trace.Metadata["threat_type"].(string)
       if aggregator, exists := as.metrics[threatType]; exists {
           aggregator.Add(trace)
       } else {
           as.metrics[threatType] = NewMetricAggregator(threatType)
           as.metrics[threatType].Add(trace)
       }
   }
   ```

#### Frontend Implementation
1. **Create WebSocket Service (`frontend/src/services/cockpitService.ts`)**
   ```typescript
   export class CockpitService {
       private ws: WebSocket | null = null;
       private eventHandlers: Map<string, EventHandler[]> = new Map();
       
       connect(url: string): Promise<void> {
           return new Promise((resolve, reject) => {
               this.ws = new WebSocket(url);
               
               this.ws.onopen = () => {
                   console.log('Connected to Safety Cockpit');
                   resolve();
               };
               
               this.ws.onmessage = (event) => {
                   const data = JSON.parse(event.data);
                   this.handleEvent(data);
               };
               
               this.ws.onerror = (error) => {
                   console.error('WebSocket error:', error);
                   reject(error);
               };
           });
       }
       
       private handleEvent(event: CockpitEvent) {
           const handlers = this.eventHandlers.get(event.type) || [];
           handlers.forEach(handler => handler(event));
       }
   }
   ```

#### Testing Cycle 21C
1. **Stream Performance Test**
   - Generate 1000 events/second
   - Verify no message loss
   - Check memory usage
   - Monitor latency

2. **WebSocket Stress Test**
   - Connect 100 clients
   - Stream events to all
   - Verify synchronization
   - Test reconnection logic

### Cycle 21D: 3D Visualization Implementation
**Duration:** 8-10 hours

#### Frontend Implementation
1. **Install Three.js Dependencies**
   ```bash
   npm install three @types/three @react-three/fiber @react-three/drei
   ```

2. **Create 3D Scene Component (`frontend/src/components/SafetyCockpit/ThreatLandscape.tsx`)**
   ```typescript
   import { Canvas } from '@react-three/fiber';
   import { OrbitControls, Stars } from '@react-three/drei';
   import { ThreatParticles } from './ThreatParticles';
   import { ProviderNodes } from './ProviderNodes';
   import { ThreatConnections } from './ThreatConnections';

   export const ThreatLandscape: React.FC = () => {
       const [threats, setThreats] = useState<Threat[]>([]);
       const [providers, setProviders] = useState<Provider[]>([]);
       
       return (
           <Canvas
               camera={{ position: [0, 0, 50], fov: 60 }}
               style={{ width: '100%', height: '600px' }}
           >
               <ambientLight intensity={0.5} />
               <pointLight position={[10, 10, 10]} />
               <Stars radius={100} depth={50} count={5000} factor={4} />
               
               <ThreatParticles threats={threats} />
               <ProviderNodes providers={providers} />
               <ThreatConnections threats={threats} providers={providers} />
               
               <OrbitControls enablePan={true} enableZoom={true} enableRotate={true} />
           </Canvas>
       );
   };
   ```

3. **Implement Threat Particles (`frontend/src/components/SafetyCockpit/ThreatParticles.tsx`)**
   ```typescript
   export const ThreatParticles: React.FC<{ threats: Threat[] }> = ({ threats }) => {
       const particlesRef = useRef<THREE.Points>(null);
       
       useFrame(() => {
           if (particlesRef.current) {
               particlesRef.current.rotation.y += 0.001;
               
               // Update particle colors based on threat severity
               const colors = particlesRef.current.geometry.attributes.color;
               threats.forEach((threat, i) => {
                   const color = getThreatColor(threat.severity);
                   colors.setXYZ(i, color.r, color.g, color.b);
               });
               colors.needsUpdate = true;
           }
       });
       
       const positions = useMemo(() => {
           const pos = new Float32Array(threats.length * 3);
           threats.forEach((threat, i) => {
               const theta = (threat.angle * Math.PI) / 180;
               const phi = (threat.elevation * Math.PI) / 180;
               const radius = threat.distance;
               
               pos[i * 3] = radius * Math.sin(phi) * Math.cos(theta);
               pos[i * 3 + 1] = radius * Math.cos(phi);
               pos[i * 3 + 2] = radius * Math.sin(phi) * Math.sin(theta);
           });
           return pos;
       }, [threats]);
       
       return (
           <points ref={particlesRef}>
               <bufferGeometry>
                   <bufferAttribute
                       attach="attributes-position"
                       count={positions.length / 3}
                       array={positions}
                       itemSize={3}
                   />
               </bufferGeometry>
               <pointsMaterial size={2} vertexColors />
           </points>
       );
   };
   ```

4. **Create Interactive Threat Details**
   ```typescript
   const ThreatInspector: React.FC = () => {
       const [selectedThreat, setSelectedThreat] = useState<Threat | null>(null);
       
       const handleThreatClick = useCallback(async (threat: Threat) => {
           setSelectedThreat(threat);
           
           // Fetch detailed trace from Opik
           const trace = await opikService.getTrace(threat.traceId);
           setTraceDetails(trace);
       }, []);
       
       return (
           <Box>
               {selectedThreat && (
                   <Paper elevation={3} sx={{ p: 2 }}>
                       <Typography variant="h6">Threat Details</Typography>
                       <Typography>Type: {selectedThreat.type}</Typography>
                       <Typography>Severity: {selectedThreat.severity}</Typography>
                       <Typography>Time: {selectedThreat.timestamp}</Typography>
                       
                       {traceDetails && (
                           <TraceViewer trace={traceDetails} />
                       )}
                   </Paper>
               )}
           </Box>
       );
   };
   ```

#### Testing Cycle 21D
1. **3D Rendering Performance**
   - Render 10,000 threat particles
   - Maintain 60 FPS
   - Test on various devices
   - Optimize particle systems

2. **Interaction Testing**
   - Click detection accuracy
   - Camera controls responsiveness
   - Touch device support
   - Zoom/pan limits

### Cycle 21E: Custom Opik Evaluators
**Duration:** 6-8 hours

#### Backend Implementation
1. **Create Base Evaluator (`backend/opik/evaluators/base.go`)**
   ```go
   type Evaluator interface {
       Name() string
       Evaluate(trace *opik.Trace) (float64, map[string]interface{})
   }

   type BaseEvaluator struct {
       name      string
       threshold float64
   }
   ```

2. **Implement Regex Pattern Effectiveness Evaluator**
   ```go
   type RegexEffectivenessEvaluator struct {
       BaseEvaluator
   }

   func (e *RegexEffectivenessEvaluator) Evaluate(trace *opik.Trace) (float64, map[string]interface{}) {
       regexSpan := trace.GetSpan("regex_moderation")
       if regexSpan == nil {
           return 0, map[string]interface{}{"error": "no regex span found"}
       }
       
       output := regexSpan.Output.(map[string]interface{})
       blocked := output["blocked"].(bool)
       confidence := output["confidence"].(float64)
       
       // Calculate effectiveness based on detection and confidence
       score := 0.0
       if blocked {
           score = confidence
       } else {
           // Check if it should have been blocked (false negative)
           if trace.Metadata["known_threat"].(bool) {
               score = 0.0
           } else {
               score = 1.0
           }
       }
       
       return score, map[string]interface{}{
           "blocked":    blocked,
           "confidence": confidence,
           "correct":    score > 0.5,
       }
   }
   ```

3. **Implement PII Detection Coverage Evaluator**
   ```go
   type PIICoverageEvaluator struct {
       BaseEvaluator
       knownPIIPatterns map[string][]string
   }

   func (e *PIICoverageEvaluator) Evaluate(trace *opik.Trace) (float64, map[string]interface{}) {
       piiSpan := trace.GetSpan("pii_detection")
       if piiSpan == nil {
           return 0, nil
       }
       
       input := piiSpan.Input.(map[string]interface{})["content"].(string)
       output := piiSpan.Output.(map[string]interface{})
       detectedTypes := output["pii_types"].([]string)
       
       // Check coverage against known PII
       expectedPII := e.extractExpectedPII(input)
       detected := len(detectedTypes)
       expected := len(expectedPII)
       
       coverage := 0.0
       if expected > 0 {
           coverage = float64(detected) / float64(expected)
       }
       
       return coverage, map[string]interface{}{
           "detected_count": detected,
           "expected_count": expected,
           "missed_pii":     e.getMissedPII(expectedPII, detectedTypes),
       }
   }
   ```

4. **Register Evaluators with Opik**
   ```go
   func RegisterEvaluators(client *OpikClient) error {
       evaluators := []Evaluator{
           &RegexEffectivenessEvaluator{BaseEvaluator{"regex_effectiveness", 0.8}},
           &LLMAccuracyEvaluator{BaseEvaluator{"llm_accuracy", 0.9}},
           &PIICoverageEvaluator{BaseEvaluator{"pii_coverage", 0.95}, loadPIIPatterns()},
           &FalsePositiveEvaluator{BaseEvaluator{"false_positive_rate", 0.1}},
           &ResponseTimeEvaluator{BaseEvaluator{"response_time", 100}},
       }
       
       for _, evaluator := range evaluators {
           client.RegisterEvaluator(evaluator)
       }
       
       return nil
   }
   ```

#### Testing Cycle 21E
1. **Evaluator Unit Tests**
   ```go
   func TestRegexEffectivenessEvaluator(t *testing.T) {
       evaluator := &RegexEffectivenessEvaluator{}
       
       // Test true positive
       trace := createTestTrace(true, true, 0.95)
       score, details := evaluator.Evaluate(trace)
       assert.Equal(t, 0.95, score)
       
       // Test false negative
       trace = createTestTrace(false, true, 0.0)
       score, _ = evaluator.Evaluate(trace)
       assert.Equal(t, 0.0, score)
   }
   ```

2. **Integration Testing**
   - Run evaluators on real traces
   - Verify scoring accuracy
   - Check performance impact
   - Test edge cases

### Cycle 21F: Dashboard Integration & Polish
**Duration:** 6-8 hours

#### Frontend Implementation
1. **Create Main Dashboard Component (`frontend/src/components/SafetyCockpit/Dashboard.tsx`)**
   ```typescript
   export const SafetyCockpitDashboard: React.FC = () => {
       const [view, setView] = useState<'3d' | 'metrics' | 'timeline'>('3d');
       const [timeRange, setTimeRange] = useState<TimeRange>('1h');
       const [filters, setFilters] = useState<Filters>({});
       
       return (
           <Box sx={{ height: '100vh', display: 'flex', flexDirection: 'column' }}>
               <AppBar position="static">
                   <Toolbar>
                       <Typography variant="h6" sx={{ flexGrow: 1 }}>
                           AI Safety Cockpit
                       </Typography>
                       <ToggleButtonGroup value={view} onChange={(e, v) => setView(v)}>
                           <ToggleButton value="3d">3D View</ToggleButton>
                           <ToggleButton value="metrics">Metrics</ToggleButton>
                           <ToggleButton value="timeline">Timeline</ToggleButton>
                       </ToggleButtonGroup>
                   </Toolbar>
               </AppBar>
               
               <Box sx={{ flexGrow: 1, position: 'relative' }}>
                   {view === '3d' && <ThreatLandscape />}
                   {view === 'metrics' && <MetricsDashboard />}
                   {view === 'timeline' && <ThreatTimeline />}
                   
                   <FloatingControls
                       timeRange={timeRange}
                       onTimeRangeChange={setTimeRange}
                       filters={filters}
                       onFiltersChange={setFilters}
                   />
               </Box>
               
               <StatusBar />
           </Box>
       );
   };
   ```

2. **Implement Metrics Dashboard**
   ```typescript
   const MetricsDashboard: React.FC = () => {
       const metrics = useMetrics();
       
       return (
           <Grid container spacing={2} sx={{ p: 2 }}>
               <Grid item xs={12} md={6}>
                   <MetricCard
                       title="Threat Detection Rate"
                       value={metrics.detectionRate}
                       format="percentage"
                       trend={metrics.detectionTrend}
                       sparkline={metrics.detectionHistory}
                   />
               </Grid>
               
               <Grid item xs={12} md={6}>
                   <MetricCard
                       title="False Positive Rate"
                       value={metrics.falsePositiveRate}
                       format="percentage"
                       trend={metrics.fpTrend}
                       target={0.05}
                   />
               </Grid>
               
               <Grid item xs={12}>
                   <ProviderComparisonChart providers={metrics.providers} />
               </Grid>
               
               <Grid item xs={12} md={6}>
                   <ThreatTypeDistribution data={metrics.threatTypes} />
               </Grid>
               
               <Grid item xs={12} md={6}>
                   <GeographicHeatmap data={metrics.geoData} />
               </Grid>
           </Grid>
       );
   };
   ```

3. **Add Export Functionality**
   ```typescript
   const ExportManager: React.FC = () => {
       const handleExport = async (format: 'pdf' | 'csv' | 'json') => {
           const data = await cockpitService.exportReport({
               format,
               timeRange: selectedTimeRange,
               includeTraces: true,
               includeMetrics: true,
           });
           
           downloadFile(data, `safety-report-${Date.now()}.${format}`);
       };
       
       return (
           <Menu>
               <MenuItem onClick={() => handleExport('pdf')}>
                   Export as PDF
               </MenuItem>
               <MenuItem onClick={() => handleExport('csv')}>
                   Export as CSV
               </MenuItem>
               <MenuItem onClick={() => handleExport('json')}>
                   Export as JSON
               </MenuItem>
           </Menu>
       );
   };
   ```

#### Testing Cycle 21F
1. **UI/UX Testing**
   - Test all view modes
   - Verify responsive design
   - Check accessibility
   - Test export functions

2. **End-to-End Testing**
   - Generate test threats
   - Verify real-time updates
   - Test filtering
   - Validate metrics accuracy

### Cycle 21G: Performance Optimization & Deployment
**Duration:** 4-6 hours

#### Optimization Implementation
1. **Backend Optimizations**
   ```go
   // Implement trace batching
   type TraceBatcher struct {
       traces    []opik.Trace
       batchSize int
       ticker    *time.Ticker
       mu        sync.Mutex
   }

   func (tb *TraceBatcher) Add(trace opik.Trace) {
       tb.mu.Lock()
       tb.traces = append(tb.traces, trace)
       
       if len(tb.traces) >= tb.batchSize {
           tb.flush()
       }
       tb.mu.Unlock()
   }
   ```

2. **Frontend Optimizations**
   ```typescript
   // Implement virtual scrolling for large datasets
   import { FixedSizeList } from 'react-window';
   
   // Use React.memo for expensive components
   export const ThreatParticles = React.memo(({ threats }) => {
       // ... component implementation
   }, (prevProps, nextProps) => {
       return prevProps.threats.length === nextProps.threats.length;
   });
   
   // Implement data decimation for visualization
   const decimateData = (data: Point[], maxPoints: number): Point[] => {
       if (data.length <= maxPoints) return data;
       
       const step = Math.ceil(data.length / maxPoints);
       return data.filter((_, index) => index % step === 0);
   };
   ```

3. **Add Performance Monitoring**
   ```go
   func (h *Handler) PerformanceMiddleware(next http.HandlerFunc) http.HandlerFunc {
       return func(w http.ResponseWriter, r *http.Request) {
           start := time.Now()
           
           // Track in Opik
           span := h.opikClient.StartHTTPSpan(r.Context(), r.URL.Path)
           defer span.End()
           
           next(w, r)
           
           duration := time.Since(start)
           span.SetMetadata(map[string]interface{}{
               "duration_ms": duration.Milliseconds(),
               "method":      r.Method,
               "status":      w.Header().Get("Status"),
           })
       }
   }
   ```

#### Deployment Configuration
1. **Docker Configuration**
   ```dockerfile
   # Add Opik configuration
   ENV OPIK_API_KEY=""
   ENV OPIK_PROJECT_NAME="qt1-safety-cockpit"
   ```

2. **Kubernetes Deployment**
   ```yaml
   apiVersion: v1
   kind: ConfigMap
   metadata:
     name: opik-config
   data:
     project_name: "qt1-safety-cockpit"
   ---
   apiVersion: v1
   kind: Secret
   metadata:
     name: opik-secret
   stringData:
     api_key: "your-opik-api-key"
   ```

#### Final Testing
1. **Load Testing**
   - 10,000 concurrent traces
   - Verify dashboard responsiveness
   - Check memory usage
   - Monitor CPU utilization

2. **Acceptance Testing**
   - All features working
   - Real-time updates < 100ms
   - 3D visualization smooth
   - Export functionality working

## Summary

### Deliverables
1. **Opik Integration**
   - Complete SDK integration
   - Custom trace collection
   - Real-time event streaming

2. **3D Visualization**
   - Interactive threat landscape
   - Real-time particle system
   - Provider performance nodes

3. **Custom Evaluators**
   - 6 custom Opik evaluators
   - Automated scoring system
   - Performance tracking

4. **Dashboard Features**
   - Multiple view modes
   - Real-time metrics
   - Export functionality
   - Historical analysis

### Performance Metrics
- Real-time updates: < 100ms latency
- 3D rendering: 60 FPS with 10,000 particles
- Trace processing: 1,000+ traces/second
- Dashboard load time: < 2 seconds

### Security & Compliance
- Secure WebSocket connections
- API key encryption
- Audit logging
- Data retention policies

### Documentation
- API documentation
- User guide
- Deployment instructions
- Troubleshooting guide