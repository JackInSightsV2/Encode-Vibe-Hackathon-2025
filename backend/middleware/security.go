package middleware

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"qt1-middleware/config"
)

// PromptInjectionDetector handles detection of prompt injection attempts
type PromptInjectionDetector struct {
	patterns []*regexp.Regexp
}

// NewPromptInjectionDetector creates a new detector with configured patterns
func NewPromptInjectionDetector() *PromptInjectionDetector {
	detector := &PromptInjectionDetector{}
	detector.compilePatterns()
	return detector
}

func (p *PromptInjectionDetector) compilePatterns() {
	basePatterns := config.AppConfig.Security.PromptInjection.Patterns
	
	// Add additional sophisticated patterns
	sophisticatedPatterns := []string{
		// Direct instruction override
		`(?i)(ignore|forget|disregard|override)\s+(previous|prior|all|any|the)\s+(instructions?|rules?|commands?|directions?)`,
		`(?i)(new|different|updated|revised)\s+(instructions?|rules?|commands?|system\s+message)`,
		
		// Role manipulation  
		`(?i)(you\s+are\s+now|act\s+as|pretend\s+to\s+be|roleplay\s+as)\s+(?:a\s+)?(?!assistant|ai|helpful)(\w+)`,
		`(?i)(switch\s+to|become\s+a|transform\s+into)\s+(?:a\s+)?(\w+)\s+(mode|character|persona)`,
		
		// System message manipulation
		`(?i)(the\s+)?(system\s+message|initial\s+prompt|base\s+instructions?)\s+(is|are|should\s+be|has\s+been)\s+(changed|modified|updated|replaced)`,
		
		// Jailbreak attempts
		`(?i)(developer|admin|god|root|sudo|debug|test)\s+(mode|access|privileges?)`,
		`(?i)(jailbreak|jail\s*break|break\s+out|escape\s+mode)`,
		`(?i)DAN\s+(mode|\d+|protocol)`,
		
		// Hypothetical scenarios
		`(?i)(in\s+a\s+)?(hypothetical|fictional|imaginary|alternate)\s+(world|universe|scenario|situation)`,
		`(?i)(what\s+if|suppose|imagine\s+if|let's\s+say)\s+(?:that\s+)?(?:you|I|we)`,
		
		// Encoding attempts (basic detection)
		`(?i)(base64|rot\d+|caesar|hex|unicode)\s*(decode|decod|decrypt)`,
		`(?i)(decode|decrypt|decipher)\s+(this|the\s+following)`,
		
		// Boundary testing
		`(?i)(how\s+)?(would\s+you\s+)?(respond|answer|react)\s+if\s+(you\s+were|I\s+told\s+you)`,
		`(?i)(tell\s+me\s+about|explain|describe)\s+(?:your\s+)?(limitations|restrictions|boundaries|rules)`,
		
		// System probing
		`(?i)(what\s+are\s+your|show\s+me\s+your|list\s+your)\s+(instructions|rules|guidelines|parameters)`,
		`(?i)(repeat|echo|output|print)\s+(your\s+)?(system\s+message|instructions|prompt)`,
	}
	
	allPatterns := append(basePatterns, sophisticatedPatterns...)
	
	for _, pattern := range allPatterns {
		if compiled, err := regexp.Compile(pattern); err == nil {
			p.patterns = append(p.patterns, compiled)
		}
	}
}

// DetectPromptInjection checks for prompt injection attempts
func (p *PromptInjectionDetector) DetectPromptInjection(message string) (bool, string) {
	if !config.AppConfig.Security.PromptInjection.Enabled {
		return false, ""
	}
	
	// Normalize the message for analysis
	normalized := p.normalizeMessage(message)
	
	// Check for direct pattern matches
	for _, pattern := range p.patterns {
		if pattern.MatchString(normalized) {
			return true, fmt.Sprintf("Potential prompt injection detected: pattern match")
		}
	}
	
	// Check for encoding attempts
	if p.detectEncodingAttempts(message) {
		return true, "Potential encoding-based injection attempt detected"
	}
	
	// Check for excessive special characters (bypass attempts)
	if p.detectBypassAttempts(message) {
		return true, "Potential bypass attempt detected"
	}
	
	// Check for repetitive injection keywords
	if p.detectRepetitiveKeywords(normalized) {
		return true, "Repetitive injection keywords detected"
	}
	
	return false, ""
}

