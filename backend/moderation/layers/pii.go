package layers

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"qt1-middleware/moderation"
)

// PIILayer implements PII (Personally Identifiable Information) detection
type PIILayer struct {
	name           string
	enabled        bool
	weight         float64
	threshold      float64
	detectors      map[string]*PIIDetector
	maskingEnabled bool
	options        map[string]interface{}
}

// PIIDetector represents a specific PII detection rule
type PIIDetector struct {
	Pattern     *regexp.Regexp
	Validator   func(string) bool
	Confidence  float64
	PIIType     string
	Description string
}

// PIIMatch represents a detected PII item
type PIIMatch struct {
	Type       string  `json:"type"`
	Value      string  `json:"value,omitempty"`
	MaskedValue string `json:"masked_value,omitempty"`
	Position   []int   `json:"position"`
	Confidence float64 `json:"confidence"`
}

// PII type constants
const (
	PIITypeEmail      = "email"
	PIITypePhone      = "phone"
	PIITypeSSN        = "ssn"
	PIITypeCreditCard = "credit_card"
	PIITypeCustom     = "custom"
)

// Common PII regex patterns
var (
	// Email pattern (RFC 5322 simplified)
	emailPattern = regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`)
	
	// Phone patterns (US and international)
	phonePatterns = []*regexp.Regexp{
		regexp.MustCompile(`\b(?:\+1[-.\s]?)?(?:\(?[0-9]{3}\)?[-.\s]?[0-9]{3}[-.\s]?[0-9]{4})\b`), // US format
		regexp.MustCompile(`\b(?:\+[1-9]\d{0,3}[-.\s]?)?(?:\(?[0-9]{1,4}\)?[-.\s]?[0-9]{1,4}[-.\s]?[0-9]{1,9})\b`), // International
	}
	
	// SSN patterns (various formats)
	ssnPatterns = []*regexp.Regexp{
		regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),        // 123-45-6789
		regexp.MustCompile(`\b\d{3}\s\d{2}\s\d{4}\b`),      // 123 45 6789
		regexp.MustCompile(`\b\d{9}\b`),                     // 123456789 (only if context suggests SSN)
	}
	
	// Credit card patterns (major card types)
	creditCardPatterns = []*regexp.Regexp{
		regexp.MustCompile(`\b(?:4[0-9]{12}(?:[0-9]{3})?)\b`),                          // Visa
		regexp.MustCompile(`\b(?:5[1-5][0-9]{14})\b`),                                  // MasterCard
		regexp.MustCompile(`\b(?:3[47][0-9]{13})\b`),                                   // American Express
		regexp.MustCompile(`\b(?:3[0-9]{13})\b`),                                       // Diners Club
		regexp.MustCompile(`\b(?:6(?:011|5[0-9]{2})[0-9]{12})\b`),                     // Discover
		regexp.MustCompile(`\b(?:[0-9]{4}[-\s]?[0-9]{4}[-\s]?[0-9]{4}[-\s]?[0-9]{4})\b`), // Generic format
	}
)

// NewPIILayer creates a new PII detection layer
func NewPIILayer(config moderation.LayerConfig) *PIILayer {
	layer := &PIILayer{
		name:           config.Name,
		enabled:        config.Enabled,
		weight:         config.Weight,
		threshold:      config.Threshold,
		detectors:      make(map[string]*PIIDetector),
		maskingEnabled: false,
		options:        config.Options,
	}
	
	// Configure masking if specified in options
	if mask, ok := config.Options["masking_enabled"]; ok {
		if maskBool, ok := mask.(bool); ok {
			layer.maskingEnabled = maskBool
		}
	}
	
	// Initialize default detectors
	layer.initializeDetectors()
	
	return layer
}

// initializeDetectors sets up the default PII detectors
func (p *PIILayer) initializeDetectors() {
	// Email detector
	p.detectors[PIITypeEmail] = &PIIDetector{
		Pattern:     emailPattern,
		Validator:   validateEmail,
		Confidence:  0.95,
		PIIType:     PIITypeEmail,
		Description: "Email address detection",
	}
	
	// Phone detector (combining multiple patterns)
	p.detectors[PIITypePhone] = &PIIDetector{
		Pattern:     phonePatterns[0], // Primary US pattern
		Validator:   validatePhone,
		Confidence:  0.90,
		PIIType:     PIITypePhone,
		Description: "Phone number detection",
	}
	
	// SSN detector
	p.detectors[PIITypeSSN] = &PIIDetector{
		Pattern:     ssnPatterns[0], // Primary SSN pattern
		Validator:   validateSSN,
		Confidence:  0.98,
		PIIType:     PIITypeSSN,
		Description: "Social Security Number detection",
	}
	
	// Credit card detector
	p.detectors[PIITypeCreditCard] = &PIIDetector{
		Pattern:     creditCardPatterns[5], // Generic credit card pattern
		Validator:   validateCreditCard,
		Confidence:  0.95,
		PIIType:     PIITypeCreditCard,
		Description: "Credit card number detection",
	}
}

// Name returns the layer name
func (p *PIILayer) Name() string {
	return p.name
}

// Weight returns the layer weight
func (p *PIILayer) Weight() float64 {
	return p.weight
}

// Enabled returns whether the layer is enabled
func (p *PIILayer) Enabled() bool {
	return p.enabled
}

// Config returns the layer configuration
func (p *PIILayer) Config() moderation.LayerConfig {
	return moderation.LayerConfig{
		Name:      p.name,
		Enabled:   p.enabled,
		Weight:    p.weight,
		Threshold: p.threshold,
		Options:   p.options,
	}
}

// Moderate performs PII detection on the given content
func (p *PIILayer) Moderate(content string, context moderation.ModerationContext) moderation.ModerationResult {
	startTime := time.Now()
	
	if !p.enabled {
		return moderation.ModerationResult{
			Score:       0.0,
			Confidence:  1.0,
			Blocked:     false,
			Reason:      "PII layer disabled",
			Category:    moderation.CategoryPII,
			LayerName:   p.name,
			Details:     map[string]interface{}{},
			ProcessTime: time.Since(startTime),
		}
	}
	
	matches := make([]PIIMatch, 0)
	maxScore := 0.0
	detectedTypes := make([]string, 0)
	
	// Run detection for each PII type
	for piiType, detector := range p.detectors {
		typeMatches := p.detectPIIType(content, piiType, detector)
		matches = append(matches, typeMatches...)
		
		if len(typeMatches) > 0 {
			detectedTypes = append(detectedTypes, piiType)
			if detector.Confidence > maxScore {
				maxScore = detector.Confidence
			}
		}
	}
	
	// Additional phone number detection with multiple patterns
	if p.detectors[PIITypePhone] != nil {
		for _, pattern := range phonePatterns[1:] {
			additionalMatches := p.detectWithPattern(content, PIITypePhone, pattern, p.detectors[PIITypePhone])
			matches = append(matches, additionalMatches...)
		}
	}
	
	// Additional SSN detection with multiple patterns
	if p.detectors[PIITypeSSN] != nil {
		for _, pattern := range ssnPatterns[1:] {
			additionalMatches := p.detectWithPattern(content, PIITypeSSN, pattern, p.detectors[PIITypeSSN])
			matches = append(matches, additionalMatches...)
		}
	}
	
	// Additional credit card detection with specific card patterns
	if p.detectors[PIITypeCreditCard] != nil {
		for _, pattern := range creditCardPatterns[:5] {
			additionalMatches := p.detectWithPattern(content, PIITypeCreditCard, pattern, p.detectors[PIITypeCreditCard])
			matches = append(matches, additionalMatches...)
		}
	}
	
	// Calculate final score based on matches and confidence
	finalScore := p.calculateScore(matches, maxScore)
	blocked := finalScore >= p.threshold
	
	// Generate reason
	reason := p.generateReason(detectedTypes, len(matches))
	
	// Prepare details
	details := map[string]interface{}{
		"pii_matches_count": len(matches),
		"detected_types":    detectedTypes,
		"masking_enabled":   p.maskingEnabled,
	}
	
	// Add matches to details (masked if enabled)
	if len(matches) > 0 {
		if p.maskingEnabled {
			maskedMatches := make([]PIIMatch, len(matches))
			for i, match := range matches {
				maskedMatches[i] = match
				maskedMatches[i].Value = "" // Remove actual value
			}
			details["matches"] = maskedMatches
		} else {
			details["matches"] = matches
		}
	}
	
	return moderation.ModerationResult{
		Score:       finalScore,
		Confidence:  maxScore,
		Blocked:     blocked,
		Reason:      reason,
		Category:    moderation.CategoryPII,
		LayerName:   p.name,
		Details:     details,
		ProcessTime: time.Since(startTime),
	}
}

// detectPIIType detects a specific type of PII in the content
func (p *PIILayer) detectPIIType(content string, piiType string, detector *PIIDetector) []PIIMatch {
	return p.detectWithPattern(content, piiType, detector.Pattern, detector)
}

// detectWithPattern detects PII using a specific pattern
func (p *PIILayer) detectWithPattern(content string, piiType string, pattern *regexp.Regexp, detector *PIIDetector) []PIIMatch {
	matches := make([]PIIMatch, 0)
	
	// Find all matches
	allMatches := pattern.FindAllStringSubmatch(content, -1)
	allIndices := pattern.FindAllStringIndex(content, -1)
	
	for i, match := range allMatches {
		if len(match) > 0 {
			value := match[0]
			
			// Validate the match if validator exists
			if detector.Validator != nil && !detector.Validator(value) {
				continue
			}
			
			// Create PII match
			piiMatch := PIIMatch{
				Type:       piiType,
				Value:      value,
				Position:   allIndices[i],
				Confidence: detector.Confidence,
			}
			
			// Add masked value if masking is enabled
			if p.maskingEnabled {
				piiMatch.MaskedValue = p.maskValue(value, piiType)
			}
			
			matches = append(matches, piiMatch)
		}
	}
	
	return matches
}

// calculateScore calculates the final score based on detected PII
func (p *PIILayer) calculateScore(matches []PIIMatch, maxConfidence float64) float64 {
	if len(matches) == 0 {
		return 0.0
	}
	
	// Base score is the highest confidence
	score := maxConfidence
	
	// Increase score based on number of matches (with diminishing returns)
	matchBonus := float64(len(matches)) * 0.1
	if matchBonus > 0.3 {
		matchBonus = 0.3
	}
	
	score += matchBonus
	
	// Cap at 1.0
	if score > 1.0 {
		score = 1.0
	}
	
	return score
}

// generateReason generates a human-readable reason for the detection
func (p *PIILayer) generateReason(detectedTypes []string, matchCount int) string {
	if len(detectedTypes) == 0 {
		return "No PII detected"
	}
	
	if len(detectedTypes) == 1 {
		return fmt.Sprintf("Detected %s (%d matches)", detectedTypes[0], matchCount)
	}
	
	return fmt.Sprintf("Detected multiple PII types: %s (%d total matches)", 
		strings.Join(detectedTypes, ", "), matchCount)
}

// maskValue masks a PII value based on its type
func (p *PIILayer) maskValue(value string, piiType string) string {
	switch piiType {
	case PIITypeEmail:
		parts := strings.Split(value, "@")
		if len(parts) == 2 {
			username := parts[0]
			domain := parts[1]
			if len(username) > 2 {
				maskedUsername := username[:1] + strings.Repeat("*", len(username)-2) + username[len(username)-1:]
				return maskedUsername + "@" + domain
			}
		}
		return strings.Repeat("*", len(value))
		
	case PIITypePhone:
		if len(value) >= 4 {
			return strings.Repeat("*", len(value)-4) + value[len(value)-4:]
		}
		return strings.Repeat("*", len(value))
		
	case PIITypeSSN:
		if strings.Contains(value, "-") {
			parts := strings.Split(value, "-")
			if len(parts) == 3 {
				return "***-**-" + parts[2]
			}
		}
		if len(value) >= 4 {
			return strings.Repeat("*", len(value)-4) + value[len(value)-4:]
		}
		return strings.Repeat("*", len(value))
		
	case PIITypeCreditCard:
		cleaned := strings.ReplaceAll(strings.ReplaceAll(value, "-", ""), " ", "")
		if len(cleaned) >= 4 {
			masked := strings.Repeat("*", len(cleaned)-4) + cleaned[len(cleaned)-4:]
			if strings.Contains(value, "-") {
				// Format with dashes
				result := ""
				for i, char := range masked {
					if i > 0 && i%4 == 0 {
						result += "-"
					}
					result += string(char)
				}
				return result
			} else if strings.Contains(value, " ") {
				// Format with spaces
				result := ""
				for i, char := range masked {
					if i > 0 && i%4 == 0 {
						result += " "
					}
					result += string(char)
				}
				return result
			}
			return masked
		}
		return strings.Repeat("*", len(value))
		
	default:
		return strings.Repeat("*", len(value))
	}
}

// Validator functions

// validateEmail validates an email address
func validateEmail(email string) bool {
	// Basic validation - ensure it has @ and domain
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	
	username := parts[0]
	domain := parts[1]
	
	// Check basic requirements
	if len(username) == 0 || len(domain) == 0 {
		return false
	}
	
	// Domain should have at least one dot
	if !strings.Contains(domain, ".") {
		return false
	}
	
	// Domain should not start or end with dot or dash
	if strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") ||
		strings.HasPrefix(domain, "-") || strings.HasSuffix(domain, "-") {
		return false
	}
	
	return true
}

// validatePhone validates a phone number
func validatePhone(phone string) bool {
	// Remove formatting
	cleaned := regexp.MustCompile(`[^\d+]`).ReplaceAllString(phone, "")
	
	// Check length (minimum 7 digits, maximum 15)
	if len(cleaned) < 7 || len(cleaned) > 15 {
		return false
	}
	
	// If it starts with +, ensure it's followed by digits
	if strings.HasPrefix(phone, "+") && len(cleaned) < 8 {
		return false
	}
	
	return true
}

// validateSSN validates a Social Security Number
func validateSSN(ssn string) bool {
	// Remove formatting
	cleaned := regexp.MustCompile(`[^\d]`).ReplaceAllString(ssn, "")
	
	// Must be exactly 9 digits
	if len(cleaned) != 9 {
		return false
	}
	
	// Check for invalid patterns
	invalidPatterns := []string{
		"000000000", "111111111", "222222222", "333333333", "444444444",
		"555555555", "666666666", "777777777", "888888888", "999999999",
		"123456789", "987654321",
	}
	
	for _, invalid := range invalidPatterns {
		if cleaned == invalid {
			return false
		}
	}
	
	// Area number (first 3 digits) cannot be 000, 666, or 900-999
	areaNumber := cleaned[:3]
	if areaNumber == "000" || areaNumber == "666" {
		return false
	}
	
	if areaNumber[0] == '9' {
		return false
	}
	
	// Group number (middle 2 digits) cannot be 00
	if cleaned[3:5] == "00" {
		return false
	}
	
	// Serial number (last 4 digits) cannot be 0000
	if cleaned[5:9] == "0000" {
		return false
	}
	
	return true
}

// validateCreditCard validates a credit card number using Luhn algorithm
func validateCreditCard(cardNumber string) bool {
	// Remove formatting
	cleaned := regexp.MustCompile(`[^\d]`).ReplaceAllString(cardNumber, "")
	
	// Check length (13-19 digits for most cards)
	if len(cleaned) < 13 || len(cleaned) > 19 {
		return false
	}
	
	// Luhn algorithm validation
	return luhnCheck(cleaned)
}

// luhnCheck performs the Luhn algorithm check
func luhnCheck(cardNumber string) bool {
	var sum int
	isEven := false
	
	// Process digits from right to left
	for i := len(cardNumber) - 1; i >= 0; i-- {
		digit := int(cardNumber[i] - '0')
		
		if isEven {
			digit *= 2
			if digit > 9 {
				digit = digit%10 + digit/10
			}
		}
		
		sum += digit
		isEven = !isEven
	}
	
	return sum%10 == 0
}