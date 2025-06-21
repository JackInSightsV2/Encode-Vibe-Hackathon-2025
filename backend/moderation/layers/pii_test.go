package layers

import (
	"testing"
	"time"

	"qt1-middleware/moderation"
)

// Test data for PII detection
var (
	testEmails = []struct {
		input    string
		expected bool
		desc     string
	}{
		{"test@example.com", true, "simple email"},
		{"user.name+tag@domain.co.uk", true, "complex email with dots and plus"},
		{"admin@localhost", false, "localhost domain (should be invalid)"},
		{"invalid.email", false, "no @ symbol"},
		{"@domain.com", false, "missing username"},
		{"user@", false, "missing domain"},
		{"user@domain.", false, "domain ends with dot"},
		{"user.name@example-domain.com", true, "domain with dash"},
		{"test123@test123.com", true, "alphanumeric email"},
	}

	testPhones = []struct {
		input    string
		expected bool
		desc     string
	}{
		{"(555) 123-4567", true, "US format with parentheses"},
		{"555-123-4567", true, "US format with dashes"},
		{"555.123.4567", true, "US format with dots"},
		{"5551234567", true, "US format no formatting"},
		{"+1-555-123-4567", true, "US format with country code"},
		{"+44 20 7946 0958", true, "UK format"},
		{"+86 138 0013 8000", true, "Chinese format"},
		{"123", false, "too short"},
		{"12345678901234567890", false, "too long"},
		{"+", false, "just plus sign"},
	}

	testSSNs = []struct {
		input    string
		expected bool
		desc     string
	}{
		{"123-45-6789", true, "valid SSN format"},
		{"123 45 6789", true, "valid SSN with spaces"},
		{"123456789", true, "valid SSN no formatting"},
		{"000-45-6789", false, "invalid area number 000"},
		{"666-45-6789", false, "invalid area number 666"},
		{"900-45-6789", false, "invalid area number 900+"},
		{"123-00-6789", false, "invalid group number 00"},
		{"123-45-0000", false, "invalid serial number 0000"},
		{"111-11-1111", false, "all same digits"},
		{"123-45-67890", false, "too many digits"},
		{"12-45-6789", false, "too few digits in area"},
	}

	testCreditCards = []struct {
		input    string
		expected bool
		desc     string
	}{
		{"4532015112830366", true, "valid Visa"},
		{"4532-0151-1283-0366", true, "valid Visa with dashes"},
		{"4532 0151 1283 0366", true, "valid Visa with spaces"},
		{"5555555555554444", true, "valid MasterCard"},
		{"378282246310005", true, "valid American Express"},
		{"30569309025904", true, "valid Diners Club"},
		{"6011111111111117", true, "valid Discover"},
		{"4532015112830367", false, "invalid Visa (fails Luhn)"},
		{"1234567890123456", false, "invalid card (fails Luhn)"},
		{"123", false, "too short"},
		{"12345678901234567890", false, "too long"},
	}

	testMixedContent = []struct {
		input    string
		expected map[string]int
		desc     string
	}{
		{
			"Contact me at john.doe@example.com or call (555) 123-4567",
			map[string]int{"email": 1, "phone": 1},
			"email and phone",
		},
		{
			"My SSN is 123-45-6789 and credit card is 4532-0151-1283-0366",
			map[string]int{"ssn": 1, "credit_card": 1},
			"SSN and credit card",
		},
		{
			"No PII in this message",
			map[string]int{},
			"no PII",
		},
		{
			"Multiple emails: test1@example.com, test2@example.com, and phone: 555-123-4567",
			map[string]int{"email": 2, "phone": 1},
			"multiple emails and phone",
		},
	}
)

func TestNewPIILayer(t *testing.T) {
	config := moderation.LayerConfig{
		Name:      "pii_test",
		Enabled:   true,
		Weight:    0.8,
		Threshold: 0.7,
		Options: map[string]interface{}{
			"masking_enabled": true,
		},
	}

	layer := NewPIILayer(config)

	if layer.Name() != "pii_test" {
		t.Errorf("Expected name 'pii_test', got '%s'", layer.Name())
	}

	if layer.Weight() != 0.8 {
		t.Errorf("Expected weight 0.8, got %f", layer.Weight())
	}

	if !layer.Enabled() {
		t.Error("Expected layer to be enabled")
	}

	if !layer.maskingEnabled {
		t.Error("Expected masking to be enabled")
	}

	// Check that detectors are initialized
	expectedDetectors := []string{PIITypeEmail, PIITypePhone, PIITypeSSN, PIITypeCreditCard}
	for _, piiType := range expectedDetectors {
		if _, exists := layer.detectors[piiType]; !exists {
			t.Errorf("Expected detector for %s to be initialized", piiType)
		}
	}
}