func (p *PromptInjectionDetector) normalizeMessage(message string) string {
	// Convert to lowercase
	normalized := strings.ToLower(message)
	
	// Remove excessive whitespace
	normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, " ")
	
	// Normalize common obfuscation attempts
	normalized = strings.ReplaceAll(normalized, "1", "i")
	normalized = strings.ReplaceAll(normalized, "3", "e")
	normalized = strings.ReplaceAll(normalized, "0", "o")
	normalized = strings.ReplaceAll(normalized, "5", "s")
	normalized = strings.ReplaceAll(normalized, "7", "t")
	
	// Remove common separators used in bypass attempts
	normalized = regexp.MustCompile(`[._\-*+]+`).ReplaceAllString(normalized, "")
	
	return strings.TrimSpace(normalized)
}

func (p *PromptInjectionDetector) detectEncodingAttempts(message string) bool {
	// Check for Base64 encoded content
	if p.looksLikeBase64(message) {
		if decoded, err := base64.StdEncoding.DecodeString(message); err == nil {
			decodedStr := string(decoded)
			// Recursively check decoded content
			if contains, _ := p.DetectPromptInjection(decodedStr); contains {
				return true
			}
		}
	}
	
	// Check for hex encoding
	if p.looksLikeHex(message) {
		return true
	}
	
	// Check for excessive unicode or special characters
	if p.hasExcessiveUnicode(message) {
		return true
	}
	
	return false
}

func (p *PromptInjectionDetector) detectBypassAttempts(message string) bool {
	// Count special characters
	specialCharCount := 0
	for _, char := range message {
		if !unicode.IsLetter(char) && !unicode.IsDigit(char) && !unicode.IsSpace(char) {
			specialCharCount++
		}
	}
	
	// If more than 30% special characters, flag as suspicious
	if len(message) > 0 && float64(specialCharCount)/float64(len(message)) > 0.3 {
		return true
	}
	
	// Check for excessive repetition of bypass characters
	bypassChars := []string{".", "_", "-", "*", "+", "|", "\\", "/"}
	for _, char := range bypassChars {
		if strings.Count(message, char) > 10 {
			return true
		}
	}
	
	return false
}

func (p *PromptInjectionDetector) detectRepetitiveKeywords(message string) bool {
	keywords := []string{"ignore", "forget", "override", "jailbreak", "instructions", "system", "admin", "developer"}
	
	for _, keyword := range keywords {
		if strings.Count(message, keyword) > 2 {
			return true
		}
	}
	
	return false
}

func (p *PromptInjectionDetector) looksLikeBase64(s string) bool {
	// Basic heuristic for Base64 detection
	if len(s) < 10 || len(s)%4 != 0 {
		return false
	}
	
	base64Pattern := regexp.MustCompile(`^[A-Za-z0-9+/]*={0,2}$`)
	return base64Pattern.MatchString(s)
}

func (p *PromptInjectionDetector) looksLikeHex(s string) bool {
	if len(s) < 10 || len(s)%2 != 0 {
		return false
	}
	
	hexPattern := regexp.MustCompile(`^[0-9A-Fa-f]+$`)
	return hexPattern.MatchString(s)
}

func (p *PromptInjectionDetector) hasExcessiveUnicode(s string) bool {
	unicodeCount := 0
	for _, char := range s {
		if char > 127 {
			unicodeCount++
		}
	}
	
	if len(s) > 0 && float64(unicodeCount)/float64(len(s)) > 0.2 {
		return true
	}
	
	return false
}