package optimizer

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"
)

// KSTestDetector implements Kolmogorov-Smirnov test for drift detection
type KSTestDetector struct {
	threshold float64
}

// NewKSTestDetector creates a new KS test detector
func NewKSTestDetector() *KSTestDetector {
	return &KSTestDetector{
		threshold: 0.05,
	}
}

func (ks *KSTestDetector) DetectDrift(baseline *MetricBaseline, current []float64) (*DriftResult, error) {
	if len(current) == 0 {
		return nil, fmt.Errorf("no current data provided")
	}
	
	// Generate baseline data for comparison (simplified)
	baselineData := ks.generateBaselineData(baseline, len(current))
	
	// Perform KS test
	dStatistic, pValue := ks.kolmogorovSmirnovTest(baselineData, current)
	
	result := &DriftResult{
		Method:        MethodKSTest,
		MetricName:    baseline.MetricName,
		DriftDetected: pValue < ks.threshold,
		DriftScore:    dStatistic,
		PValue:        pValue,
		Threshold:     ks.threshold,
		Confidence:    1.0 - pValue,
		Direction:     ks.determineDriftDirection(baseline, current),
		DetectedAt:    time.Now(),
		Metadata: map[string]interface{}{
			"baseline_mean": baseline.Mean,
			"current_mean":  calculateMean(current),
			"d_statistic":   dStatistic,
		},
	}
	
	return result, nil
}

func (ks *KSTestDetector) Configure(config map[string]interface{}) error {
	if threshold, exists := config["threshold"]; exists {
		if t, ok := threshold.(float64); ok {
			ks.threshold = t
		}
	}
	return nil
}

func (ks *KSTestDetector) GetMethodType() DriftDetectionMethodType {
	return MethodKSTest
}

func (ks *KSTestDetector) Reset() {
	// Nothing to reset for KS test
}

func (ks *KSTestDetector) generateBaselineData(baseline *MetricBaseline, size int) []float64 {
	// Generate synthetic data based on baseline statistics
	data := make([]float64, size)
	for i := range data {
		// Simple normal distribution generation
		data[i] = baseline.Mean + (randNormFloat64() * baseline.StdDev)
	}
	return data
}

func (ks *KSTestDetector) kolmogorovSmirnovTest(sample1, sample2 []float64) (dStatistic, pValue float64) {
	// Sort both samples
	sorted1 := make([]float64, len(sample1))
	sorted2 := make([]float64, len(sample2))
	copy(sorted1, sample1)
	copy(sorted2, sample2)
	sort.Float64s(sorted1)
	sort.Float64s(sorted2)
	
	n1, n2 := float64(len(sorted1)), float64(len(sorted2))
	
	// Calculate empirical distribution functions and find maximum difference
	maxDiff := 0.0
	i, j := 0, 0
	
	for i < len(sorted1) && j < len(sorted2) {
		x1, x2 := sorted1[i], sorted2[j]
		
		if x1 <= x2 {
			cdf1 := float64(i+1) / n1
			cdf2 := float64(j) / n2
			diff := math.Abs(cdf1 - cdf2)
			if diff > maxDiff {
				maxDiff = diff
			}
			i++
		}
		
		if x2 <= x1 {
			cdf1 := float64(i) / n1
			cdf2 := float64(j+1) / n2
			diff := math.Abs(cdf1 - cdf2)
			if diff > maxDiff {
				maxDiff = diff
			}
			j++
		}
	}
	
	// Calculate p-value using asymptotic distribution
	sqrtN := math.Sqrt(n1 * n2 / (n1 + n2))
	lambda := maxDiff * sqrtN
	
	// Approximate p-value calculation
	pValue = 2.0 * math.Exp(-2.0 * lambda * lambda)
	
	return maxDiff, pValue
}

func (ks *KSTestDetector) determineDriftDirection(baseline *MetricBaseline, current []float64) DriftDirection {
	currentMean := calculateMean(current)
	if currentMean > baseline.Mean*1.05 {
		return DirectionIncrease
	} else if currentMean < baseline.Mean*0.95 {
		return DirectionDecrease
	}
	return DirectionUnknown
}

