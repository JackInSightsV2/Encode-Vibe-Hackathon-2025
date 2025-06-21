package moderation

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// ModerationEngine orchestrates the moderation pipeline
type ModerationEngine struct {
	layers        []ModerationLayer
	cache         *ModerationCache
	config        *AdvancedModerationConfig
	configManager *ConfigManager
	stats         *ModerationStats
	mutex         sync.RWMutex
}

// NewModerationEngine creates a new moderation engine
func NewModerationEngine(configPath string) (*ModerationEngine, error) {
	// Initialize configuration manager
	configManager := NewConfigManager(configPath)
	config, err := configManager.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %v", err)
	}
	
	// Initialize cache
	cache := NewModerationCache(config.Cache)
	
	// Initialize stats
	stats := &ModerationStats{
		LayerStats: make(map[string]LayerStats),
	}
	
	engine := &ModerationEngine{
		layers:        make([]ModerationLayer, 0),
		cache:         cache,
		config:        config,
		configManager: configManager,
		stats:         stats,
	}
	
	return engine, nil
}

// NewModerationEngineWithConfig creates a new moderation engine with explicit parameters for testing
func NewModerationEngineWithConfig(layers []ModerationLayer, cache *ModerationCache, config *AdvancedModerationConfig, configManager *ConfigManager) *ModerationEngine {
	stats := &ModerationStats{
		LayerStats: make(map[string]LayerStats),
	}
	
	engine := &ModerationEngine{
		layers:        layers,
		cache:         cache,
		config:        config,
		configManager: configManager,
		stats:         stats,
	}
	
	return engine
}

// RegisterLayer adds a moderation layer to the pipeline
func (me *ModerationEngine) RegisterLayer(layer ModerationLayer) error {
	me.mutex.Lock()
	defer me.mutex.Unlock()
	
	// Check if layer with this name already exists
	for _, existingLayer := range me.layers {
		if existingLayer.Name() == layer.Name() {
			return fmt.Errorf("layer with name '%s' already registered", layer.Name())
		}
	}
	
	me.layers = append(me.layers, layer)
	
	// Initialize stats for this layer
	if me.stats.LayerStats == nil {
		me.stats.LayerStats = make(map[string]LayerStats)
	}
	me.stats.LayerStats[layer.Name()] = LayerStats{
		LayerName: layer.Name(),
	}
	
	log.Printf("Registered moderation layer: %s (weight: %.2f)", layer.Name(), layer.Weight())
	return nil
}

// SetLayers sets the layers for the moderation engine
func (me *ModerationEngine) SetLayers(layers []ModerationLayer) {
	me.mutex.Lock()
	defer me.mutex.Unlock()
	me.layers = layers
	
	// Initialize stats for all layers
	if me.stats.LayerStats == nil {
		me.stats.LayerStats = make(map[string]LayerStats)
	}
	for _, layer := range layers {
		me.stats.LayerStats[layer.Name()] = LayerStats{
			LayerName: layer.Name(),
		}
	}
}

// Moderate processes content through the moderation pipeline
func (me *ModerationEngine) Moderate(content string, context ModerationContext) (*AggregatedResult, error) {
	startTime := time.Now()
	
	// Check if moderation is enabled
	if !me.config.Enabled {
		return &AggregatedResult{
			FinalScore:     0.0,
			FinalDecision:  false,
			Action:         ActionLog,
			Severity:       SeverityLow,
			LayerResults:   []ModerationResult{},
			ProcessTime:    time.Since(startTime),
			CacheHit:       false,
			RequestContext: context,
		}, nil
	}
	
	// Check cache first
	if cachedResult, found := me.cache.Get(content, context); found {
		me.incrementTotalRequests()
		return cachedResult, nil
	}
	
	// Process through all enabled layers
	layerResults := make([]ModerationResult, 0)
	
	me.mutex.RLock()
	layers := make([]ModerationLayer, len(me.layers))
	copy(layers, me.layers)
	me.mutex.RUnlock()
	
	for _, layer := range layers {
		// Skip disabled layers
		if !layer.Enabled() {
			continue
		}
		
		// Process through this layer
		layerStartTime := time.Now()
		result := layer.Moderate(content, context)
		layerProcessTime := time.Since(layerStartTime)
		
		// Set layer-specific metadata
		result.LayerName = layer.Name()
		result.ProcessTime = layerProcessTime
		
		layerResults = append(layerResults, result)
		
		// Update layer stats
		me.updateLayerStats(layer.Name(), result, layerProcessTime)
	}
	
	// Aggregate results
	aggregatedResult := me.aggregateResults(layerResults, context, time.Since(startTime))
	
	// Cache the result
	me.cache.Set(content, context, *aggregatedResult)
	
	// Update global stats
	me.incrementTotalRequests()
	if aggregatedResult.FinalDecision {
		me.incrementBlockedRequests()
	}
	if aggregatedResult.Action == ActionFlag {
		me.incrementFlaggedRequests()
	}
	
	return aggregatedResult, nil
}

