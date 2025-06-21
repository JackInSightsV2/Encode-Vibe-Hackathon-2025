package optimizer

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// DriftDetector monitors safety metrics for drift and degradation
type DriftDetector struct {
	baselines       map[string]*MetricBaseline
	detectors       map[string]DriftDetectionMethod
	alertManager    *DriftAlertManager
	config          *DriftDetectionConfig
	mu              sync.RWMutex
	running         bool
	stopChan        chan struct{}
}

// DriftDetectionConfig configures drift detection behavior
type DriftDetectionConfig struct {
	BaselineWindow          time.Duration                   `json:"baseline_window"`
	DetectionWindow         time.Duration                   `json:"detection_window"`
	SamplingInterval        time.Duration                   `json:"sampling_interval"`
	AlertThreshold          float64                         `json:"alert_threshold"`
	CriticalThreshold       float64                         `json:"critical_threshold"`
	MinSamples              int                             `json:"min_samples"`
	StatisticalMethods      []string                        `json:"statistical_methods"`
	AdaptiveBaseline        bool                            `json:"adaptive_baseline"`
	BaselineUpdateFrequency time.Duration                   `json:"baseline_update_frequency"`
	MetricConfigs           map[string]*MetricConfig        `json:"metric_configs"`
	NotificationChannels    []string                        `json:"notification_channels"`
	AutoResponseEnabled     bool                            `json:"auto_response_enabled"`
}

// MetricConfig configures drift detection for a specific metric
type MetricConfig struct {
	Enabled             bool                        `json:"enabled"`
	DriftMethods        []DriftDetectionMethodType  `json:"drift_methods"`
	Sensitivity         DriftSensitivity            `json:"sensitivity"`
	AlertThreshold      float64                     `json:"alert_threshold"`
	CriticalThreshold   float64                     `json:"critical_threshold"`
	ChangePointDetection bool                        `json:"change_point_detection"`
	SeasonalityAdjustment bool                       `json:"seasonality_adjustment"`
	NoiseFilter         *NoiseFilter                `json:"noise_filter"`
}

// DriftDetectionMethodType defines different drift detection methods
type DriftDetectionMethodType string

const (
	MethodKSTest           DriftDetectionMethodType = "ks_test"
	MethodWelchTest        DriftDetectionMethodType = "welch_test"
	MethodCUSUM            DriftDetectionMethodType = "cusum"
	MethodEWMA             DriftDetectionMethodType = "ewma"
	MethodPageHinkley      DriftDetectionMethodType = "page_hinkley"
	MethodADWIN            DriftDetectionMethodType = "adwin"
	MethodStatisticalTest  DriftDetectionMethodType = "statistical_test"
)

// DriftSensitivity defines sensitivity levels for drift detection
type DriftSensitivity string

const (
	SensitivityLow    DriftSensitivity = "low"
	SensitivityMedium DriftSensitivity = "medium"
	SensitivityHigh   DriftSensitivity = "high"
)

// MetricBaseline stores baseline statistics for a metric
type MetricBaseline struct {
	MetricName      string                   `json:"metric_name"`
	Mean            float64                  `json:"mean"`
	StdDev          float64                  `json:"std_dev"`
	Median          float64                  `json:"median"`
	Min             float64                  `json:"min"`
	Max             float64                  `json:"max"`
	Percentiles     map[string]float64       `json:"percentiles"`
	Distribution    *DistributionModel       `json:"distribution"`
	SampleSize      int                      `json:"sample_size"`
	WindowStart     time.Time                `json:"window_start"`
	WindowEnd       time.Time                `json:"window_end"`
	LastUpdated     time.Time                `json:"last_updated"`
	Confidence      float64                  `json:"confidence"`
	Seasonality     *SeasonalityPattern      `json:"seasonality"`
	TrendComponent  *TrendComponent          `json:"trend_component"`
}

// DistributionModel represents the statistical distribution of baseline data
type DistributionModel struct {
	Type       string             `json:"type"` // normal, exponential, poisson, etc.
	Parameters map[string]float64 `json:"parameters"`
	GoodnessOfFit float64          `json:"goodness_of_fit"`
}

// SeasonalityPattern captures seasonal patterns in metrics
type SeasonalityPattern struct {
	Type        string             `json:"type"` // daily, weekly, monthly
	Amplitude   float64            `json:"amplitude"`
	Phase       float64            `json:"phase"`
	Confidence  float64            `json:"confidence"`
	Components  map[string]float64 `json:"components"`
}

