package moderation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"qt1-middleware/config"
	"qt1-middleware/opik"
	"sync"
	"time"

	"github.com/sashabaranov/go-openai"
)

// EngineWithOpik represents the moderation engine with Opik integration
type EngineWithOpik struct {
	layers       []ModerationLayer
	config       config.AdvancedModerationConfig
	cache        *ModerationCache
	opikClient   *opik.OpikClient
	openaiClient *openai.Client
	mu           sync.RWMutex
	stats        *ModerationStats
}

// NewEngineWithOpik creates a new moderation engine with Opik support
func NewEngineWithOpik(cfg config.AdvancedModerationConfig, opikClient *opik.OpikClient, openaiClient *openai.Client) (*EngineWithOpik, error) {
	engine := &EngineWithOpik{
		config:       cfg,
		layers:       make([]ModerationLayer, 0),
		opikClient:   opikClient,
		openaiClient: openaiClient,
		stats: &ModerationStats{
			LayerStats: make(map[string]LayerStats),
		},
	}

	// Initialize cache if enabled
	if cfg.Cache.Enabled {
		// Convert config.CacheConfig to moderation.CacheConfig
		cacheConfig := CacheConfig{
			Enabled:    cfg.Cache.Enabled,
			TTLMinutes: cfg.Cache.TTLMinutes,
			MaxEntries: cfg.Cache.MaxEntries,
		}
		engine.cache = NewModerationCache(cacheConfig)
	}

	// Initialize layers
	if err := engine.initializeLayers(); err != nil {
		return nil, err
	}

	return engine, nil
}

// initializeLayers initializes all configured moderation layers
func (e *EngineWithOpik) initializeLayers() error {
	for _, layerConfig := range e.config.Layers {
		if !layerConfig.Enabled {
			continue
		}

		// Convert config.LayerConfig to moderation.LayerConfig
		modLayerConfig := LayerConfig{
			Name:      layerConfig.Name,
			Enabled:   layerConfig.Enabled,
			Weight:    layerConfig.Weight,
			Threshold: layerConfig.Threshold,
			Options:   layerConfig.Options,
		}

		layer, err := e.createLayer(modLayerConfig)
		if err != nil {
			return fmt.Errorf("failed to create layer %s: %w", layerConfig.Name, err)
		}

		e.layers = append(e.layers, layer)
	}

	if len(e.layers) == 0 {
		return fmt.Errorf("no moderation layers enabled")
	}

	return nil
}

// createLayer creates a moderation layer based on configuration
func (e *EngineWithOpik) createLayer(cfg LayerConfig) (ModerationLayer, error) {
	switch cfg.Name {
	case "regex":
		return NewRegexModerator(cfg, e.opikClient)
	case "openai", "llm":
		if e.openaiClient == nil {
			return nil, fmt.Errorf("OpenAI client not configured")
		}
		return NewLLMModerator(cfg, e.opikClient, e.openaiClient)
	case "pii":
		return NewPIIDetector(cfg, e.opikClient)
	default:
		return nil, fmt.Errorf("unknown layer type: %s", cfg.Name)
	}
}

