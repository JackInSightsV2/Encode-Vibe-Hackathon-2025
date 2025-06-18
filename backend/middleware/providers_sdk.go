package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/sashabaranov/go-openai"
	"qt1-middleware/config"
)

// SDKProviderRouter handles routing using official SDKs
type SDKProviderRouter struct {
	openaiClient *openai.Client
	httpClient   *http.Client
}

// NewSDKProviderRouter creates a new SDK-based provider router
func NewSDKProviderRouter() *SDKProviderRouter {
	router := &SDKProviderRouter{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	
	// Initialize OpenAI client if API key is available
	if openaiKey := getOpenAIKey(); openaiKey != "" {
		router.openaiClient = openai.NewClient(openaiKey)
		fmt.Printf("DEBUG: OpenAI client initialized with key: %s...%s\n", openaiKey[:10], openaiKey[len(openaiKey)-4:])
	} else {
		fmt.Printf("DEBUG: No OpenAI API key found. Checked OPENAI_API_KEY and OPENAI_KEY env vars.\n")
	}
	
	return router
}

// ReinitializeClients recreates the SDK clients with updated configuration
func (r *SDKProviderRouter) ReinitializeClients() {
	// Reinitialize OpenAI client
	if openaiKey := getOpenAIKey(); openaiKey != "" {
		r.openaiClient = openai.NewClient(openaiKey)
		fmt.Printf("DEBUG: OpenAI client reinitialized with updated API key\n")
	} else {
		r.openaiClient = nil
		fmt.Printf("DEBUG: OpenAI client cleared - no API key found\n")
	}
}

// RouteRequestSDK routes requests using official SDKs
func (r *SDKProviderRouter) RouteRequestSDK(req ChatRequestExtended) (*ProxyResponse, error) {
	// Determine which provider to use
	provider := r.selectProviderForSDK(req)
	
	switch provider {
	case "openai":
		return r.routeToOpenAI(req)
	case "anthropic":
		return r.routeToAnthropic(req)
	default:
		return nil, fmt.Errorf("no suitable SDK provider found for request")
	}
}

func (r *SDKProviderRouter) selectProviderForSDK(req ChatRequestExtended) string {
	// 1. Use explicitly specified provider if SDK is available
	if req.Provider != "" {
		switch req.Provider {
		case "openai":
			if r.openaiClient != nil {
				return "openai"
			}
		case "anthropic":
			// Anthropic will use HTTP client for now
			return "anthropic"
		}
	}
	
	// 2. Use model-based routing
	if req.Model != "" {
		if isOpenAIModel(req.Model) && r.openaiClient != nil {
			return "openai"
		}
		if isAnthropicModel(req.Model) {
			return "anthropic"
		}
	}
	
	// 3. Use default provider from config if SDK is available
	defaultProvider := config.AppConfig.Routing.DefaultProvider
	switch defaultProvider {
	case "openai":
		if r.openaiClient != nil {
			return "openai"
		}
	case "anthropic":
		return "anthropic"
	}
	
	// 4. Fall back to first available SDK
	if r.openaiClient != nil {
		return "openai"
	}
	
	return ""
}

func (r *SDKProviderRouter) routeToOpenAI(req ChatRequestExtended) (*ProxyResponse, error) {
	// Always check for API key and reinitialize if needed
	openaiKey := getOpenAIKey()
	fmt.Printf("=== OPENAI DEBUG ===\n")
	fmt.Printf("API Key found: %t\n", openaiKey != "")
	if openaiKey != "" {
		fmt.Printf("Key preview: %s...%s\n", openaiKey[:10], openaiKey[len(openaiKey)-4:])
	}
	
	if openaiKey == "" {
		return nil, fmt.Errorf("No OpenAI API key found - check config or environment variables")
	}
	
	// Always create fresh client to ensure we have latest config
	r.openaiClient = openai.NewClient(openaiKey)
	fmt.Printf("OpenAI client created successfully\n")
	
	// Use current model - gpt-4o-mini is latest and cheapest
	model := req.Model
	if model == "" {
		model = "gpt-4o-mini"
	}
	fmt.Printf("Using model: %s\n", model)
	
	// Create chat completion request
	chatReq := openai.ChatCompletionRequest{
		Model: model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: req.Message,
			},
		},
		MaxTokens:   2000,
		Temperature: 0.7,
	}
	
	// Apply custom settings
	if req.Settings != nil {
		if temp, ok := req.Settings["temperature"].(float64); ok {
			chatReq.Temperature = float32(temp)
		}
		if maxTokens, ok := req.Settings["max_tokens"].(float64); ok {
			chatReq.MaxTokens = int(maxTokens)
		}
	}
	
	// Make request with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	fmt.Printf("Making OpenAI API request...\n")
	resp, err := r.openaiClient.CreateChatCompletion(ctx, chatReq)
	if err != nil {
		fmt.Printf("OpenAI API ERROR: %v\n", err)
		return nil, fmt.Errorf("OpenAI API error: %w", err)
	}
	
	fmt.Printf("OpenAI API response received successfully\n")
	fmt.Printf("Response choices: %d\n", len(resp.Choices))
	
	// Format response
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response choices returned from OpenAI")
	}
	
	content := resp.Choices[0].Message.Content
	fmt.Printf("Response content preview: %.100s...\n", content)
	
	response := map[string]interface{}{
		"success": true,
		"message": content,
		"model":   resp.Model,
		"usage":   resp.Usage,
	}
	
	fmt.Printf("=== END OPENAI DEBUG ===\n")
	
	return &ProxyResponse{
		StatusCode: 200,
		Body:       marshalResponse(response),
	}, nil
}