// TrendComponent captures trend information
type TrendComponent struct {
	Slope       float64   `json:"slope"`
	Intercept   float64   `json:"intercept"`
	RSquared    float64   `json:"r_squared"`
	Direction   string    `json:"direction"` // increasing, decreasing, stable
	Confidence  float64   `json:"confidence"`
}

// DriftDetectionMethod interface for different drift detection algorithms
type DriftDetectionMethod interface {
	DetectDrift(baseline *MetricBaseline, current []float64) (*DriftResult, error)
	Configure(config map[string]interface{}) error
	GetMethodType() DriftDetectionMethodType
	Reset()
}

// DriftResult contains the result of drift detection
type DriftResult struct {
	Method          DriftDetectionMethodType `json:"method"`
	MetricName      string                   `json:"metric_name"`
	DriftDetected   bool                     `json:"drift_detected"`
	DriftScore      float64                  `json:"drift_score"`
	PValue          float64                  `json:"p_value"`
	Threshold       float64                  `json:"threshold"`
	Confidence      float64                  `json:"confidence"`
	ChangePoint     *ChangePoint             `json:"change_point,omitempty"`
	Severity        DriftSeverity            `json:"severity"`
	Direction       DriftDirection           `json:"direction"`
	DetectedAt      time.Time                `json:"detected_at"`
	Metadata        map[string]interface{}   `json:"metadata"`
}

// ChangePoint represents a detected change point in the data
type ChangePoint struct {
	Timestamp    time.Time `json:"timestamp"`
	Index        int       `json:"index"`
	Confidence   float64   `json:"confidence"`
	MagnitudeChange float64 `json:"magnitude_change"`
	Type         string    `json:"type"` // mean_shift, variance_change, distribution_change
}

// DriftSeverity defines the severity of detected drift
type DriftSeverity string

const (
	SeverityLow      DriftSeverity = "low"
	SeverityMedium   DriftSeverity = "medium"
	SeverityHigh     DriftSeverity = "high"
	SeverityCritical DriftSeverity = "critical"
)

// DriftDirection indicates the direction of drift
type DriftDirection string

const (
	DirectionIncrease DriftDirection = "increase"
	DirectionDecrease DriftDirection = "decrease"
	DirectionBidirectional DriftDirection = "bidirectional"
	DirectionUnknown  DriftDirection = "unknown"
)

// NoiseFilter defines noise filtering parameters
type NoiseFilter struct {
	Enabled       bool    `json:"enabled"`
	Type          string  `json:"type"` // moving_average, gaussian, median
	WindowSize    int     `json:"window_size"`
	Threshold     float64 `json:"threshold"`
}

// DriftAlertManager manages drift detection alerts
type DriftAlertManager struct {
	alerts          []DriftAlert
	subscribers     map[string][]DriftSubscriber
	rateLimiter     *AlertRateLimiter
	mu              sync.RWMutex
}