// ModerateWithOpik performs moderation using all enabled layers with Opik tracing
func (e *EngineWithOpik) ModerateWithOpik(ctx context.Context, content string, moderationCtx ModerationContext) (*AggregatedResult, error) {
	startTime := time.Now()

	// Check cache first
	if e.cache != nil {
		if cached, found := e.cache.Get(content, moderationCtx); found {
			e.updateStats(true, time.Since(startTime))
			cached.CacheHit = true
			return cached, nil
		}
	}

	// Create Opik trace if enabled
	var trace *opik.Trace
	if e.opikClient != nil && e.config.Enabled {
		fmt.Printf("DEBUG: Creating Opik trace - client not nil: %t, config enabled: %t\n", e.opikClient != nil, e.config.Enabled)
		
		endpoint := ""
		if ep, ok := moderationCtx.Metadata["endpoint"].(string); ok {
			endpoint = ep
		}

		request := opik.ModerationRequest{
			ID:        moderationCtx.RequestID,
			Content:   content,
			UserID:    moderationCtx.UserID,
			SessionID: moderationCtx.SessionID,
			IPAddress: moderationCtx.IPAddress,
			Provider:  "qt1-middleware",
			Endpoint:  endpoint,
			Timestamp: moderationCtx.Timestamp,
		}
		
		var err error
		trace, err = e.opikClient.TraceModeration(ctx, request)
		if err != nil {
			// Log error but continue without tracing
			fmt.Printf("Failed to create Opik trace: %v\n", err)
		} else {
			fmt.Printf("DEBUG: Successfully created Opik trace with ID: %s\n", trace.ID)
		}
	} else {
		fmt.Printf("DEBUG: Skipping Opik trace creation - client nil: %t, config enabled: %t\n", e.opikClient == nil, e.config.Enabled)
	}

	// Run all layers
	layerResults := e.runLayersWithTracing(ctx, content, moderationCtx, trace)

	// Aggregate results
	aggregated := e.aggregateResults(layerResults, moderationCtx)
	aggregated.ProcessTime = time.Since(startTime)

	// End trace
	if trace != nil {
		fmt.Printf("DEBUG: Ending Opik trace %s with final_score: %.3f, allowed: %t\n", trace.ID, aggregated.FinalScore, !aggregated.FinalDecision)
		e.opikClient.EndTrace(trace, map[string]interface{}{
			"allowed":        !aggregated.FinalDecision,
			"final_score":    aggregated.FinalScore,
			"severity":       aggregated.Severity,
			"action":         aggregated.Action,
			"layers_checked": len(layerResults),
		})
		fmt.Printf("DEBUG: Trace %s ended successfully\n", trace.ID)
	} else {
		fmt.Printf("DEBUG: No trace to end\n")
	}

	// Update cache
	if e.cache != nil && !aggregated.FinalDecision {
		e.cache.Set(content, moderationCtx, *aggregated)
	}

	// Update statistics
	e.updateStats(false, aggregated.ProcessTime)
	e.updateLayerStats(layerResults)

	return aggregated, nil
}

// runLayersWithTracing runs all moderation layers with Opik tracing
func (e *EngineWithOpik) runLayersWithTracing(ctx context.Context, content string, moderationCtx ModerationContext, trace *opik.Trace) []ModerationResult {
	e.mu.RLock()
	layers := e.layers
	e.mu.RUnlock()

	results := make([]ModerationResult, 0, len(layers))
	
	// Use wait group for parallel execution
	var wg sync.WaitGroup
	resultsChan := make(chan ModerationResult, len(layers))

	for _, layer := range layers {
		if !layer.Enabled() {
			continue
		}

		wg.Add(1)
		go func(l ModerationLayer) {
			defer wg.Done()

			// Create span for this layer if tracing
			var span *opik.Span
			if trace != nil {
				spanType := e.getSpanType(l.Name())
				span = e.opikClient.StartSpan(trace, spanType, map[string]interface{}{
					"content": content,
				})
			}

			// Run moderation with or without tracing
			var result ModerationResult
			if span != nil {
				// Use tracing-aware methods if available
				switch mod := l.(type) {
				case *RegexModerator:
					if res, err := mod.ModerateWithTracing(ctx, content, span); err == nil {
						result = ModerationResult{
							Score:       res.Confidence,
							Confidence:  res.Confidence,
							Blocked:     res.Blocked,
							Reason:      res.Reason,
							Category:    CategoryToxicity,
							LayerName:   l.Name(),
							Details:     res.Metadata,
							ProcessTime: res.ProcessingTime,
						}
					}
				case *LLMModerator:
					if res, err := mod.ModerateWithTracing(ctx, content, span); err == nil {
						result = ModerationResult{
							Score:       res.Confidence,
							Confidence:  res.Confidence,
							Blocked:     res.Blocked,
							Reason:      res.Reason,
							Category:    CategoryToxicity,
							LayerName:   l.Name(),
							Details:     res.Metadata,
							ProcessTime: res.ProcessingTime,
						}
					}
				case *PIIDetector:
					if res, err := mod.DetectWithTracing(ctx, content, span); err == nil {
						score := 0.0
						if res.PIIFound {
							score = res.Confidence
						}
						result = ModerationResult{
							Score:       score,
							Confidence:  res.Confidence,
							Blocked:     res.PIIFound && score > 0.6,
							Reason:      fmt.Sprintf("Found %d PII items", res.PIICount),
							Category:    CategoryPII,
							LayerName:   l.Name(),
							Details:     map[string]interface{}{"pii_result": res},
							ProcessTime: time.Since(res.ProcessedAt),
						}
					}
				default:
					// Fallback to non-tracing method
					result = l.Moderate(content, moderationCtx)
				}
			} else {
				// No tracing
				result = l.Moderate(content, moderationCtx)
			}

			resultsChan <- result
		}(layer)
	}

	// Wait for all layers to complete
	wg.Wait()
	close(resultsChan)

	// Collect results
	for result := range resultsChan {
		results = append(results, result)
	}

	return results
}