// WelchTestDetector implements Welch's t-test for drift detection
type WelchTestDetector struct {
	threshold float64
}

func NewWelchTestDetector() *WelchTestDetector {
	return &WelchTestDetector{
		threshold: 0.05,
	}
}

func (wt *WelchTestDetector) DetectDrift(baseline *MetricBaseline, current []float64) (*DriftResult, error) {
	if len(current) == 0 {
		return nil, fmt.Errorf("no current data provided")
	}
	
	currentMean := calculateMean(current)
	currentStdDev := calculateStdDev(current, currentMean)
	
	// Perform Welch's t-test
	tStatistic, pValue := wt.welchTTest(
		baseline.Mean, baseline.StdDev, baseline.SampleSize,
		currentMean, currentStdDev, len(current),
	)
	
	result := &DriftResult{
		Method:        MethodWelchTest,
		MetricName:    baseline.MetricName,
		DriftDetected: pValue < wt.threshold,
		DriftScore:    math.Abs(tStatistic),
		PValue:        pValue,
		Threshold:     wt.threshold,
		Confidence:    1.0 - pValue,
		Direction:     wt.determineDriftDirection(baseline.Mean, currentMean),
		DetectedAt:    time.Now(),
		Metadata: map[string]interface{}{
			"baseline_mean":  baseline.Mean,
			"current_mean":   currentMean,
			"t_statistic":    tStatistic,
			"baseline_stddev": baseline.StdDev,
			"current_stddev":  currentStdDev,
		},
	}
	
	return result, nil
}

func (wt *WelchTestDetector) Configure(config map[string]interface{}) error {
	if threshold, exists := config["threshold"]; exists {
		if t, ok := threshold.(float64); ok {
			wt.threshold = t
		}
	}
	return nil
}

func (wt *WelchTestDetector) GetMethodType() DriftDetectionMethodType {
	return MethodWelchTest
}

func (wt *WelchTestDetector) Reset() {
	// Nothing to reset for Welch test
}

func (wt *WelchTestDetector) welchTTest(mean1, stddev1 float64, n1 int, mean2, stddev2 float64, n2 int) (tStatistic, pValue float64) {
	if n1 <= 1 || n2 <= 1 {
		return 0, 1.0
	}
	
	var1 := stddev1 * stddev1
	var2 := stddev2 * stddev2
	
	// Calculate standard error
	se := math.Sqrt(var1/float64(n1) + var2/float64(n2))
	
	if se == 0 {
		return 0, 1.0
	}
	
	// Calculate t-statistic
	tStatistic = (mean1 - mean2) / se
	
	// Calculate degrees of freedom (Welch-Satterthwaite equation)
	df := math.Pow(var1/float64(n1)+var2/float64(n2), 2) /
		(math.Pow(var1/float64(n1), 2)/float64(n1-1) + math.Pow(var2/float64(n2), 2)/float64(n2-1))
	
	// Approximate p-value using t-distribution
	pValue = 2.0 * (1.0 - tCDF(math.Abs(tStatistic), int(df)))
	
	return tStatistic, pValue
}

func (wt *WelchTestDetector) determineDriftDirection(baselineMean, currentMean float64) DriftDirection {
	if currentMean > baselineMean*1.05 {
		return DirectionIncrease
	} else if currentMean < baselineMean*0.95 {
		return DirectionDecrease
	}
	return DirectionUnknown
}

// CUSUMDetector implements CUSUM (Cumulative Sum) drift detection
type CUSUMDetector struct {
	threshold   float64
	drift       float64
	sumPositive float64
	sumNegative float64
	reference   float64
}

func NewCUSUMDetector() *CUSUMDetector {
	return &CUSUMDetector{
		threshold: 4.0,
		drift:     0.5,
	}
}

