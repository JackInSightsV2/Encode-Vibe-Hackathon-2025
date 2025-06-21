package optimizer

import (
	"fmt"
	"math"
	"sort"
	"time"
)

// StatisticsEngine provides statistical analysis for experiments
type StatisticsEngine struct {
	confidenceLevel float64
	minSampleSize   int
}

// StatisticalTest represents different types of statistical tests
type StatisticalTest string

const (
	TestTypeZTest     StatisticalTest = "z_test"
	TestTypeTTest     StatisticalTest = "t_test"
	TestTypeChi2      StatisticalTest = "chi2_test"
	TestTypeFisher    StatisticalTest = "fisher_exact"
	TestTypeMannWhitney StatisticalTest = "mann_whitney"
)

// StatisticalResult contains the results of a statistical test
type StatisticalResult struct {
	TestType     StatisticalTest `json:"test_type"`
	PValue       float64         `json:"p_value"`
	ZScore       float64         `json:"z_score,omitempty"`
	TScore       float64         `json:"t_score,omitempty"`
	ChiSquare    float64         `json:"chi_square,omitempty"`
	DegreesOfFreedom int         `json:"degrees_of_freedom,omitempty"`
	Significant  bool            `json:"significant"`
	EffectSize   float64         `json:"effect_size"`
	ConfidenceInterval ConfidenceInterval `json:"confidence_interval"`
	SampleSizes  map[string]int  `json:"sample_sizes"`
	PowerAnalysis *PowerAnalysis  `json:"power_analysis,omitempty"`
}

// ConfidenceInterval represents a confidence interval
type ConfidenceInterval struct {
	Lower      float64 `json:"lower"`
	Upper      float64 `json:"upper"`
	Level      float64 `json:"level"`
	Metric     string  `json:"metric"`
}

// PowerAnalysis contains power analysis results
type PowerAnalysis struct {
	Power              float64 `json:"power"`
	SampleSizeNeeded   int     `json:"sample_size_needed"`
	MinDetectableEffect float64 `json:"min_detectable_effect"`
	ActualEffectSize   float64 `json:"actual_effect_size"`
}

// SafetyMonitor monitors experiments for safety violations
type SafetyMonitor struct {
	thresholds *SafetyThresholds
	violations map[string][]SafetyViolation
}