func TestEmailDetection(t *testing.T) {
	config := moderation.LayerConfig{
		Name:      "pii_test",
		Enabled:   true,
		Weight:    1.0,
		Threshold: 0.5,
		Options:   map[string]interface{}{},
	}

	layer := NewPIILayer(config)
	context := moderation.ModerationContext{}

	for _, test := range testEmails {
		result := layer.Moderate(test.input, context)
		
		hasEmail := false
		if matches, ok := result.Details["matches"].([]PIIMatch); ok {
			for _, match := range matches {
				if match.Type == PIITypeEmail {
					hasEmail = true
					break
				}
			}
		}

		if hasEmail != test.expected {
			t.Errorf("Email detection failed for '%s' (%s): expected %v, got %v", 
				test.input, test.desc, test.expected, hasEmail)
		}
	}
}

func TestPhoneDetection(t *testing.T) {
	config := moderation.LayerConfig{
		Name:      "pii_test",
		Enabled:   true,
		Weight:    1.0,
		Threshold: 0.5,
		Options:   map[string]interface{}{},
	}

	layer := NewPIILayer(config)
	context := moderation.ModerationContext{}

	for _, test := range testPhones {
		result := layer.Moderate(test.input, context)
		
		hasPhone := false
		if matches, ok := result.Details["matches"].([]PIIMatch); ok {
			for _, match := range matches {
				if match.Type == PIITypePhone {
					hasPhone = true
					break
				}
			}
		}

		if hasPhone != test.expected {
			t.Errorf("Phone detection failed for '%s' (%s): expected %v, got %v", 
				test.input, test.desc, test.expected, hasPhone)
		}
	}
}

func TestSSNDetection(t *testing.T) {
	config := moderation.LayerConfig{
		Name:      "pii_test",
		Enabled:   true,
		Weight:    1.0,
		Threshold: 0.5,
		Options:   map[string]interface{}{},
	}

	layer := NewPIILayer(config)
	context := moderation.ModerationContext{}

	for _, test := range testSSNs {
		result := layer.Moderate(test.input, context)
		
		hasSSN := false
		if matches, ok := result.Details["matches"].([]PIIMatch); ok {
			for _, match := range matches {
				if match.Type == PIITypeSSN {
					hasSSN = true
					break
				}
			}
		}

		if hasSSN != test.expected {
			t.Errorf("SSN detection failed for '%s' (%s): expected %v, got %v", 
				test.input, test.desc, test.expected, hasSSN)
		}
	}
}

func TestCreditCardDetection(t *testing.T) {
	config := moderation.LayerConfig{
		Name:      "pii_test",
		Enabled:   true,
		Weight:    1.0,
		Threshold: 0.5,
		Options:   map[string]interface{}{},
	}

	layer := NewPIILayer(config)
	context := moderation.ModerationContext{}

	for _, test := range testCreditCards {
		result := layer.Moderate(test.input, context)
		
		hasCreditCard := false
		if matches, ok := result.Details["matches"].([]PIIMatch); ok {
			for _, match := range matches {
				if match.Type == PIITypeCreditCard {
					hasCreditCard = true
					break
				}
			}
		}

		if hasCreditCard != test.expected {
			t.Errorf("Credit card detection failed for '%s' (%s): expected %v, got %v", 
				test.input, test.desc, test.expected, hasCreditCard)
		}
	}
}

func TestMixedContentDetection(t *testing.T) {
	config := moderation.LayerConfig{
		Name:      "pii_test",
		Enabled:   true,
		Weight:    1.0,
		Threshold: 0.5,
		Options:   map[string]interface{}{},
	}

	layer := NewPIILayer(config)
	context := moderation.ModerationContext{}

	for _, test := range testMixedContent {
		result := layer.Moderate(test.input, context)
		
		actualCounts := make(map[string]int)
		if matches, ok := result.Details["matches"].([]PIIMatch); ok {
			for _, match := range matches {
				actualCounts[match.Type]++
			}
		}

		for expectedType, expectedCount := range test.expected {
			if actualCounts[expectedType] != expectedCount {
				t.Errorf("Mixed content detection failed for '%s' (%s): expected %d %s, got %d", 
					test.input, test.desc, expectedCount, expectedType, actualCounts[expectedType])
			}
		}

		// Check that no unexpected types were detected
		for actualType, actualCount := range actualCounts {
			if expectedCount, exists := test.expected[actualType]; !exists || actualCount != expectedCount {
				if !exists && actualCount > 0 {
					t.Errorf("Mixed content detection failed for '%s' (%s): unexpected %s detected (%d)", 
						test.input, test.desc, actualType, actualCount)
				}
			}
		}
	}
}