func (cusum *CUSUMDetector) DetectDrift(baseline *MetricBaseline, current []float64) (*DriftResult, error) {
	if len(current) == 0 {
		return nil, fmt.Errorf("no current data provided")
	}
	
	cusum.reference = baseline.Mean
	
	// Reset CUSUM statistics
	cusum.sumPositive = 0
	cusum.sumNegative = 0
	
	maxCusum := 0.0
	changePointIndex := -1
	
	// Calculate CUSUM for current data
	for i, value := range current {
		normalizedValue := (value - cusum.reference) / baseline.StdDev
		
		// Update positive and negative CUSUMs
		cusum.sumPositive = math.Max(0, cusum.sumPositive+normalizedValue-cusum.drift)
		cusum.sumNegative = math.Max(0, cusum.sumNegative-normalizedValue-cusum.drift)
		
		currentMax := math.Max(cusum.sumPositive, cusum.sumNegative)
		if currentMax > maxCusum {
			maxCusum = currentMax
			changePointIndex = i
		}
	}
	
	driftDetected := maxCusum > cusum.threshold
	
	result := &DriftResult{
		Method:        MethodCUSUM,
		MetricName:    baseline.MetricName,
		DriftDetected: driftDetected,
		DriftScore:    maxCusum,
		PValue:        cusum.cusumToPValue(maxCusum),
		Threshold:     cusum.threshold,
		Confidence:    math.Min(0.99, maxCusum/cusum.threshold),
		Direction:     cusum.determineDriftDirection(),
		DetectedAt:    time.Now(),
		Metadata: map[string]interface{}{
			"max_cusum":         maxCusum,
			"positive_cusum":    cusum.sumPositive,
			"negative_cusum":    cusum.sumNegative,
			"change_point":      changePointIndex,
		},
	}
	
	if driftDetected && changePointIndex >= 0 {
		result.ChangePoint = &ChangePoint{
			Index:      changePointIndex,
			Confidence: math.Min(0.95, maxCusum/cusum.threshold),
			Type:       "mean_shift",
		}
	}
	
	return result, nil
}

func (cusum *CUSUMDetector) Configure(config map[string]interface{}) error {
	if threshold, exists := config["threshold"]; exists {
		if t, ok := threshold.(float64); ok {
			cusum.threshold = t
		}
	}
	if drift, exists := config["drift"]; exists {
		if d, ok := drift.(float64); ok {
			cusum.drift = d
		}
	}
	return nil
}

func (cusum *CUSUMDetector) GetMethodType() DriftDetectionMethodType {
	return MethodCUSUM
}

func (cusum *CUSUMDetector) Reset() {
	cusum.sumPositive = 0
	cusum.sumNegative = 0
}

func (cusum *CUSUMDetector) cusumToPValue(cusumValue float64) float64 {
	// Simplified conversion of CUSUM value to p-value
	if cusumValue <= 0 {
		return 1.0
	}
	return math.Exp(-2.0 * cusumValue * cusumValue / (cusumValue + 1))
}

func (cusum *CUSUMDetector) determineDriftDirection() DriftDirection {
	if cusum.sumPositive > cusum.sumNegative {
		return DirectionIncrease
	} else if cusum.sumNegative > cusum.sumPositive {
		return DirectionDecrease
	}
	return DirectionUnknown
}

// EWMADetector implements Exponentially Weighted Moving Average drift detection
type EWMADetector struct {
	threshold float64
	lambda    float64
	ewma      float64
	variance  float64
	initialized bool
}

func NewEWMADetector() *EWMADetector {
	return &EWMADetector{
		threshold: 3.0,
		lambda:    0.2,
	}
}

