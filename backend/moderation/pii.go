package moderation

import (
	"context"
	"fmt"
	"qt1-middleware/opik"
	"regexp"
	"strings"
	"sync"
	"time"
)

// PIIPattern represents a pattern for detecting PII
type PIIPattern struct {
	Type        string         `json:"type"`
	Pattern     *regexp.Regexp `json:"-"`
	PatternStr  string         `json:"pattern"`
	Confidence  float64        `json:"confidence"`
	MaskFormat  string         `json:"mask_format"`
}

// PIIDetector implements PII detection moderation
type PIIDetector struct {
	config       LayerConfig
	opikClient   *opik.OpikClient
	patterns     []PIIPattern
	maskingEnabled bool
	mu           sync.RWMutex
}

// PIIDetection represents a single PII detection
type PIIDetection struct {
	Type        string  `json:"type"`
	Value       string  `json:"value"`
	MaskedValue string  `json:"masked_value"`
	StartIndex  int     `json:"start_index"`
	EndIndex    int     `json:"end_index"`
	Confidence  float64 `json:"confidence"`
}

// NewPIIDetector creates a new PII detector
func NewPIIDetector(config LayerConfig, opikClient *opik.OpikClient) (*PIIDetector, error) {
	pd := &PIIDetector{
		config:       config,
		opikClient:   opikClient,
		patterns:     make([]PIIPattern, 0),
		maskingEnabled: true,
	}

	// Check if masking is enabled in config
	if enabled, ok := config.Options["masking_enabled"].(bool); ok {
		pd.maskingEnabled = enabled
	}

	// Initialize patterns
	if err := pd.initializePatterns(); err != nil {
		return nil, err
	}

	return pd, nil
}

// Name returns the name of this moderation layer
func (pd *PIIDetector) Name() string {
	return "pii"
}

// Weight returns the weight of this layer
func (pd *PIIDetector) Weight() float64 {
	return pd.config.Weight
}

// Enabled returns whether this layer is enabled
func (pd *PIIDetector) Enabled() bool {
	return pd.config.Enabled
}

// Config returns the layer configuration
func (pd *PIIDetector) Config() LayerConfig {
	return pd.config
}

// Moderate performs PII detection
func (pd *PIIDetector) Moderate(content string, context ModerationContext) ModerationResult {
	startTime := time.Now()

	result := ModerationResult{
		Score:      0.0,
		Confidence: 1.0,
		Blocked:    false,
		LayerName:  pd.Name(),
		Details:    make(map[string]interface{}),
	}

	// Detect PII
	detections := pd.detectPII(content)

	// Calculate score based on detections
	if len(detections) > 0 {
		// Score based on number and types of PII found
		result.Score = pd.calculatePIIScore(detections)
		result.Category = CategoryPII
		result.Reason = fmt.Sprintf("Found %d PII items", len(detections))
		
		// Mask PII if enabled
		maskedContent := content
		if pd.maskingEnabled {
			maskedContent = pd.maskPII(content, detections)
		}
		
		result.Details["pii_found"] = true
		result.Details["pii_count"] = len(detections)
		result.Details["pii_types"] = pd.getPIITypes(detections)
		result.Details["masked_content"] = maskedContent
		result.Details["detections"] = detections
	} else {
		result.Details["pii_found"] = false
		result.Details["pii_count"] = 0
	}

	// Determine if content should be blocked
	threshold := pd.config.Threshold
	if threshold == 0 {
		threshold = 0.6 // Default threshold for PII
	}
	result.Blocked = result.Score >= threshold

	result.ProcessTime = time.Since(startTime)
	return result
}

// DetectWithTracing performs PII detection with Opik tracing
func (pd *PIIDetector) DetectWithTracing(ctx context.Context, content string, span *opik.Span) (*PIIResult, error) {
	defer span.End()

	startTime := time.Now()

	// Detect PII
	detections := pd.detectPII(content)

	// Create result
	result := &PIIResult{
		PIIFound:   len(detections) > 0,
		PIITypes:   pd.getPIITypes(detections),
		PIICount:   len(detections),
		Detections: detections,
		Confidence: pd.calculateAverageConfidence(detections),
		ProcessedAt: time.Now(),
	}

	// Mask PII if enabled and found
	if pd.maskingEnabled && len(detections) > 0 {
		result.MaskedText = pd.maskPII(content, detections)
	}

	// Set span output
	span.SetOutput(map[string]interface{}{
		"pii_found":     result.PIIFound,
		"pii_types":     result.PIITypes,
		"pii_count":     result.PIICount,
		"masked_output": result.MaskedText,
		"confidence":    result.Confidence,
	})

	// Set span metadata
	span.SetMetadata(map[string]interface{}{
		"duration_ms":     time.Since(startTime).Milliseconds(),
		"pattern_count":   len(pd.patterns),
		"content_length":  len(content),
		"masking_enabled": pd.maskingEnabled,
	})

	// Calculate and set detection accuracy
	accuracy := pd.calculateAccuracy(detections)
	span.SetScore("pii_detection_accuracy", accuracy)

	return result, nil
}