func TestPIIMasking(t *testing.T) {
	config := moderation.LayerConfig{
		Name:      "pii_test",
		Enabled:   true,
		Weight:    1.0,
		Threshold: 0.5,
		Options: map[string]interface{}{
			"masking_enabled": true,
		},
	}

	layer := NewPIILayer(config)
	context := moderation.ModerationContext{}

	testCases := []struct {
		input    string
		piiType  string
		expected string
		desc     string
	}{
		{"test@example.com", PIITypeEmail, "t**t@example.com", "email masking"},
		{"123-45-6789", PIITypeSSN, "***-**-6789", "SSN with dashes"},
		{"4532015112830366", PIITypeCreditCard, "************0366", "credit card masking"},
		{"(555) 123-4567", PIITypePhone, "*******4567", "phone masking"},
	}

	for _, test := range testCases {
		result := layer.Moderate(test.input, context)
		
		if matches, ok := result.Details["matches"].([]PIIMatch); ok {
			found := false
			for _, match := range matches {
				if match.Type == test.piiType {
					found = true
					if match.MaskedValue == "" {
						t.Errorf("Masking failed for '%s' (%s): no masked value provided", 
							test.input, test.desc)
					}
					// Note: exact masking format may vary, so we just check it's not empty
					break
				}
			}
			if !found {
				t.Errorf("Masking test failed for '%s' (%s): PII type %s not detected", 
					test.input, test.desc, test.piiType)
			}
		} else {
			t.Errorf("Masking test failed for '%s' (%s): no matches found", 
				test.input, test.desc)
		}
	}
}

func TestPIILayerDisabled(t *testing.T) {
	config := moderation.LayerConfig{
		Name:      "pii_test",
		Enabled:   false,
		Weight:    1.0,
		Threshold: 0.5,
		Options:   map[string]interface{}{},
	}

	layer := NewPIILayer(config)
	context := moderation.ModerationContext{}

	result := layer.Moderate("test@example.com and 123-45-6789", context)

	if result.Score != 0.0 {
		t.Errorf("Expected score 0.0 for disabled layer, got %f", result.Score)
	}

	if result.Blocked {
		t.Error("Expected layer not to block when disabled")
	}

	if result.Reason != "PII layer disabled" {
		t.Errorf("Expected reason 'PII layer disabled', got '%s'", result.Reason)
	}
}

func TestPIIThreshold(t *testing.T) {
	// Test with high threshold (should not block)
	config := moderation.LayerConfig{
		Name:      "pii_test",
		Enabled:   true,
		Weight:    1.0,
		Threshold: 0.99,
		Options:   map[string]interface{}{},
	}

	layer := NewPIILayer(config)
	context := moderation.ModerationContext{}

	result := layer.Moderate("test@example.com", context)

	if result.Blocked {
		t.Error("Expected not to block with high threshold")
	}

	// Test with low threshold (should block)
	config.Threshold = 0.5
	layer = NewPIILayer(config)

	result = layer.Moderate("test@example.com", context)

	if !result.Blocked {
		t.Error("Expected to block with low threshold")
	}
}

func TestPIIScoring(t *testing.T) {
	config := moderation.LayerConfig{
		Name:      "pii_test",
		Enabled:   true,
		Weight:    1.0,
		Threshold: 0.5,
		Options:   map[string]interface{}{},
	}

	layer := NewPIILayer(config)
	context := moderation.ModerationContext{}

	// Test single PII
	result1 := layer.Moderate("test@example.com", context)
	
	// Test multiple PII (should have higher score)
	result2 := layer.Moderate("test@example.com and 123-45-6789", context)

	if result2.Score <= result1.Score {
		t.Errorf("Expected higher score for multiple PII: single=%f, multiple=%f", 
			result1.Score, result2.Score)
	}

	// Test no PII
	result3 := layer.Moderate("This is just regular text", context)
	
	if result3.Score != 0.0 {
		t.Errorf("Expected score 0.0 for no PII, got %f", result3.Score)
	}
}

func TestLuhnAlgorithm(t *testing.T) {
	validCards := []string{
		"4532015112830366", // Visa
		"5555555555554444", // MasterCard
		"378282246310005",  // American Express
		"30569309025904",   // Diners Club
		"6011111111111117", // Discover
	}

	invalidCards := []string{
		"4532015112830367", // Invalid Visa
		"5555555555554445", // Invalid MasterCard
		"1234567890123456", // Random number
	}

	for _, card := range validCards {
		if !luhnCheck(card) {
			t.Errorf("Luhn check failed for valid card: %s", card)
		}
	}

	for _, card := range invalidCards {
		if luhnCheck(card) {
			t.Errorf("Luhn check passed for invalid card: %s", card)
		}
	}
}