func (ewma *EWMADetector) DetectDrift(baseline *MetricBaseline, current []float64) (*DriftResult, error) {
	if len(current) == 0 {
		return nil, fmt.Errorf("no current data provided")
	}
	
	if !ewma.initialized {
		ewma.ewma = baseline.Mean
		ewma.variance = baseline.StdDev * baseline.StdDev
		ewma.initialized = true
	}
	
	maxDeviation := 0.0
	changePointIndex := -1
	
	// Process each data point
	for i, value := range current {
		// Update EWMA
		ewma.ewma = ewma.lambda*value + (1-ewma.lambda)*ewma.ewma
		
		// Update variance estimate
		deviation := value - ewma.ewma
		ewma.variance = ewma.lambda*(deviation*deviation) + (1-ewma.lambda)*ewma.variance
		
		// Calculate standardized deviation
		if ewma.variance > 0 {
			standardizedDev := math.Abs(deviation) / math.Sqrt(ewma.variance)
			if standardizedDev > maxDeviation {
				maxDeviation = standardizedDev
				changePointIndex = i
			}
		}
	}
	
	driftDetected := maxDeviation > ewma.threshold
	
	result := &DriftResult{
		Method:        MethodEWMA,
		MetricName:    baseline.MetricName,
		DriftDetected: driftDetected,
		DriftScore:    maxDeviation,
		PValue:        ewma.deviationToPValue(maxDeviation),
		Threshold:     ewma.threshold,
		Confidence:    math.Min(0.99, maxDeviation/ewma.threshold),
		Direction:     ewma.determineDriftDirection(baseline.Mean),
		DetectedAt:    time.Now(),
		Metadata: map[string]interface{}{
			"ewma_value":      ewma.ewma,
			"ewma_variance":   ewma.variance,
			"max_deviation":   maxDeviation,
			"lambda":          ewma.lambda,
		},
	}
	
	if driftDetected && changePointIndex >= 0 {
		result.ChangePoint = &ChangePoint{
			Index:      changePointIndex,
			Confidence: math.Min(0.95, maxDeviation/ewma.threshold),
			Type:       "variance_change",
		}
	}
	
	return result, nil
}

func (ewma *EWMADetector) Configure(config map[string]interface{}) error {
	if threshold, exists := config["threshold"]; exists {
		if t, ok := threshold.(float64); ok {
			ewma.threshold = t
		}
	}
	if lambda, exists := config["lambda"]; exists {
		if l, ok := lambda.(float64); ok {
			ewma.lambda = l
		}
	}
	return nil
}

func (ewma *EWMADetector) GetMethodType() DriftDetectionMethodType {
	return MethodEWMA
}

func (ewma *EWMADetector) Reset() {
	ewma.ewma = 0
	ewma.variance = 0
	ewma.initialized = false
}

func (ewma *EWMADetector) deviationToPValue(deviation float64) float64 {
	// Convert standardized deviation to p-value using normal distribution
	return 2.0 * (1.0 - normalCDF(deviation))
}

func (ewma *EWMADetector) determineDriftDirection(baselineMean float64) DriftDirection {
	if ewma.ewma > baselineMean*1.05 {
		return DirectionIncrease
	} else if ewma.ewma < baselineMean*0.95 {
		return DirectionDecrease
	}
	return DirectionUnknown
}

// PageHinkleyDetector implements Page-Hinkley test for drift detection
type PageHinkleyDetector struct {
	threshold float64
	delta     float64
	lambda    float64
	sum       float64
	min       float64
	max       float64
}

func NewPageHinkleyDetector() *PageHinkleyDetector {
	return &PageHinkleyDetector{
		threshold: 10.0,
		delta:     0.005,
		lambda:    50.0,
	}
}