// initializePatterns initializes PII detection patterns
func (pd *PIIDetector) initializePatterns() error {
	patterns := []struct {
		Type       string
		Pattern    string
		Confidence float64
		MaskFormat string
	}{
		// Email addresses
		{
			Type:       "email",
			Pattern:    `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`,
			Confidence: 0.95,
			MaskFormat: "***@***",
		},
		// Phone numbers (US format)
		{
			Type:       "phone",
			Pattern:    `\b(?:\+?1[-.]?)?\(?([0-9]{3})\)?[-.]?([0-9]{3})[-.]?([0-9]{4})\b`,
			Confidence: 0.90,
			MaskFormat: "***-***-****",
		},
		// Social Security Numbers
		{
			Type:       "ssn",
			Pattern:    `\b(?!000|666|9\d{2})\d{3}[-\s]?(?!00)\d{2}[-\s]?(?!0000)\d{4}\b`,
			Confidence: 0.98,
			MaskFormat: "***-**-****",
		},
		// Credit Card Numbers (basic pattern)
		{
			Type:       "credit_card",
			Pattern:    `\b(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}|3[47][0-9]{13}|3(?:0[0-5]|[68][0-9])[0-9]{11}|6(?:011|5[0-9]{2})[0-9]{12})\b`,
			Confidence: 0.95,
			MaskFormat: "****-****-****-****",
		},
		// IP Addresses
		{
			Type:       "ip_address",
			Pattern:    `\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b`,
			Confidence: 0.85,
			MaskFormat: "***.***.***.***",
		},
		// Date of Birth (various formats)
		{
			Type:       "dob",
			Pattern:    `\b(?:0[1-9]|1[0-2])[-/](?:0[1-9]|[12]\d|3[01])[-/](?:19|20)\d{2}\b`,
			Confidence: 0.70,
			MaskFormat: "**/**/****",
		},
		// Driver's License (generic pattern)
		{
			Type:       "drivers_license",
			Pattern:    `\b[A-Z]{1,2}\d{6,8}\b`,
			Confidence: 0.60,
			MaskFormat: "**********",
		},
		// Passport Number (generic pattern)
		{
			Type:       "passport",
			Pattern:    `\b[A-Z][0-9]{8}\b`,
			Confidence: 0.65,
			MaskFormat: "*********",
		},
	}

	// Load config-specific settings
	configuredPatterns := pd.loadConfigPatterns()

	// Compile patterns
	for _, p := range patterns {
		// Check if this pattern type is enabled in config
		if pd.isPatternEnabled(p.Type) {
			confidence := p.Confidence
			// Override confidence from config if available
			if conf := pd.getConfiguredConfidence(p.Type); conf > 0 {
				confidence = conf
			}

			compiled, err := regexp.Compile(p.Pattern)
			if err != nil {
				return fmt.Errorf("failed to compile PII pattern for %s: %w", p.Type, err)
			}

			pd.patterns = append(pd.patterns, PIIPattern{
				Type:        p.Type,
				Pattern:     compiled,
				PatternStr:  p.Pattern,
				Confidence:  confidence,
				MaskFormat:  p.MaskFormat,
			})
		}
	}

	// Add custom patterns from config
	pd.patterns = append(pd.patterns, configuredPatterns...)

	return nil
}

// loadConfigPatterns loads custom patterns from configuration
func (pd *PIIDetector) loadConfigPatterns() []PIIPattern {
	patterns := make([]PIIPattern, 0)

	if customPatterns, ok := pd.config.Options["custom_patterns"].([]interface{}); ok {
		for _, pattern := range customPatterns {
			if patternMap, ok := pattern.(map[string]interface{}); ok {
				patternStr := patternMap["pattern"].(string)
				compiled, err := regexp.Compile(patternStr)
				if err != nil {
					continue // Skip invalid patterns
				}

				patterns = append(patterns, PIIPattern{
					Type:        patternMap["type"].(string),
					Pattern:     compiled,
					PatternStr:  patternStr,
					Confidence:  patternMap["confidence"].(float64),
					MaskFormat:  patternMap["mask_format"].(string),
				})
			}
		}
	}

	return patterns
}