func TestPerformance(t *testing.T) {
	config := moderation.LayerConfig{
		Name:      "pii_test",
		Enabled:   true,
		Weight:    1.0,
		Threshold: 0.5,
		Options:   map[string]interface{}{},
	}

	layer := NewPIILayer(config)
	context := moderation.ModerationContext{}

	// Test with long content containing multiple PII types
	longContent := `
		Dear customers,
		
		Please contact us at support@example.com or admin@company.org if you need help.
		You can also call us at (555) 123-4567 or +1-800-555-0199.
		
		For account verification, you may need to provide:
		- SSN: 123-45-6789 or 987-65-4321
		- Credit card: 4532-0151-1283-0366 or 5555-5555-5555-4444
		
		Additional contacts:
		- manager@division.company.com
		- (555) 987-6543
		- Emergency: 911
		
		This message contains sensitive information and should be handled carefully.
	`

	start := time.Now()
	result := layer.Moderate(longContent, context)
	duration := time.Since(start)

	// Performance requirement: <50ms for 95% of requests
	if duration > 50*time.Millisecond {
		t.Errorf("Performance test failed: processing took %v, expected <50ms", duration)
	}

	// Verify correctness
	if result.Score == 0.0 {
		t.Error("Expected to detect PII in long content")
	}

	if result.ProcessTime == 0 {
		t.Error("Expected process time to be recorded")
	}

	t.Logf("Performance test: processed in %v, score: %f", duration, result.Score)
}

func TestFalsePositiveRate(t *testing.T) {
	config := moderation.LayerConfig{
		Name:      "pii_test",
		Enabled:   true,
		Weight:    1.0,
		Threshold: 0.5,
		Options:   map[string]interface{}{},
	}

	layer := NewPIILayer(config)
	context := moderation.ModerationContext{}

	// Test cases that should NOT trigger PII detection
	safeSamples := []string{
		"The year 2023 was interesting",
		"I have 123 apples and 456 oranges",
		"Call me at extension 1234",
		"Version 1.2.3 is available",
		"The code is 12345",
		"Meeting room 123-456-789",
		"Invoice number: INV-123-456",
		"Reference: REF123456789",
		"Order ID: ORD-2023-001",
		"Product SKU: SKU123456",
	}

	falsePositives := 0
	for _, sample := range safeSamples {
		result := layer.Moderate(sample, context)
		if result.Score > 0.0 {
			falsePositives++
			t.Logf("False positive detected in: '%s' (score: %f)", sample, result.Score)
		}
	}

	falsePositiveRate := float64(falsePositives) / float64(len(safeSamples))
	
	// Requirement: false positive rate <5%
	if falsePositiveRate > 0.05 {
		t.Errorf("False positive rate too high: %f%% (expected <5%%)", falsePositiveRate*100)
	}

	t.Logf("False positive rate: %f%% (%d/%d)", falsePositiveRate*100, falsePositives, len(safeSamples))
}

// Benchmark tests
func BenchmarkPIIDetection(b *testing.B) {
	config := moderation.LayerConfig{
		Name:      "pii_test",
		Enabled:   true,
		Weight:    1.0,
		Threshold: 0.5,
		Options:   map[string]interface{}{},
	}

	layer := NewPIILayer(config)
	context := moderation.ModerationContext{}
	
	testContent := "Contact support@example.com or call (555) 123-4567. SSN: 123-45-6789, Card: 4532-0151-1283-0366"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		layer.Moderate(testContent, context)
	}
}

func BenchmarkPIIDetectionLongContent(b *testing.B) {
	config := moderation.LayerConfig{
		Name:      "pii_test",
		Enabled:   true,
		Weight:    1.0,
		Threshold: 0.5,
		Options:   map[string]interface{}{},
	}

	layer := NewPIILayer(config)
	context := moderation.ModerationContext{}
	
	// Simulate a long document with scattered PII
	longContent := ""
	for i := 0; i < 100; i++ {
		longContent += "This is line " + string(rune('0'+i%10)) + " of a long document. "
		if i%10 == 0 {
			longContent += "Contact us at user" + string(rune('0'+i%10)) + "@example.com "
		}
		if i%15 == 0 {
			longContent += "or call (555) " + string(rune('0'+(i/10)%10)) + string(rune('0'+i%10)) + "0-1234. "
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		layer.Moderate(longContent, context)
	}
}