// aggregateResults combines layer results into a final decision
func (me *ModerationEngine) aggregateResults(layerResults []ModerationResult, context ModerationContext, totalTime time.Duration) *AggregatedResult {
	if len(layerResults) == 0 {
		return &AggregatedResult{
			FinalScore:     0.0,
			FinalDecision:  false,
			Action:         ActionLog,
			Severity:       SeverityLow,
			LayerResults:   layerResults,
			ProcessTime:    totalTime,
			CacheHit:       false,
			RequestContext: context,
		}
	}
	
	// Calculate weighted score
	totalScore := 0.0
	totalWeight := 0.0
	highestSingleScore := 0.0
	
	for _, result := range layerResults {
		// Find the layer configuration to get weight
		layerWeight := me.getLayerWeight(result.LayerName)
		
		totalScore += result.Score * layerWeight
		totalWeight += layerWeight
		
		if result.Score > highestSingleScore {
			highestSingleScore = result.Score
		}
	}
	
	// Normalize score if we have weights
	finalScore := 0.0
	if totalWeight > 0 {
		finalScore = totalScore / totalWeight
	}
	
	// Determine action and severity based on score
	action, severity := me.determineAction(finalScore)
	
	// Final decision: block if action is block or block_and_alert
	finalDecision := action == ActionBlock || action == ActionBlockAlert
	
	return &AggregatedResult{
		FinalScore:     finalScore,
		FinalDecision:  finalDecision,
		Action:         action,
		Severity:       severity,
		LayerResults:   layerResults,
		ProcessTime:    totalTime,
		CacheHit:       false,
		RequestContext: context,
	}
}

// getLayerWeight returns the configured weight for a layer
func (me *ModerationEngine) getLayerWeight(layerName string) float64 {
	// First check registered layers (direct access to layer weight)
	me.mutex.RLock()
	defer me.mutex.RUnlock()
	
	for _, layer := range me.layers {
		if layer.Name() == layerName {
			return layer.Weight()
		}
	}
	
	// Fallback to config layers
	for _, layerConfig := range me.config.Layers {
		if layerConfig.Name == layerName {
			return layerConfig.Weight
		}
	}
	
	return 0.0 // Default weight if not found
}

// determineAction determines the action and severity based on the final score
func (me *ModerationEngine) determineAction(score float64) (string, string) {
	thresholds := me.config.Thresholds
	actions := me.config.Actions
	
	if score >= thresholds.Critical {
		return actions.Critical, SeverityCritical
	} else if score >= thresholds.High {
		return actions.High, SeverityHigh
	} else if score >= thresholds.Medium {
		return actions.Medium, SeverityMedium
	} else if score >= thresholds.Low {
		return actions.Low, SeverityLow
	}
	
	return ActionLog, SeverityLow
}