// isPatternEnabled checks if a pattern type is enabled in config
func (pd *PIIDetector) isPatternEnabled(patternType string) bool {
	key := fmt.Sprintf("detect_%s", patternType)
	if enabled, ok := pd.config.Options[key].(bool); ok {
		return enabled
	}
	return true // Enabled by default
}

// getConfiguredConfidence gets the configured confidence for a pattern type
func (pd *PIIDetector) getConfiguredConfidence(patternType string) float64 {
	key := fmt.Sprintf("%s_confidence", patternType)
	if conf, ok := pd.config.Options[key].(float64); ok {
		return conf
	}
	return 0
}

// detectPII detects PII in content
func (pd *PIIDetector) detectPII(content string) []PIIDetection {
	pd.mu.RLock()
	defer pd.mu.RUnlock()

	detections := make([]PIIDetection, 0)

	for _, pattern := range pd.patterns {
		matches := pattern.Pattern.FindAllStringSubmatchIndex(content, -1)
		for _, match := range matches {
			if len(match) >= 2 {
				value := content[match[0]:match[1]]
				maskedValue := pd.maskValue(value, pattern.MaskFormat)

				detections = append(detections, PIIDetection{
					Type:        pattern.Type,
					Value:       value,
					MaskedValue: maskedValue,
					StartIndex:  match[0],
					EndIndex:    match[1],
					Confidence:  pattern.Confidence,
				})
			}
		}
	}

	return detections
}

// maskPII masks PII in content
func (pd *PIIDetector) maskPII(content string, detections []PIIDetection) string {
	// Sort detections by start index in reverse order
	// This ensures we replace from end to start to maintain indices
	for i := 0; i < len(detections); i++ {
		for j := i + 1; j < len(detections); j++ {
			if detections[i].StartIndex < detections[j].StartIndex {
				detections[i], detections[j] = detections[j], detections[i]
			}
		}
	}

	masked := content
	for _, detection := range detections {
		masked = masked[:detection.StartIndex] + detection.MaskedValue + masked[detection.EndIndex:]
	}

	return masked
}

// maskValue masks a value according to the format
func (pd *PIIDetector) maskValue(value, format string) string {
	if format == "" {
		// Default masking: replace with asterisks
		return strings.Repeat("*", len(value))
	}
	return format
}

// getPIITypes extracts unique PII types from detections
func (pd *PIIDetector) getPIITypes(detections []PIIDetection) []string {
	typeMap := make(map[string]bool)
	for _, d := range detections {
		typeMap[d.Type] = true
	}

	types := make([]string, 0, len(typeMap))
	for t := range typeMap {
		types = append(types, t)
	}

	return types
}

// calculatePIIScore calculates a score based on PII detections
func (pd *PIIDetector) calculatePIIScore(detections []PIIDetection) float64 {
	if len(detections) == 0 {
		return 0.0
	}

	// Weights for different PII types
	weights := map[string]float64{
		"ssn":             1.0,
		"credit_card":     0.95,
		"passport":        0.9,
		"drivers_license": 0.85,
		"phone":           0.7,
		"email":           0.6,
		"dob":             0.6,
		"ip_address":      0.5,
	}

	totalScore := 0.0
	for _, d := range detections {
		weight := 0.5 // Default weight
		if w, ok := weights[d.Type]; ok {
			weight = w
		}
		totalScore += weight * d.Confidence
	}

	// Normalize score
	score := totalScore / float64(len(detections))
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// calculateAverageConfidence calculates average confidence of detections
func (pd *PIIDetector) calculateAverageConfidence(detections []PIIDetection) float64 {
	if len(detections) == 0 {
		return 1.0 // No PII found with high confidence
	}

	total := 0.0
	for _, d := range detections {
		total += d.Confidence
	}

	return total / float64(len(detections))
}

// calculateAccuracy estimates detection accuracy
func (pd *PIIDetector) calculateAccuracy(detections []PIIDetection) float64 {
	if len(detections) == 0 {
		return 1.0 // No false positives
	}

	// Simple accuracy based on confidence levels
	return pd.calculateAverageConfidence(detections)
}

// GetStats returns statistics about the PII detector
func (pd *PIIDetector) GetStats() map[string]interface{} {
	pd.mu.RLock()
	defer pd.mu.RUnlock()

	return map[string]interface{}{
		"pattern_count":   len(pd.patterns),
		"enabled":         pd.Enabled(),
		"weight":          pd.Weight(),
		"threshold":       pd.config.Threshold,
		"masking_enabled": pd.maskingEnabled,
	}
}