// getSpanType maps layer name to Opik span type
func (e *EngineWithOpik) getSpanType(layerName string) opik.SpanType {
	switch layerName {
	case "regex":
		return opik.SpanTypeRegex
	case "llm", "openai":
		return opik.SpanTypeLLM
	case "pii":
		return opik.SpanTypePII
	default:
		return opik.SpanTypeSecurity
	}
}

// aggregateResults aggregates results from all layers
func (e *EngineWithOpik) aggregateResults(layerResults []ModerationResult, context ModerationContext) *AggregatedResult {
	// Calculate weighted score
	totalScore := 0.0
	totalWeight := 0.0

	for _, result := range layerResults {
		layer := e.getLayerByName(result.LayerName)
		if layer != nil {
			weight := layer.Weight()
			totalScore += result.Score * weight
			totalWeight += weight
		}
	}

	finalScore := 0.0
	if totalWeight > 0 {
		finalScore = totalScore / totalWeight
	}

	// Determine severity and action
	severity := e.determineSeverity(finalScore)
	action := e.determineAction(severity)
	shouldBlock := action == ActionBlock || action == ActionBlockAlert

	return &AggregatedResult{
		FinalScore:     finalScore,
		FinalDecision:  shouldBlock,
		Action:         action,
		Severity:       severity,
		LayerResults:   layerResults,
		CacheHit:       false,
		RequestContext: context,
	}
}

// getLayerByName retrieves a layer by name
func (e *EngineWithOpik) getLayerByName(name string) ModerationLayer {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, layer := range e.layers {
		if layer.Name() == name {
			return layer
		}
	}
	return nil
}

// determineSeverity determines severity based on score
func (e *EngineWithOpik) determineSeverity(score float64) string {
	if score >= e.config.Thresholds.Critical {
		return SeverityCritical
	} else if score >= e.config.Thresholds.High {
		return SeverityHigh
	} else if score >= e.config.Thresholds.Medium {
		return SeverityMedium
	}
	return SeverityLow
}

// determineAction determines action based on severity
func (e *EngineWithOpik) determineAction(severity string) string {
	switch severity {
	case SeverityCritical:
		return e.config.Actions.Critical
	case SeverityHigh:
		return e.config.Actions.High
	case SeverityMedium:
		return e.config.Actions.Medium
	default:
		return e.config.Actions.Low
	}
}

// generateCacheKey generates a cache key for content
func (e *EngineWithOpik) generateCacheKey(content string, ctx ModerationContext) string {
	h := sha256.New()
	h.Write([]byte(content))
	h.Write([]byte(ctx.UserID))
	h.Write([]byte(ctx.UserType))
	return hex.EncodeToString(h.Sum(nil))
}

// updateStats updates engine statistics
func (e *EngineWithOpik) updateStats(cacheHit bool, processTime time.Duration) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.stats.TotalRequests++
	if cacheHit {
		e.stats.CacheHits++
	}

	// Update average process time
	currentAvg := e.stats.AverageProcessTime
	e.stats.AverageProcessTime = (currentAvg*time.Duration(e.stats.TotalRequests-1) + processTime) / time.Duration(e.stats.TotalRequests)
}

// updateLayerStats updates per-layer statistics
func (e *EngineWithOpik) updateLayerStats(results []ModerationResult) {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, result := range results {
		stats, exists := e.stats.LayerStats[result.LayerName]
		if !exists {
			stats = LayerStats{LayerName: result.LayerName}
		}

		stats.TotalProcessed++
		if result.Blocked {
			stats.Detections++
		}

		// Update average score
		stats.AverageScore = (stats.AverageScore*float64(stats.TotalProcessed-1) + result.Score) / float64(stats.TotalProcessed)
		
		// Update average process time
		stats.AverageProcessTime = (stats.AverageProcessTime*time.Duration(stats.TotalProcessed-1) + result.ProcessTime) / time.Duration(stats.TotalProcessed)

		e.stats.LayerStats[result.LayerName] = stats
	}

	// Update blocked/flagged counts
	for _, result := range results {
		if result.Blocked {
			e.stats.BlockedRequests++
			break
		} else if result.Score > e.config.Thresholds.Medium {
			e.stats.FlaggedRequests++
			break
		}
	}
}