// updateLayerStats updates statistics for a specific layer
func (me *ModerationEngine) updateLayerStats(layerName string, result ModerationResult, processTime time.Duration) {
	me.mutex.Lock()
	defer me.mutex.Unlock()
	
	stats := me.stats.LayerStats[layerName]
	stats.TotalProcessed++
	
	if result.Blocked {
		stats.Detections++
	}
	
	// Update average score (simple moving average)
	if stats.TotalProcessed == 1 {
		stats.AverageScore = result.Score
	} else {
		stats.AverageScore = (stats.AverageScore*float64(stats.TotalProcessed-1) + result.Score) / float64(stats.TotalProcessed)
	}
	
	// Update average process time
	if stats.TotalProcessed == 1 {
		stats.AverageProcessTime = processTime
	} else {
		avgNanos := (stats.AverageProcessTime.Nanoseconds()*int64(stats.TotalProcessed-1) + processTime.Nanoseconds()) / int64(stats.TotalProcessed)
		stats.AverageProcessTime = time.Duration(avgNanos)
	}
	
	me.stats.LayerStats[layerName] = stats
}

// GetStats returns current moderation statistics
func (me *ModerationEngine) GetStats() ModerationStats {
	me.mutex.RLock()
	defer me.mutex.RUnlock()
	
	stats := *me.stats
	
	// Add cache stats
	cacheStats := me.cache.GetStats()
	stats.CacheHits = cacheStats.Hits
	
	// Calculate average process time
	totalTime := int64(0)
	totalProcessed := int64(0)
	
	for _, layerStats := range stats.LayerStats {
		totalTime += layerStats.AverageProcessTime.Nanoseconds() * layerStats.TotalProcessed
		totalProcessed += layerStats.TotalProcessed
	}
	
	if totalProcessed > 0 {
		stats.AverageProcessTime = time.Duration(totalTime / totalProcessed)
	}
	
	return stats
}

// ReloadConfig reloads the configuration from file
func (me *ModerationEngine) ReloadConfig() error {
	config, err := me.configManager.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to reload config: %v", err)
	}
	
	me.mutex.Lock()
	me.config = config
	me.mutex.Unlock()
	
	log.Printf("Moderation configuration reloaded")
	return nil
}

// IsEnabled returns whether moderation is enabled
func (me *ModerationEngine) IsEnabled() bool {
	me.mutex.RLock()
	defer me.mutex.RUnlock()
	return me.config.Enabled
}

// GetConfig returns the current configuration
func (me *ModerationEngine) GetConfig() *AdvancedModerationConfig {
	me.mutex.RLock()
	defer me.mutex.RUnlock()
	return me.config
}

// Close shuts down the moderation engine
func (me *ModerationEngine) Close() {
	if me.cache != nil {
		me.cache.Close()
	}
}

// Helper methods for stats tracking

func (me *ModerationEngine) incrementTotalRequests() {
	me.mutex.Lock()
	me.stats.TotalRequests++
	me.mutex.Unlock()
}

func (me *ModerationEngine) incrementBlockedRequests() {
	me.mutex.Lock()
	me.stats.BlockedRequests++
	me.mutex.Unlock()
}

func (me *ModerationEngine) incrementFlaggedRequests() {
	me.mutex.Lock()
	me.stats.FlaggedRequests++
	me.mutex.Unlock()
}

// GetLayerNames returns the names of all registered layers
func (me *ModerationEngine) GetLayerNames() []string {
	me.mutex.RLock()
	defer me.mutex.RUnlock()
	
	names := make([]string, len(me.layers))
	for i, layer := range me.layers {
		names[i] = layer.Name()
	}
	return names
}

// GetEnabledLayers returns only the enabled layers
func (me *ModerationEngine) GetEnabledLayers() []ModerationLayer {
	me.mutex.RLock()
	defer me.mutex.RUnlock()
	
	enabled := make([]ModerationLayer, 0)
	for _, layer := range me.layers {
		if layer.Enabled() {
			enabled = append(enabled, layer)
		}
	}
	return enabled
}

// GetLayerInfo returns the weight and enabled status for a given layer name
func (me *ModerationEngine) GetLayerInfo(layerName string) (weight float64, enabled bool, found bool) {
	me.mutex.RLock()
	defer me.mutex.RUnlock()
	
	// First check registered layers
	for _, layer := range me.layers {
		if layer.Name() == layerName {
			return layer.Weight(), layer.Enabled(), true
		}
	}
	
	// Fallback to config layers
	for _, layerConfig := range me.config.Layers {
		if layerConfig.Name == layerName {
			return layerConfig.Weight, layerConfig.Enabled, true
		}
	}
	
	return 0.0, false, false
}