// SafetyViolation represents a safety threshold violation
type SafetyViolation struct {
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Metric      string                 `json:"metric"`
	Value       float64                `json:"value"`
	Threshold   float64                `json:"threshold"`
	Timestamp   time.Time              `json:"timestamp"`
	Message     string                 `json:"message"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// NewStatisticsEngine creates a new statistics engine
func NewStatisticsEngine() *StatisticsEngine {
	return &StatisticsEngine{
		confidenceLevel: 0.95,
		minSampleSize:   100,
	}
}

// NewSafetyMonitor creates a new safety monitor
func NewSafetyMonitor() *SafetyMonitor {
	return &SafetyMonitor{
		thresholds: GetDefaultGlobalConstraints().SafetyThresholds,
		violations: make(map[string][]SafetyViolation),
	}
}

// CalculateSignificance calculates statistical significance between two variants
func (se *StatisticsEngine) CalculateSignificance(control, variant *VariantMetrics) *SignificanceTest {
	if control.SampleSize < se.minSampleSize || variant.SampleSize < se.minSampleSize {
		return &SignificanceTest{
			PValue:      1.0,
			Significant: false,
			Method:      "insufficient_sample_size",
		}
	}
	
	// Perform proportion test for detection rate
	result := se.performProportionTest(
		control.DetectionRate, control.SampleSize,
		variant.DetectionRate, variant.SampleSize,
	)
	
	return &SignificanceTest{
		PValue:      result.PValue,
		ZScore:      result.ZScore,
		Significant: result.Significant,
		Improvement: calculateImprovement(control.DetectionRate, variant.DetectionRate),
		Method:      string(result.TestType),
	}
}

// performProportionTest performs a two-proportion z-test
func (se *StatisticsEngine) performProportionTest(p1 float64, n1 int, p2 float64, n2 int) *StatisticalResult {
	// Two-proportion z-test
	
	// Calculate pooled proportion
	x1 := p1 * float64(n1)
	x2 := p2 * float64(n2)
	pooledP := (x1 + x2) / float64(n1 + n2)
	
	// Calculate standard error
	stdErr := math.Sqrt(pooledP * (1 - pooledP) * (1/float64(n1) + 1/float64(n2)))
	
	// Calculate z-score
	zScore := (p2 - p1) / stdErr
	
	// Calculate p-value (two-tailed)
	pValue := 2 * (1 - normalCDF(math.Abs(zScore)))
	
	// Check significance
	alpha := 1.0 - se.confidenceLevel
	significant := pValue < alpha
	
	// Calculate effect size (Cohen's h)
	effectSize := 2 * (math.Asin(math.Sqrt(p2)) - math.Asin(math.Sqrt(p1)))
	
	// Calculate confidence interval for difference
	diffSE := math.Sqrt(p1*(1-p1)/float64(n1) + p2*(1-p2)/float64(n2))
	margin := 1.96 * diffSE // 95% CI
	diff := p2 - p1
	
	return &StatisticalResult{
		TestType:    TestTypeZTest,
		PValue:      pValue,
		ZScore:      zScore,
		Significant: significant,
		EffectSize:  effectSize,
		ConfidenceInterval: ConfidenceInterval{
			Lower: diff - margin,
			Upper: diff + margin,
			Level: se.confidenceLevel,
			Metric: "detection_rate_difference",
		},
		SampleSizes: map[string]int{
			"control": n1,
			"variant": n2,
		},
	}
}

// performTTest performs a two-sample t-test for continuous metrics
func (se *StatisticsEngine) performTTest(mean1, std1 float64, n1 int, mean2, std2 float64, n2 int) *StatisticalResult {
	// Welch's t-test (unequal variances)
	
	// Calculate standard error
	se1 := std1 * std1 / float64(n1)
	se2 := std2 * std2 / float64(n2)
	stdError := math.Sqrt(se1 + se2)
	
	// Calculate t-score
	tScore := (mean2 - mean1) / stdError
	
	// Calculate degrees of freedom (Welch-Satterthwaite equation)
	df := math.Pow(se1+se2, 2) / (se1*se1/float64(n1-1) + se2*se2/float64(n2-1))
	
	// Calculate p-value (approximation)
	pValue := 2 * (1 - tCDF(math.Abs(tScore), int(df)))
	
	// Check significance
	alpha := 1.0 - se.confidenceLevel
	significant := pValue < alpha
	
	// Calculate effect size (Cohen's d)
	pooledStd := math.Sqrt(((float64(n1)-1)*std1*std1 + (float64(n2)-1)*std2*std2) / float64(n1+n2-2))
	effectSize := (mean2 - mean1) / pooledStd
	
	return &StatisticalResult{
		TestType:         TestTypeTTest,
		PValue:           pValue,
		TScore:           tScore,
		DegreesOfFreedom: int(df),
		Significant:      significant,
		EffectSize:       effectSize,
		SampleSizes: map[string]int{
			"control": n1,
			"variant": n2,
		},
	}
}

// CalculatePowerAnalysis calculates statistical power for an experiment
func (se *StatisticsEngine) CalculatePowerAnalysis(control, variant *VariantMetrics, targetEffect float64) *PowerAnalysis {
	alpha := 1.0 - se.confidenceLevel
	
	// Calculate current effect size
	actualEffect := calculateEffectSize(control.DetectionRate, variant.DetectionRate)
	
	// Calculate power (simplified calculation)
	// This is a basic implementation - in production you'd use more sophisticated methods
	n := float64(min(control.SampleSize, variant.SampleSize))
	
	// Effect size for power calculation
	effectForPower := targetEffect
	if actualEffect != 0 {
		effectForPower = actualEffect
	}
	
	// Simplified power calculation
	power := calculatePower(effectForPower, n, alpha)
	
	// Calculate required sample size for desired power (0.8)
	sampleSizeNeeded := calculateSampleSizeForPower(effectForPower, 0.8, alpha)
	
	return &PowerAnalysis{
		Power:               power,
		SampleSizeNeeded:    sampleSizeNeeded,
		MinDetectableEffect: targetEffect,
		ActualEffectSize:    actualEffect,
	}
}

// UpdateExperimentStatistics updates statistical metrics for an experiment
func (se *StatisticsEngine) UpdateExperimentStatistics(experiment *Experiment) {
	controlMetrics := experiment.Results.GetVariantMetrics("control")
	if controlMetrics == nil {
		return
	}
	
	// Update statistics for each variant
	for variantID, variantMetrics := range experiment.Results.VariantMetrics {
		if variantID == "control" {
			continue
		}
		
		// Calculate significance
		significance := se.CalculateSignificance(controlMetrics, variantMetrics)
		experiment.Results.Significance = significance
		
		// Calculate confidence intervals
		if variantMetrics.SampleSize >= se.minSampleSize {
			_ = se.calculateConfidenceInterval(variantMetrics.DetectionRate, variantMetrics.SampleSize)
			// Store CI in variant metadata (simplified for this implementation)
			if variantMetrics.ResponseTimes == nil {
				variantMetrics.ResponseTimes = make([]int64, 0)
			}
		}
		
		// Update fitness score with statistical confidence
		if significance.Significant {
			// Boost fitness score for statistically significant improvements
			improvementBoost := math.Min(significance.Improvement * 0.1, 0.2)
			variantMetrics.FitnessScore += improvementBoost
		}
	}
}

// calculateConfidenceInterval calculates confidence interval for a proportion
func (se *StatisticsEngine) calculateConfidenceInterval(proportion float64, sampleSize int) ConfidenceInterval {
	// Wilson score interval (more accurate than normal approximation)
	n := float64(sampleSize)
	z := 1.96 // 95% confidence level
	
	center := (proportion + z*z/(2*n)) / (1 + z*z/n)
	margin := z * math.Sqrt((proportion*(1-proportion) + z*z/(4*n))/n) / (1 + z*z/n)
	
	return ConfidenceInterval{
		Lower: math.Max(0, center - margin),
		Upper: math.Min(1, center + margin),
		Level: se.confidenceLevel,
		Metric: "detection_rate",
	}
}

// CheckSequentialSignificance performs sequential significance testing
func (se *StatisticsEngine) CheckSequentialSignificance(experiment *Experiment) (*StatisticalResult, bool) {
	// Alpha spending function for sequential testing
	// This implements a simplified version of O'Brien-Fleming boundaries
	
	controlMetrics := experiment.Results.GetVariantMetrics("control")
	if controlMetrics == nil {
		return nil, false
	}
	
	// Find the variant with highest sample size
	var bestVariant *VariantMetrics
	for variantID, variant := range experiment.Results.VariantMetrics {
		if variantID == "control" {
			continue
		}
		
		if bestVariant == nil || variant.SampleSize > bestVariant.SampleSize {
			bestVariant = variant
		}
	}
	
	if bestVariant == nil {
		return nil, false
	}
	
	// Calculate information fraction (how far along we are)
	targetSampleSize := experiment.Config.MinSampleSize
	infoFraction := float64(bestVariant.SampleSize) / float64(targetSampleSize)
	
	if infoFraction < 0.1 {
		return nil, false // Too early to test
	}
	
	// Calculate adjusted alpha for sequential testing
	adjustedAlpha := se.calculateSequentialAlpha(infoFraction)
	
	// Perform test with adjusted alpha
	originalConfidence := se.confidenceLevel
	se.confidenceLevel = 1.0 - adjustedAlpha
	
	result := se.performProportionTest(
		controlMetrics.DetectionRate, controlMetrics.SampleSize,
		bestVariant.DetectionRate, bestVariant.SampleSize,
	)
	
	se.confidenceLevel = originalConfidence // Restore original confidence level
	
	return result, result.Significant
}

// calculateSequentialAlpha calculates adjusted alpha for sequential testing
func (se *StatisticsEngine) calculateSequentialAlpha(infoFraction float64) float64 {
	// O'Brien-Fleming boundary approximation
	if infoFraction <= 0 {
		return 0
	}
	
	baseAlpha := 0.05
	adjustment := 2 * (1 - normalCDF(1.96/math.Sqrt(infoFraction)))
	
	return math.Min(adjustment, baseAlpha)
}

// StartMonitoring starts safety monitoring for an experiment
func (sm *SafetyMonitor) StartMonitoring(experiment *Experiment) {
	// Initialize violation tracking for this experiment
	sm.violations[experiment.ID] = make([]SafetyViolation, 0)
}

// StopMonitoring stops safety monitoring for an experiment
func (sm *SafetyMonitor) StopMonitoring(experimentID string) {
	delete(sm.violations, experimentID)
}

// CheckSafetyViolation checks for safety threshold violations
func (sm *SafetyMonitor) CheckSafetyViolation(experiment *Experiment) *SafetyViolation {
	for variantID, metrics := range experiment.Results.VariantMetrics {
		// Check detection rate
		if metrics.DetectionRate < sm.thresholds.MinDetectionRate {
			violation := SafetyViolation{
				Type:      "detection_rate_below_threshold",
				Severity:  "critical",
				Metric:    "detection_rate",
				Value:     metrics.DetectionRate,
				Threshold: sm.thresholds.MinDetectionRate,
				Timestamp: time.Now(),
				Message:   fmt.Sprintf("Variant %s detection rate %.3f below minimum %.3f", 
					variantID, metrics.DetectionRate, sm.thresholds.MinDetectionRate),
				Metadata: map[string]interface{}{
					"variant_id":   variantID,
					"sample_size":  metrics.SampleSize,
				},
			}
			
			sm.violations[experiment.ID] = append(sm.violations[experiment.ID], violation)
			return &violation
		}
		
		// Check false positive rate
		if metrics.FalsePositiveRate > sm.thresholds.MaxFalsePositiveRate {
			violation := SafetyViolation{
				Type:      "false_positive_rate_above_threshold",
				Severity:  "warning",
				Metric:    "false_positive_rate",
				Value:     metrics.FalsePositiveRate,
				Threshold: sm.thresholds.MaxFalsePositiveRate,
				Timestamp: time.Now(),
				Message:   fmt.Sprintf("Variant %s false positive rate %.3f above maximum %.3f", 
					variantID, metrics.FalsePositiveRate, sm.thresholds.MaxFalsePositiveRate),
				Metadata: map[string]interface{}{
					"variant_id":   variantID,
					"sample_size":  metrics.SampleSize,
				},
			}
			
			sm.violations[experiment.ID] = append(sm.violations[experiment.ID], violation)
			return &violation
		}
		
		// Check response time
		responseTimeMs := metrics.AvgResponseTime.Seconds() * 1000
		if responseTimeMs > sm.thresholds.MaxResponseTimeMs {
			violation := SafetyViolation{
				Type:      "response_time_above_threshold",
				Severity:  "warning",
				Metric:    "response_time_ms",
				Value:     responseTimeMs,
				Threshold: sm.thresholds.MaxResponseTimeMs,
				Timestamp: time.Now(),
				Message:   fmt.Sprintf("Variant %s response time %.1fms above maximum %.1fms", 
					variantID, responseTimeMs, sm.thresholds.MaxResponseTimeMs),
				Metadata: map[string]interface{}{
					"variant_id":   variantID,
					"sample_size":  metrics.SampleSize,
				},
			}
			
			sm.violations[experiment.ID] = append(sm.violations[experiment.ID], violation)
			return &violation
		}
		
		// Check user satisfaction
		if metrics.UserSatisfaction < sm.thresholds.MinUserSatisfaction && metrics.SampleSize > 10 {
			violation := SafetyViolation{
				Type:      "user_satisfaction_below_threshold",
				Severity:  "warning",
				Metric:    "user_satisfaction",
				Value:     metrics.UserSatisfaction,
				Threshold: sm.thresholds.MinUserSatisfaction,
				Timestamp: time.Now(),
				Message:   fmt.Sprintf("Variant %s user satisfaction %.3f below minimum %.3f", 
					variantID, metrics.UserSatisfaction, sm.thresholds.MinUserSatisfaction),
				Metadata: map[string]interface{}{
					"variant_id":   variantID,
					"sample_size":  metrics.SampleSize,
				},
			}
			
			sm.violations[experiment.ID] = append(sm.violations[experiment.ID], violation)
			return &violation
		}
	}
	
	return nil
}

// GetViolationHistory returns violation history for an experiment
func (sm *SafetyMonitor) GetViolationHistory(experimentID string) []SafetyViolation {
	if violations, exists := sm.violations[experimentID]; exists {
		return violations
	}
	return make([]SafetyViolation, 0)
}

// Utility functions

func calculateImprovement(baseline, variant float64) float64 {
	if baseline == 0 {
		return 0
	}
	return (variant - baseline) / baseline
}

func calculateEffectSize(baseline, variant float64) float64 {
	// Cohen's h for proportions
	return 2 * (math.Asin(math.Sqrt(variant)) - math.Asin(math.Sqrt(baseline)))
}

func calculatePower(effectSize, sampleSize, alpha float64) float64 {
	// Simplified power calculation
	// In production, you'd use more sophisticated statistical libraries
	
	criticalValue := 1.96 // for alpha = 0.05
	
	// Standard error for effect size
	se := math.Sqrt(2 / sampleSize)
	
	// Non-central parameter
	ncp := effectSize / se
	
	// Power approximation
	power := 1 - normalCDF(criticalValue - ncp)
	
	return math.Max(0, math.Min(1, power))
}

func calculateSampleSizeForPower(effectSize, power, alpha float64) int {
	// Simplified sample size calculation
	zAlpha := 1.96  // for alpha = 0.05
	zBeta := 0.84   // for power = 0.8
	
	if power == 0.8 {
		zBeta = 0.84
	} else if power == 0.9 {
		zBeta = 1.28
	} else {
		// Approximate z-score for given power
		zBeta = -normalInverse(1 - power)
	}
	
	sampleSize := 2 * math.Pow(zAlpha + zBeta, 2) / math.Pow(effectSize, 2)
	
	return int(math.Ceil(sampleSize))
}

// Approximation functions for statistical distributions

func normalCDF(x float64) float64 {
	// Approximation of normal CDF using error function
	return 0.5 * (1 + math.Erf(x/math.Sqrt(2)))
}

func normalInverse(p float64) float64 {
	// Approximation of inverse normal CDF
	if p <= 0 {
		return math.Inf(-1)
	}
	if p >= 1 {
		return math.Inf(1)
	}
	
	// Beasley-Springer-Moro algorithm approximation
	a := []float64{0, -3.969683028665376e+01, 2.209460984245205e+02, -2.759285104469687e+02, 1.383577518672690e+02, -3.066479806614716e+01, 2.506628277459239e+00}
	b := []float64{0, -5.447609879822406e+01, 1.615858368580409e+02, -1.556989798598866e+02, 6.680131188771972e+01, -1.328068155288572e+01}
	
	if p < 0.5 {
		return -normalInverse(1 - p)
	}
	
	y := math.Sqrt(-2 * math.Log(1 - p))
	
	num := a[6]
	den := 1.0
	
	for i := 5; i >= 1; i-- {
		num = num*y + a[i]
		den = den*y + b[i]
	}
	
	return y - num/den
}

func tCDF(t float64, df int) float64 {
	// Approximation of t-distribution CDF
	// For large df, approximate with normal distribution
	if df > 30 {
		return normalCDF(t)
	}
	
	// Simplified approximation for smaller df
	// In production, you'd use a proper statistical library
	x := t / math.Sqrt(float64(df))
	return 0.5 + math.Atan(x)/math.Pi
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// CalculateBayesianCredibleInterval calculates Bayesian credible intervals
func (se *StatisticsEngine) CalculateBayesianCredibleInterval(successes, trials int, credibilityLevel float64) ConfidenceInterval {
	// Beta distribution with uniform prior (Beta(1,1))
	alpha := float64(successes + 1)
	beta := float64(trials - successes + 1)
	
	// For simplicity, use normal approximation to beta distribution
	mean := alpha / (alpha + beta)
	variance := (alpha * beta) / ((alpha + beta) * (alpha + beta) * (alpha + beta + 1))
	std := math.Sqrt(variance)
	
	// Calculate credible interval
	tail := (1.0 - credibilityLevel) / 2.0
	zScore := -normalInverse(tail)
	
	lower := math.Max(0, mean - zScore*std)
	upper := math.Min(1, mean + zScore*std)
	
	return ConfidenceInterval{
		Lower: lower,
		Upper: upper,
		Level: credibilityLevel,
		Metric: "bayesian_proportion",
	}
}

// CalculateMultipleComparisonAdjustment applies multiple comparison correction
func (se *StatisticsEngine) CalculateMultipleComparisonAdjustment(pValues []float64, method string) []float64 {
	n := len(pValues)
	if n == 0 {
		return pValues
	}
	
	adjusted := make([]float64, n)
	
	switch method {
	case "bonferroni":
		// Bonferroni correction
		for i, p := range pValues {
			adjusted[i] = math.Min(1.0, p * float64(n))
		}
		
	case "benjamini_hochberg":
		// Benjamini-Hochberg (FDR) procedure
		
		// Create sorted indices
		type pValueIndex struct {
			value float64
			index int
		}
		
		sorted := make([]pValueIndex, n)
		for i, p := range pValues {
			sorted[i] = pValueIndex{value: p, index: i}
		}
		
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].value < sorted[j].value
		})
		
		// Apply BH procedure
		for i := 0; i < n; i++ {
			rank := i + 1
			adjustment := float64(n) / float64(rank)
			originalIndex := sorted[i].index
			adjusted[originalIndex] = math.Min(1.0, sorted[i].value * adjustment)
		}
		
		// Ensure monotonicity
		for i := n - 2; i >= 0; i-- {
			curr := sorted[i].index
			next := sorted[i+1].index
			if adjusted[curr] > adjusted[next] {
				adjusted[curr] = adjusted[next]
			}
		}
		
	default:
		// No adjustment
		copy(adjusted, pValues)
	}
	
	return adjusted
}