// DriftAlert represents a drift detection alert
type DriftAlert struct {
	ID              string                 `json:"id"`
	MetricName      string                 `json:"metric_name"`
	DriftResult     *DriftResult           `json:"drift_result"`
	Severity        DriftSeverity          `json:"severity"`
	Message         string                 `json:"message"`
	TriggeredAt     time.Time              `json:"triggered_at"`
	ResolvedAt      *time.Time             `json:"resolved_at,omitempty"`
	Status          AlertStatus            `json:"status"`
	Actions         []AlertAction          `json:"actions"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// AlertStatus represents the status of an alert
type AlertStatus string

const (
	AlertStatusActive   AlertStatus = "active"
	AlertStatusResolved AlertStatus = "resolved"
	AlertStatusMuted    AlertStatus = "muted"
)

// AlertAction represents an action taken in response to an alert
type AlertAction struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	ExecutedAt  time.Time              `json:"executed_at"`
	Status      string                 `json:"status"`
	Result      map[string]interface{} `json:"result"`
}

// DriftSubscriber interface for drift alert subscribers
type DriftSubscriber interface {
	OnDriftDetected(alert DriftAlert) error
	GetSubscriberID() string
}

// AlertRateLimiter prevents alert spam
type AlertRateLimiter struct {
	limits          map[string]*RateLimit
	mu              sync.RWMutex
}

// RateLimit defines rate limiting parameters
type RateLimit struct {
	MaxAlerts    int           `json:"max_alerts"`
	TimeWindow   time.Duration `json:"time_window"`
	AlertCount   int           `json:"alert_count"`
	WindowStart  time.Time     `json:"window_start"`
}

// NewDriftDetector creates a new drift detector
func NewDriftDetector(config *DriftDetectionConfig) *DriftDetector {
	if config == nil {
		config = GetDefaultDriftDetectionConfig()
	}
	
	detector := &DriftDetector{
		baselines:    make(map[string]*MetricBaseline),
		detectors:    make(map[string]DriftDetectionMethod),
		alertManager: NewDriftAlertManager(),
		config:       config,
		stopChan:     make(chan struct{}),
	}
	
	// Initialize detection methods
	detector.initializeDetectionMethods()
	
	return detector
}

// GetDefaultDriftDetectionConfig returns default drift detection configuration
func GetDefaultDriftDetectionConfig() *DriftDetectionConfig {
	return &DriftDetectionConfig{
		BaselineWindow:          24 * time.Hour,
		DetectionWindow:         1 * time.Hour,
		SamplingInterval:        1 * time.Minute,
		AlertThreshold:          0.05,
		CriticalThreshold:       0.01,
		MinSamples:              100,
		StatisticalMethods:      []string{"ks_test", "welch_test", "cusum"},
		AdaptiveBaseline:        true,
		BaselineUpdateFrequency: 4 * time.Hour,
		MetricConfigs: map[string]*MetricConfig{
			"detection_rate": {
				Enabled:               true,
				DriftMethods:          []DriftDetectionMethodType{MethodKSTest, MethodCUSUM},
				Sensitivity:           SensitivityHigh,
				AlertThreshold:        0.02,
				CriticalThreshold:     0.005,
				ChangePointDetection:  true,
				SeasonalityAdjustment: false,
			},
			"false_positive_rate": {
				Enabled:               true,
				DriftMethods:          []DriftDetectionMethodType{MethodWelchTest, MethodEWMA},
				Sensitivity:           SensitivityHigh,
				AlertThreshold:        0.03,
				CriticalThreshold:     0.01,
				ChangePointDetection:  true,
				SeasonalityAdjustment: false,
			},
			"response_time_ms": {
				Enabled:               true,
				DriftMethods:          []DriftDetectionMethodType{MethodKSTest, MethodPageHinkley},
				Sensitivity:           SensitivityMedium,
				AlertThreshold:        0.05,
				CriticalThreshold:     0.02,
				ChangePointDetection:  true,
				SeasonalityAdjustment: true,
			},
			"user_satisfaction": {
				Enabled:               true,
				DriftMethods:          []DriftDetectionMethodType{MethodWelchTest, MethodCUSUM},
				Sensitivity:           SensitivityMedium,
				AlertThreshold:        0.05,
				CriticalThreshold:     0.02,
				ChangePointDetection:  false,
				SeasonalityAdjustment: false,
			},
		},
		NotificationChannels: []string{"email", "slack", "webhook"},
		AutoResponseEnabled:  true,
	}
}

// Start begins drift detection monitoring
func (dd *DriftDetector) Start(ctx context.Context) error {
	dd.mu.Lock()
	if dd.running {
		dd.mu.Unlock()
		return fmt.Errorf("drift detector is already running")
	}
	dd.running = true
	dd.mu.Unlock()
	
	// Start monitoring goroutine
	go dd.monitorDrift(ctx)
	
	return nil
}

// Stop stops drift detection monitoring
func (dd *DriftDetector) Stop() error {
	dd.mu.Lock()
	defer dd.mu.Unlock()
	
	if !dd.running {
		return fmt.Errorf("drift detector is not running")
	}
	
	close(dd.stopChan)
	dd.running = false
	
	return nil
}

// EstablishBaseline creates baseline metrics from historical data
func (dd *DriftDetector) EstablishBaseline(metricName string, data []float64, timestamp time.Time) error {
	if len(data) < dd.config.MinSamples {
		return fmt.Errorf("insufficient data points for baseline: got %d, need %d", len(data), dd.config.MinSamples)
	}
	
	baseline := &MetricBaseline{
		MetricName:  metricName,
		SampleSize:  len(data),
		WindowStart: timestamp.Add(-dd.config.BaselineWindow),
		WindowEnd:   timestamp,
		LastUpdated: time.Now(),
	}
	
	// Calculate basic statistics
	baseline.Mean = calculateMean(data)
	baseline.StdDev = calculateStdDev(data, baseline.Mean)
	baseline.Median = calculateMedian(data)
	baseline.Min = calculateMin(data)
	baseline.Max = calculateMax(data)
	
	// Calculate percentiles
	baseline.Percentiles = calculatePercentiles(data, []float64{5, 25, 75, 95})
	
	// Fit distribution model
	baseline.Distribution = dd.fitDistribution(data)
	
	// Calculate confidence
	baseline.Confidence = dd.calculateBaselineConfidence(data)
	
	// Detect seasonality if enabled
	if config, exists := dd.config.MetricConfigs[metricName]; exists && config.SeasonalityAdjustment {
		baseline.Seasonality = dd.detectSeasonality(data, timestamp)
	}
	
	// Detect trend
	baseline.TrendComponent = dd.detectTrend(data)
	
	dd.mu.Lock()
	dd.baselines[metricName] = baseline
	dd.mu.Unlock()
	
	return nil
}

// DetectDrift checks for drift in current metric values
func (dd *DriftDetector) DetectDrift(metricName string, currentData []float64) ([]*DriftResult, error) {
	dd.mu.RLock()
	baseline, exists := dd.baselines[metricName]
	dd.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("no baseline found for metric: %s", metricName)
	}
	
	config, exists := dd.config.MetricConfigs[metricName]
	if !exists || !config.Enabled {
		return nil, fmt.Errorf("drift detection not enabled for metric: %s", metricName)
	}
	
	results := make([]*DriftResult, 0)
	
	// Apply noise filtering if configured
	filteredData := currentData
	if config.NoiseFilter != nil && config.NoiseFilter.Enabled {
		filteredData = dd.applyNoiseFilter(currentData, config.NoiseFilter)
	}
	
	// Run each configured detection method
	for _, methodType := range config.DriftMethods {
		if detector, exists := dd.detectors[string(methodType)]; exists {
			result, err := detector.DetectDrift(baseline, filteredData)
			if err != nil {
				continue // Skip failed detections
			}
			
			// Adjust sensitivity based on configuration
			result = dd.adjustSensitivity(result, config.Sensitivity)
			
			// Determine severity
			result.Severity = dd.determineSeverity(result, config)
			
			results = append(results, result)
		}
	}
	
	// Process results and trigger alerts if necessary
	dd.processDetectionResults(metricName, results)
	
	return results, nil
}

// UpdateBaseline updates an existing baseline with new data
func (dd *DriftDetector) UpdateBaseline(metricName string, newData []float64, timestamp time.Time) error {
	dd.mu.Lock()
	defer dd.mu.Unlock()
	
	baseline, exists := dd.baselines[metricName]
	if !exists {
		return dd.EstablishBaseline(metricName, newData, timestamp)
	}
	
	if !dd.config.AdaptiveBaseline {
		return nil // Baseline updates disabled
	}
	
	// Check if enough time has passed for update
	if time.Since(baseline.LastUpdated) < dd.config.BaselineUpdateFrequency {
		return nil
	}
	
	// Combine old and new data with appropriate weighting
	combinedData := dd.combineDataForBaseline(baseline, newData)
	
	// Recalculate baseline statistics
	baseline.Mean = calculateMean(combinedData)
	baseline.StdDev = calculateStdDev(combinedData, baseline.Mean)
	baseline.Median = calculateMedian(combinedData)
	baseline.Percentiles = calculatePercentiles(combinedData, []float64{5, 25, 75, 95})
	baseline.LastUpdated = time.Now()
	baseline.SampleSize = len(combinedData)
	
	return nil
}

// monitorDrift continuously monitors for drift
func (dd *DriftDetector) monitorDrift(ctx context.Context) {
	ticker := time.NewTicker(dd.config.SamplingInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-dd.stopChan:
			return
		case <-ticker.C:
			dd.performDriftCheck()
		}
	}
}

// performDriftCheck performs drift detection for all monitored metrics
func (dd *DriftDetector) performDriftCheck() {
	dd.mu.RLock()
	metrics := make([]string, 0, len(dd.baselines))
	for metricName := range dd.baselines {
		metrics = append(metrics, metricName)
	}
	dd.mu.RUnlock()
	
	for _, metricName := range metrics {
		// Get current metric data (in real implementation, this would fetch from metrics system)
		currentData := dd.getCurrentMetricData(metricName)
		if len(currentData) == 0 {
			continue
		}
		
		// Detect drift
		results, err := dd.DetectDrift(metricName, currentData)
		if err != nil {
			continue
		}
		
		// Process any detected drift
		for _, result := range results {
			if result.DriftDetected {
				dd.handleDriftDetection(metricName, result)
			}
		}
	}
}

// initializeDetectionMethods initializes drift detection methods
func (dd *DriftDetector) initializeDetectionMethods() {
	dd.detectors[string(MethodKSTest)] = NewKSTestDetector()
	dd.detectors[string(MethodWelchTest)] = NewWelchTestDetector()
	dd.detectors[string(MethodCUSUM)] = NewCUSUMDetector()
	dd.detectors[string(MethodEWMA)] = NewEWMADetector()
	dd.detectors[string(MethodPageHinkley)] = NewPageHinkleyDetector()
}

// Statistical utility functions

func calculateMean(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	
	sum := 0.0
	for _, value := range data {
		sum += value
	}
	
	return sum / float64(len(data))
}

func calculateStdDev(data []float64, mean float64) float64 {
	if len(data) <= 1 {
		return 0
	}
	
	sumSquaredDiffs := 0.0
	for _, value := range data {
		diff := value - mean
		sumSquaredDiffs += diff * diff
	}
	
	variance := sumSquaredDiffs / float64(len(data)-1)
	return math.Sqrt(variance)
}

func calculateMedian(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	
	sorted := make([]float64, len(data))
	copy(sorted, data)
	sort.Float64s(sorted)
	
	n := len(sorted)
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}

func calculateMin(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	
	min := data[0]
	for _, value := range data[1:] {
		if value < min {
			min = value
		}
	}
	return min
}

func calculateMax(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	
	max := data[0]
	for _, value := range data[1:] {
		if value > max {
			max = value
		}
	}
	return max
}

func calculatePercentiles(data []float64, percentiles []float64) map[string]float64 {
	if len(data) == 0 {
		return make(map[string]float64)
	}
	
	sorted := make([]float64, len(data))
	copy(sorted, data)
	sort.Float64s(sorted)
	
	result := make(map[string]float64)
	n := len(sorted)
	
	for _, p := range percentiles {
		if p < 0 || p > 100 {
			continue
		}
		
		index := (p / 100.0) * float64(n-1)
		lower := int(math.Floor(index))
		upper := int(math.Ceil(index))
		
		if lower == upper {
			result[fmt.Sprintf("p%.0f", p)] = sorted[lower]
		} else {
			weight := index - float64(lower)
			result[fmt.Sprintf("p%.0f", p)] = sorted[lower]*(1-weight) + sorted[upper]*weight
		}
	}
	
	return result
}

// Additional helper methods

func (dd *DriftDetector) fitDistribution(data []float64) *DistributionModel {
	// Simplified distribution fitting - in production use proper statistical libraries
	mean := calculateMean(data)
	stdDev := calculateStdDev(data, mean)
	
	return &DistributionModel{
		Type: "normal",
		Parameters: map[string]float64{
			"mean":   mean,
			"stddev": stdDev,
		},
		GoodnessOfFit: 0.8, // Simplified
	}
}

func (dd *DriftDetector) calculateBaselineConfidence(data []float64) float64 {
	// Simplified confidence calculation based on sample size
	sampleSize := float64(len(data))
	confidence := math.Min(0.99, 0.5+0.4*math.Log(sampleSize)/math.Log(1000))
	return confidence
}

func (dd *DriftDetector) detectSeasonality(data []float64, timestamp time.Time) *SeasonalityPattern {
	// Simplified seasonality detection
	return &SeasonalityPattern{
		Type:       "daily",
		Amplitude:  0.1,
		Phase:      0.0,
		Confidence: 0.6,
		Components: map[string]float64{
			"daily": 0.1,
		},
	}
}

func (dd *DriftDetector) detectTrend(data []float64) *TrendComponent {
	// Simplified linear trend detection
	if len(data) < 2 {
		return &TrendComponent{
			Direction:  "stable",
			Confidence: 0.0,
		}
	}
	
	// Calculate simple linear regression
	n := float64(len(data))
	sumX := n * (n - 1) / 2  // 0 + 1 + 2 + ... + (n-1)
	sumY := 0.0
	sumXY := 0.0
	sumX2 := n * (n - 1) * (2*n - 1) / 6  // 0² + 1² + 2² + ... + (n-1)²
	
	for i, y := range data {
		x := float64(i)
		sumY += y
		sumXY += x * y
	}
	
	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	intercept := (sumY - slope*sumX) / n
	
	// Determine direction
	direction := "stable"
	if math.Abs(slope) > 0.001 {
		if slope > 0 {
			direction = "increasing"
		} else {
			direction = "decreasing"
		}
	}
	
	return &TrendComponent{
		Slope:      slope,
		Intercept:  intercept,
		Direction:  direction,
		Confidence: 0.7, // Simplified
	}
}

func (dd *DriftDetector) applyNoiseFilter(data []float64, filter *NoiseFilter) []float64 {
	if !filter.Enabled || len(data) == 0 {
		return data
	}
	
	switch filter.Type {
	case "moving_average":
		return dd.applyMovingAverage(data, filter.WindowSize)
	case "median":
		return dd.applyMedianFilter(data, filter.WindowSize)
	default:
		return data
	}
}

func (dd *DriftDetector) applyMovingAverage(data []float64, windowSize int) []float64 {
	if windowSize <= 1 || len(data) <= windowSize {
		return data
	}
	
	filtered := make([]float64, len(data))
	
	for i := range data {
		start := max(0, i-windowSize/2)
		end := min(len(data), i+windowSize/2+1)
		
		sum := 0.0
		count := 0
		for j := start; j < end; j++ {
			sum += data[j]
			count++
		}
		
		filtered[i] = sum / float64(count)
	}
	
	return filtered
}

func (dd *DriftDetector) applyMedianFilter(data []float64, windowSize int) []float64 {
	if windowSize <= 1 || len(data) <= windowSize {
		return data
	}
	
	filtered := make([]float64, len(data))
	
	for i := range data {
		start := max(0, i-windowSize/2)
		end := min(len(data), i+windowSize/2+1)
		
		window := make([]float64, end-start)
		copy(window, data[start:end])
		
		filtered[i] = calculateMedian(window)
	}
	
	return filtered
}

func (dd *DriftDetector) adjustSensitivity(result *DriftResult, sensitivity DriftSensitivity) *DriftResult {
	// Adjust thresholds based on sensitivity
	switch sensitivity {
	case SensitivityHigh:
		result.Threshold *= 0.5
	case SensitivityLow:
		result.Threshold *= 2.0
	// Medium sensitivity uses default threshold
	}
	
	// Recheck drift detection with adjusted threshold
	result.DriftDetected = result.DriftScore > result.Threshold
	
	return result
}

func (dd *DriftDetector) determineSeverity(result *DriftResult, config *MetricConfig) DriftSeverity {
	if result.DriftScore > config.CriticalThreshold {
		return SeverityCritical
	} else if result.DriftScore > config.AlertThreshold {
		return SeverityHigh
	} else if result.DriftScore > config.AlertThreshold*0.5 {
		return SeverityMedium
	}
	return SeverityLow
}

func (dd *DriftDetector) processDetectionResults(metricName string, results []*DriftResult) {
	for _, result := range results {
		if result.DriftDetected && result.Severity >= SeverityMedium {
			alert := DriftAlert{
				ID:          generateDriftAlertID(),
				MetricName:  metricName,
				DriftResult: result,
				Severity:    result.Severity,
				Message:     fmt.Sprintf("Drift detected in %s using %s method", metricName, result.Method),
				TriggeredAt: time.Now(),
				Status:      AlertStatusActive,
				Metadata: map[string]interface{}{
					"drift_score": result.DriftScore,
					"threshold":   result.Threshold,
				},
			}
			
			dd.alertManager.TriggerAlert(alert)
		}
	}
}

func (dd *DriftDetector) getCurrentMetricData(metricName string) []float64 {
	// Simulate getting current metric data
	// In real implementation, this would fetch from metrics system
	data := make([]float64, 100)
	
	baseline := dd.baselines[metricName]
	if baseline == nil {
		return []float64{}
	}
	
	// Generate data around baseline with potential drift
	for i := range data {
		noise := (rand.Float64() - 0.5) * 0.1
		drift := 0.0
		
		// Simulate drift for demonstration
		if time.Since(baseline.LastUpdated) > 30*time.Minute {
			drift = 0.05 // 5% drift
		}
		
		data[i] = baseline.Mean + drift + noise*baseline.StdDev
	}
	
	return data
}

func (dd *DriftDetector) combineDataForBaseline(baseline *MetricBaseline, newData []float64) []float64 {
	// Simple combination - in production use more sophisticated methods
	// like exponential decay or sliding window
	
	maxBaselineSize := 1000
	decay := 0.9 // Weight for old data
	
	// Create synthetic old data based on baseline stats
	oldDataSize := min(maxBaselineSize, baseline.SampleSize)
	combinedData := make([]float64, 0, oldDataSize+len(newData))
	
	// Add weighted old data (simplified)
	for i := 0; i < oldDataSize; i++ {
		value := baseline.Mean + rand.NormFloat64()*baseline.StdDev
		combinedData = append(combinedData, value*decay)
	}
	
	// Add new data
	combinedData = append(combinedData, newData...)
	
	return combinedData
}

func (dd *DriftDetector) handleDriftDetection(metricName string, result *DriftResult) {
	// Implement automatic response if enabled
	if dd.config.AutoResponseEnabled {
		switch result.Severity {
		case SeverityCritical:
			dd.handleCriticalDrift(metricName, result)
		case SeverityHigh:
			dd.handleHighSeverityDrift(metricName, result)
		}
	}
}

func (dd *DriftDetector) handleCriticalDrift(metricName string, result *DriftResult) {
	// Critical drift response - could trigger automatic rollback
	fmt.Printf("CRITICAL DRIFT DETECTED in %s: score=%.4f, method=%s\n", 
		metricName, result.DriftScore, result.Method)
}

func (dd *DriftDetector) handleHighSeverityDrift(metricName string, result *DriftResult) {
	// High severity drift response - increase monitoring, alert teams
	fmt.Printf("HIGH SEVERITY DRIFT DETECTED in %s: score=%.4f, method=%s\n", 
		metricName, result.DriftScore, result.Method)
}

// Helper functions

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func generateDriftAlertID() string {
	return fmt.Sprintf("drift_alert_%d_%s", time.Now().Unix(), generateRandomString(6))
}

// Alert Manager implementation

func NewDriftAlertManager() *DriftAlertManager {
	return &DriftAlertManager{
		alerts:      make([]DriftAlert, 0),
		subscribers: make(map[string][]DriftSubscriber),
		rateLimiter: NewAlertRateLimiter(),
	}
}

func (dam *DriftAlertManager) TriggerAlert(alert DriftAlert) {
	dam.mu.Lock()
	defer dam.mu.Unlock()
	
	// Check rate limiting
	if !dam.rateLimiter.AllowAlert(alert.MetricName) {
		return
	}
	
	dam.alerts = append(dam.alerts, alert)
	
	// Notify subscribers
	if subscribers, exists := dam.subscribers[alert.MetricName]; exists {
		for _, subscriber := range subscribers {
			go subscriber.OnDriftDetected(alert)
		}
	}
	
	// Print alert for demonstration
	fmt.Printf("DRIFT ALERT [%s]: %s\n", alert.Severity, alert.Message)
}

func NewAlertRateLimiter() *AlertRateLimiter {
	return &AlertRateLimiter{
		limits: make(map[string]*RateLimit),
	}
}

func (arl *AlertRateLimiter) AllowAlert(metricName string) bool {
	arl.mu.Lock()
	defer arl.mu.Unlock()
	
	limit, exists := arl.limits[metricName]
	if !exists {
		limit = &RateLimit{
			MaxAlerts:   10,
			TimeWindow:  1 * time.Hour,
			AlertCount:  0,
			WindowStart: time.Now(),
		}
		arl.limits[metricName] = limit
	}
	
	// Reset window if expired
	if time.Since(limit.WindowStart) > limit.TimeWindow {
		limit.AlertCount = 0
		limit.WindowStart = time.Now()
	}
	
	// Check if limit exceeded
	if limit.AlertCount >= limit.MaxAlerts {
		return false
	}
	
	limit.AlertCount++
	return true
}