// GetStats returns current statistics
func (e *EngineWithOpik) GetStats() ModerationStats {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Create a copy to avoid race conditions
	statsCopy := *e.stats
	statsCopy.LayerStats = make(map[string]LayerStats)
	for k, v := range e.stats.LayerStats {
		statsCopy.LayerStats[k] = v
	}

	return statsCopy
}

// GetLayerNames returns the names of all layers
func (e *EngineWithOpik) GetLayerNames() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	names := make([]string, len(e.layers))
	for i, layer := range e.layers {
		names[i] = layer.Name()
	}
	return names
}

// GetEnabledLayers returns all enabled layers
func (e *EngineWithOpik) GetEnabledLayers() []ModerationLayer {
	e.mu.RLock()
	defer e.mu.RUnlock()

	enabled := make([]ModerationLayer, 0, len(e.layers))
	for _, layer := range e.layers {
		if layer.Enabled() {
			enabled = append(enabled, layer)
		}
	}
	return enabled
}

// GetLayerInfo returns information about a specific layer
func (e *EngineWithOpik) GetLayerInfo(layerName string) (weight float64, enabled bool, found bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, layer := range e.layers {
		if layer.Name() == layerName {
			return layer.Weight(), layer.Enabled(), true
		}
	}
	return 0, false, false
}

// IsEnabled returns whether the engine is enabled
func (e *EngineWithOpik) IsEnabled() bool {
	return e.config.Enabled
}

// GetConfig returns the current configuration
func (e *EngineWithOpik) GetConfig() *AdvancedModerationConfig {
	// Convert from config.AdvancedModerationConfig to moderation.AdvancedModerationConfig
	return &AdvancedModerationConfig{
		Enabled: e.config.Enabled,
		Layers: convertFromAppLayerConfigs(e.config.Layers),
		Thresholds: ModerationThresholds{
			Low:      e.config.Thresholds.Low,
			Medium:   e.config.Thresholds.Medium,
			High:     e.config.Thresholds.High,
			Critical: e.config.Thresholds.Critical,
		},
		Actions: ActionConfig{
			Low:      e.config.Actions.Low,
			Medium:   e.config.Actions.Medium,
			High:     e.config.Actions.High,
			Critical: e.config.Actions.Critical,
		},
		Cache: CacheConfig{
			Enabled:    e.config.Cache.Enabled,
			TTLMinutes: e.config.Cache.TTLMinutes,
			MaxEntries: e.config.Cache.MaxEntries,
		},
		Analytics: AnalyticsConfig{
			Enabled:           e.config.Analytics.Enabled,
			CollectDetails:    e.config.Analytics.CollectDetails,
			RetentionDays:     e.config.Analytics.RetentionDays,
			EnablePerformance: e.config.Analytics.EnablePerformance,
		},
	}
}

// RegisterLayer adds a new layer to the engine
func (e *EngineWithOpik) RegisterLayer(layer ModerationLayer) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.layers = append(e.layers, layer)
	return nil
}

// SetLayers replaces all layers
func (e *EngineWithOpik) SetLayers(layers []ModerationLayer) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.layers = layers
}

// ReloadConfig reloads the configuration
func (e *EngineWithOpik) ReloadConfig() error {
	// For now, just return nil as reloading isn't implemented
	return nil
}

// convertFromAppLayerConfigs converts app config layer configs to moderation layer configs
func convertFromAppLayerConfigs(appLayers []config.AdvancedLayerConfig) []LayerConfig {
	layers := make([]LayerConfig, len(appLayers))
	for i, layer := range appLayers {
		layers[i] = LayerConfig{
			Name:      layer.Name,
			Enabled:   layer.Enabled,
			Weight:    layer.Weight,
			Threshold: layer.Threshold,
			Options:   layer.Options,
		}
	}
	return layers
}

// UpdateLayerConfig updates configuration for a specific layer
func (e *EngineWithOpik) UpdateLayerConfig(layerName string, config LayerConfig) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Find and update the layer
	for i, layer := range e.layers {
		if layer.Name() == layerName {
			// Create new layer with updated config
			newLayer, err := e.createLayer(config)
			if err != nil {
				return err
			}
			e.layers[i] = newLayer
			return nil
		}
	}

	return fmt.Errorf("layer %s not found", layerName)
}

// Close shuts down the engine
func (e *EngineWithOpik) Close() {
	if e.cache != nil {
		e.cache.Close()
	}
}