func (ph *PageHinkleyDetector) DetectDrift(baseline *MetricBaseline, current []float64) (*DriftResult, error) {
	if len(current) == 0 {
		return nil, fmt.Errorf("no current data provided")
	}
	
	ph.Reset()
	
	maxDifference := 0.0
	changePointIndex := -1
	
	// Process each data point
	for i, value := range current {
		// Normalize value
		normalizedValue := (value - baseline.Mean) / baseline.StdDev
		
		// Update sum
		ph.sum += normalizedValue - ph.delta
		
		// Update min and max
		if ph.sum < ph.min {
			ph.min = ph.sum
		}
		if ph.sum > ph.max {
			ph.max = ph.sum
		}
		
		// Check for drift
		upwardDrift := ph.sum - ph.min
		downwardDrift := ph.max - ph.sum
		
		currentMax := math.Max(upwardDrift, downwardDrift)
		if currentMax > maxDifference {
			maxDifference = currentMax
			changePointIndex = i
		}
	}
	
	driftDetected := maxDifference > ph.threshold
	
	result := &DriftResult{
		Method:        MethodPageHinkley,
		MetricName:    baseline.MetricName,
		DriftDetected: driftDetected,
		DriftScore:    maxDifference,
		PValue:        ph.scoreToPValue(maxDifference),
		Threshold:     ph.threshold,
		Confidence:    math.Min(0.99, maxDifference/ph.threshold),
		Direction:     ph.determineDriftDirection(),
		DetectedAt:    time.Now(),
		Metadata: map[string]interface{}{
			"sum":         ph.sum,
			"min":         ph.min,
			"max":         ph.max,
			"delta":       ph.delta,
			"lambda":      ph.lambda,
		},
	}
	
	if driftDetected && changePointIndex >= 0 {
		result.ChangePoint = &ChangePoint{
			Index:      changePointIndex,
			Confidence: math.Min(0.95, maxDifference/ph.threshold),
			Type:       "distribution_change",
		}
	}
	
	return result, nil
}

func (ph *PageHinkleyDetector) Configure(config map[string]interface{}) error {
	if threshold, exists := config["threshold"]; exists {
		if t, ok := threshold.(float64); ok {
			ph.threshold = t
		}
	}
	if delta, exists := config["delta"]; exists {
		if d, ok := delta.(float64); ok {
			ph.delta = d
		}
	}
	if lambda, exists := config["lambda"]; exists {
		if l, ok := lambda.(float64); ok {
			ph.lambda = l
		}
	}
	return nil
}

func (ph *PageHinkleyDetector) GetMethodType() DriftDetectionMethodType {
	return MethodPageHinkley
}

func (ph *PageHinkleyDetector) Reset() {
	ph.sum = 0
	ph.min = 0
	ph.max = 0
}

func (ph *PageHinkleyDetector) scoreToPValue(score float64) float64 {
	// Simplified conversion of Page-Hinkley score to p-value
	if score <= 0 {
		return 1.0
	}
	return math.Exp(-score/ph.lambda)
}

func (ph *PageHinkleyDetector) determineDriftDirection() DriftDirection {
	upwardDrift := ph.sum - ph.min
	downwardDrift := ph.max - ph.sum
	
	if upwardDrift > downwardDrift {
		return DirectionIncrease
	} else if downwardDrift > upwardDrift {
		return DirectionDecrease
	}
	return DirectionBidirectional
}

// Additional utility functions for drift detection

func normalCDF(x float64) float64 {
	// Approximation of normal CDF using error function
	return 0.5 * (1 + math.Erf(x/math.Sqrt(2)))
}

func tCDF(t float64, df int) float64 {
	// Approximation of t-distribution CDF
	// For large df, approximate with normal distribution
	if df > 30 {
		return normalCDF(t)
	}
	
	// Simplified approximation for smaller df
	x := t / math.Sqrt(float64(df))
	return 0.5 + math.Atan(x)/math.Pi
}

// Random number generation for synthetic data
var (
	randSeed = time.Now().UnixNano()
	randState = randSeed
	stored bool
	spare float64
)

// Simple linear congruential generator for consistent results
func randFloat64() float64 {
	randState = (randState*1103515245 + 12345) & 0x7fffffff
	return float64(randState) / 0x7fffffff
}

func randNormFloat64() float64 {
	// Box-Muller transform for normal distribution
	if stored {
		stored = false
		return spare
	}
	
	stored = true
	u := randFloat64()
	v := randFloat64()
	mag := math.Sqrt(-2.0 * math.Log(u))
	spare = mag * math.Cos(2.0*math.Pi*v)
	return mag * math.Sin(2.0*math.Pi*v)
}