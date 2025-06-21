package moderation

import (
	"context"
	"encoding/json"
	"fmt"
	"qt1-middleware/opik"
	"strings"
	"sync"
	"time"

	"github.com/sashabaranov/go-openai"
)

// LLMModerator implements LLM-based content moderation
type LLMModerator struct {
	config       LayerConfig
	opikClient   *opik.OpikClient
	openaiClient *openai.Client
	model        string
	provider     string
	threshold    float64
	mu           sync.RWMutex
}

// LLMResponse represents the expected response format from the LLM
type LLMResponse struct {
	Safe       bool                   `json:"safe"`
	Confidence float64                `json:"confidence"`
	Categories map[string]float64     `json:"categories"`
	Reason     string                 `json:"reason"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// NewLLMModerator creates a new LLM-based moderator
func NewLLMModerator(config LayerConfig, opikClient *opik.OpikClient, openaiClient *openai.Client) (*LLMModerator, error) {
	// Extract configuration
	model := "gpt-3.5-turbo"
	if m, ok := config.Options["model"].(string); ok {
		model = m
	}

	provider := "openai"
	if p, ok := config.Options["provider"].(string); ok {
		provider = p
	}

	threshold := 0.7
	if t, ok := config.Options["threshold"].(float64); ok {
		threshold = t
	}

	return &LLMModerator{
		config:       config,
		opikClient:   opikClient,
		openaiClient: openaiClient,
		model:        model,
		provider:     provider,
		threshold:    threshold,
	}, nil
}

// Name returns the name of this moderation layer
func (lm *LLMModerator) Name() string {
	return "llm"
}

// Weight returns the weight of this layer
func (lm *LLMModerator) Weight() float64 {
	return lm.config.Weight
}

// Enabled returns whether this layer is enabled
func (lm *LLMModerator) Enabled() bool {
	return lm.config.Enabled
}

// Config returns the layer configuration
func (lm *LLMModerator) Config() LayerConfig {
	return lm.config
}

// Moderate performs LLM-based moderation
func (lm *LLMModerator) Moderate(content string, moderationCtx ModerationContext) ModerationResult {
	startTime := time.Now()

	result := ModerationResult{
		Score:      0.0,
		Confidence: 0.0,
		Blocked:    false,
		LayerName:  lm.Name(),
		Details:    make(map[string]interface{}),
	}

	// Skip if content is too short
	if len(strings.TrimSpace(content)) < 3 {
		result.Details["skipped"] = "content too short"
		result.ProcessTime = time.Since(startTime)
		return result
	}

	// Call LLM for moderation
	llmResult, err := lm.callLLMProvider(context.Background(), content)
	if err != nil {
		result.Details["error"] = err.Error()
		result.ProcessTime = time.Since(startTime)
		return result
	}

	// Process LLM response
	result.Score = lm.calculateScore(llmResult)
	result.Confidence = llmResult.Confidence
	result.Blocked = result.Score >= lm.threshold
	result.Reason = llmResult.Reason
	result.Category = lm.determineCategory(llmResult.Categories)
	result.Details["categories"] = llmResult.Categories
	result.Details["model"] = lm.model
	result.Details["provider"] = lm.provider
	result.ProcessTime = time.Since(startTime)

	return result
}

// ModerateWithTracing performs moderation with Opik tracing
func (lm *LLMModerator) ModerateWithTracing(ctx context.Context, content string, span *opik.Span) (*ModerationResult, error) {
	defer span.End()

	span.SetMetadata(map[string]interface{}{
		"model":      lm.model,
		"provider":   lm.provider,
		"threshold":  lm.threshold,
	})

	startTime := time.Now()

	// Create moderation context
	moderationCtx := ModerationContext{
		RequestID: span.TraceID,
		Timestamp: time.Now(),
	}

	// Perform moderation
	result := lm.Moderate(content, moderationCtx)

	// Set span output
	span.SetOutput(map[string]interface{}{
		"flagged":    result.Blocked,
		"categories": result.Details["categories"],
		"scores":     map[string]float64{result.Category: result.Score},
		"confidence": result.Confidence,
		"reason":     result.Reason,
	})

	// Add performance metrics
	span.SetMetadata(map[string]interface{}{
		"duration_ms":    time.Since(startTime).Milliseconds(),
		"content_length": len(content),
		"token_count":    lm.estimateTokens(content),
	})

	// Set accuracy score
	span.SetScore("accuracy", result.Confidence)

	// Convert to ModerationResult type
	categories := make(map[string]float64)
	if cats, ok := result.Details["categories"].(map[string]float64); ok {
		categories = cats
	}

	moderationResult := &ModerationResult{
		Blocked:        result.Blocked,
		Reason:         result.Reason,
		Confidence:     result.Confidence,
		Categories:     categories,
		Scores:         map[string]float64{"llm": result.Score},
		Flagged:        result.Blocked,
		ProcessedAt:    time.Now(),
		ProcessingTime: result.ProcessTime,
		Metadata:       result.Details,
	}

	return moderationResult, nil
}

// callLLMProvider calls the LLM provider for moderation
func (lm *LLMModerator) callLLMProvider(ctx context.Context, content string) (*LLMResponse, error) {
	// Build the moderation prompt
	systemPrompt := `You are a content moderation assistant. Analyze the given content and respond with a JSON object containing:
- "safe": boolean indicating if the content is safe
- "confidence": float between 0 and 1 indicating confidence in the assessment
- "categories": object with scores (0-1) for each category: violence, hate, harassment, self-harm, sexual, dangerous
- "reason": brief explanation of the decision

Be strict but fair. Consider context and intent.`

	userPrompt := fmt.Sprintf("Analyze this content for safety:\n\n%s", content)

	// Create the request
	req := openai.ChatCompletionRequest{
		Model: lm.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userPrompt},
		},
		Temperature: 0.3,
		MaxTokens:   200,
	}

	// Call OpenAI
	resp, err := lm.openaiClient.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from LLM")
	}

	// Parse the JSON response
	var llmResp LLMResponse
	responseText := resp.Choices[0].Message.Content
	
	// Try to extract JSON from the response
	jsonStart := strings.Index(responseText, "{")
	jsonEnd := strings.LastIndex(responseText, "}")
	if jsonStart >= 0 && jsonEnd >= jsonStart {
		jsonText := responseText[jsonStart:jsonEnd+1]
		if err := json.Unmarshal([]byte(jsonText), &llmResp); err != nil {
			// Fallback to text parsing
			return lm.parseTextResponse(responseText), nil
		}
	} else {
		// Fallback to text parsing
		return lm.parseTextResponse(responseText), nil
	}

	return &llmResp, nil
}

// parseTextResponse parses a text response when JSON parsing fails
func (lm *LLMModerator) parseTextResponse(text string) *LLMResponse {
	textLower := strings.ToLower(text)
	
	// Default response
	resp := &LLMResponse{
		Safe:       true,
		Confidence: 0.5,
		Categories: make(map[string]float64),
		Reason:     "Unable to parse structured response",
	}

	// Check for safety indicators
	if strings.Contains(textLower, "unsafe") || strings.Contains(textLower, "not safe") ||
		strings.Contains(textLower, "harmful") || strings.Contains(textLower, "dangerous") {
		resp.Safe = false
		resp.Confidence = 0.8
	}

	// Extract categories if mentioned
	categories := []string{"violence", "hate", "harassment", "self-harm", "sexual", "dangerous"}
	for _, cat := range categories {
		if strings.Contains(textLower, cat) {
			resp.Categories[cat] = 0.7
		}
	}

	// Extract reason
	if idx := strings.Index(textLower, "because"); idx > 0 && idx < len(text)-10 {
		resp.Reason = strings.TrimSpace(text[idx+7:])
		if len(resp.Reason) > 200 {
			resp.Reason = resp.Reason[:200] + "..."
		}
	}

	return resp
}

// calculateScore calculates the overall moderation score
func (lm *LLMModerator) calculateScore(llmResp *LLMResponse) float64 {
	if !llmResp.Safe {
		// If marked unsafe, use the highest category score
		maxScore := 0.5
		for _, score := range llmResp.Categories {
			if score > maxScore {
				maxScore = score
			}
		}
		return maxScore
	}

	// If marked safe, use a weighted average of category scores
	totalScore := 0.0
	count := 0
	for _, score := range llmResp.Categories {
		totalScore += score
		count++
	}
	
	if count > 0 {
		return totalScore / float64(count)
	}
	
	return 0.0
}

// determineCategory determines the primary category based on scores
func (lm *LLMModerator) determineCategory(categories map[string]float64) string {
	maxCategory := CategoryToxicity
	maxScore := 0.0
	
	for cat, score := range categories {
		if score > maxScore {
			maxScore = score
			maxCategory = cat
		}
	}
	
	// Map to our standard categories
	categoryMap := map[string]string{
		"violence":   CategoryViolence,
		"hate":       CategoryHateSpeech,
		"harassment": CategoryHateSpeech,
		"self-harm":  CategoryViolence,
		"sexual":     CategoryToxicity,
		"dangerous":  CategoryToxicity,
	}
	
	if mapped, ok := categoryMap[maxCategory]; ok {
		return mapped
	}
	
	return maxCategory
}

// estimateTokens estimates the number of tokens in the content
func (lm *LLMModerator) estimateTokens(content string) int {
	// Rough estimation: 1 token ≈ 4 characters
	return len(content) / 4
}

// GetStats returns statistics about the LLM moderator
func (lm *LLMModerator) GetStats() map[string]interface{} {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	return map[string]interface{}{
		"model":     lm.model,
		"provider":  lm.provider,
		"enabled":   lm.Enabled(),
		"weight":    lm.Weight(),
		"threshold": lm.threshold,
	}
}