func (r *SDKProviderRouter) routeToAnthropic(req ChatRequestExtended) (*ProxyResponse, error) {
	// For now, fall back to HTTP client for Anthropic
	// TODO: Implement proper Anthropic SDK when available
	return nil, fmt.Errorf("Anthropic SDK routing not implemented yet - use target_url instead")
}

// Helper functions

func getOpenAIKey() string {
	// Check config first (takes precedence over env vars)
	if provider, ok := config.AppConfig.Providers["openai"]; ok && provider.APIKey != "" {
		return provider.APIKey
	}
	
	// Fall back to environment variables
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		return key
	}
	if key := os.Getenv("OPENAI_KEY"); key != "" {
		return key
	}
	
	return ""
}

func getAnthropicKey() string {
	// Check environment variable first
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		return key
	}
	
	// Check config
	if provider, ok := config.AppConfig.Providers["anthropic"]; ok {
		return provider.APIKey
	}
	return ""
}

// TestOpenAIClient tests the OpenAI client with the provided message
func (r *SDKProviderRouter) TestOpenAIClient(message string) (map[string]interface{}, error) {
	key := getOpenAIKey()
	if key == "" {
		return nil, fmt.Errorf("no OpenAI API key found")
	}
	
	log.Printf("Testing OpenAI client with key: %s...%s", 
		key[:minInt(10, len(key))], 
		key[maxInt(0, len(key)-4):])
	
	client := openai.NewClient(key)
	
	// Use the provided message or a default one
	if message == "" {
		message = "Say this is a test!"
	}
	
	// Make the API request
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: "gpt-4.1-nano",
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    "user",
					Content: message,
				},
			},
			MaxTokens: 200,
			Temperature: 0.7,
		},
	)
	
	if err != nil {
		errMsg := fmt.Sprintf("OpenAI API error: %v", err)
		log.Println(errMsg)
		return map[string]interface{}{
			"success": false,
			"error":   errMsg,
		}, err
	}
	
	if len(resp.Choices) == 0 {
		errMsg := "no response from OpenAI"
		log.Println(errMsg)
		return map[string]interface{}{
			"success": false,
			"error":   errMsg,
		}, fmt.Errorf(errMsg)
	}
	
	// Log the first few characters of the response
	responseContent := resp.Choices[0].Message.Content
	log.Printf("OpenAI API response: %.100s...", responseContent)
	
	// Return the full response
	return map[string]interface{}{
		"success": true,
		"message": responseContent,
		"model":   resp.Model,
		"usage":   resp.Usage,
	}, nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func marshalResponse(response map[string]interface{}) []byte {
	data, _ := json.Marshal(response)
	return data
}