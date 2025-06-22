# Phase 4: Model Provider Abstraction

## Overview

Transform QT-1 from OpenAI-specific to a truly model-agnostic system supporting OpenAI, Anthropic Claude, Google Gemini, OpenRouter, and Ollama. Users can mix and match models for different components (chat, moderation, embeddings).

## Prerequisites

- Phase 1-3 completed successfully
- OpenAI integration fully operational
- AI features and embeddings working

## Architecture

```
Provider Abstraction:
Request → Provider Selection → Provider-Specific Client → Unified Response → Processing
```

## Project Structure Additions

```
backend/
├── providers/
│   ├── interfaces.go      # Common interfaces for all providers
│   ├── registry.go        # Provider registration and management
│   ├── openai/
│   │   ├── client.go      # OpenAI implementation
│   │   ├── chat.go        # Chat completions
│   │   ├── moderation.go  # Moderation API
│   │   └── embeddings.go  # Embeddings API
│   ├── anthropic/
│   │   ├── client.go      # Claude implementation
│   │   ├── chat.go        # Messages API
│   │   └── utils.go       # Claude-specific utilities
│   ├── google/
│   │   ├── client.go      # Gemini implementation
│   │   ├── chat.go        # Gemini Pro API
│   │   └── embeddings.go  # Gecko embeddings
│   ├── openrouter/
│   │   ├── client.go      # OpenRouter implementation
│   │   └── models.go      # Available models management
│   └── ollama/
│       ├── client.go      # Ollama local implementation
│       ├── chat.go        # Chat API
│       └── embeddings.go  # Local embeddings
├── routing/
│   ├── selector.go        # Model selection logic
│   ├── load_balancer.go   # Load balancing across providers
│   └── fallback.go        # Fallback mechanisms
└── data/
    ├── provider_configs.yaml  # Provider configurations
    └── model_mappings.yaml    # Model capability mappings
```

## Tasks Checklist

### 1. Provider Interface Design
- [ ] Create `providers/interfaces.go`:
  ```go
  type ChatProvider interface {
      ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error)
      StreamChatCompletion(ctx context.Context, req ChatRequest) (<-chan ChatResponse, error)
      Name() string
      Models() []string
      MaxTokens() int
      SupportsStreaming() bool
  }

  type ModerationProvider interface {
      ModerateContent(ctx context.Context, content string) (*ModerationResponse, error)
      Name() string
      Categories() []string
  }

  type EmbeddingProvider interface {
      CreateEmbedding(ctx context.Context, text string) (*EmbeddingResponse, error)
      CreateEmbeddings(ctx context.Context, texts []string) (*EmbeddingsResponse, error)
      Name() string
      Dimensions() int
      MaxInputTokens() int
  }
  ```

- [ ] Common types and structs:
  ```go
  type ChatRequest struct {
      Messages    []Message `json:"messages"`
      Model       string    `json:"model,omitempty"`
      MaxTokens   int       `json:"max_tokens,omitempty"`
      Temperature float64   `json:"temperature,omitempty"`
      Stream      bool      `json:"stream,omitempty"`
  }

  type Message struct {
      Role    string `json:"role"`
      Content string `json:"content"`
  }

  type ChatResponse struct {
      ID       string   `json:"id"`
      Model    string   `json:"model"`
      Choices  []Choice `json:"choices"`
      Usage    Usage    `json:"usage"`
      Provider string   `json:"provider"`
  }
  ```

### 2. Provider Registry System
- [ ] Create `providers/registry.go`:
  - `RegisterProvider()` - Register new providers
  - `GetProvider(name)` - Retrieve provider by name
  - `ListProviders()` - List all available providers
  - `GetCapabilities(provider)` - Get provider capabilities
  - Health checking for all providers

- [ ] Provider configuration management:
  - Load provider settings from config
  - API key management per provider
  - Rate limiting per provider
  - Provider-specific settings

### 3. OpenAI Provider Implementation
- [ ] Refactor existing OpenAI code into `providers/openai/`:
  - Move existing implementation to new structure
  - Implement all required interfaces
  - Add streaming support
  - Enhanced error handling

- [ ] Create `providers/openai/client.go`:
  - API client with proper authentication
  - Rate limiting and retry logic
  - Model management and validation
  - Cost tracking per request

### 4. Anthropic Claude Provider
- [ ] Create `providers/anthropic/client.go`:
  - Anthropic API client implementation
  - Authentication with API keys
  - Model selection (Claude-3, Claude-3.5)
  - Request/response transformation

- [ ] Create `providers/anthropic/chat.go`:
  - Messages API integration
  - Streaming support for Claude
  - Context window management
  - Claude-specific parameters (system prompts, etc.)

- [ ] Environment variables:
  ```
  ANTHROPIC_API_KEY=your_anthropic_key
  ANTHROPIC_BASE_URL=https://api.anthropic.com
  ```

### 5. Google Gemini Provider
- [ ] Create `providers/google/client.go`:
  - Google AI Studio API client
  - Authentication with API keys
  - Model selection (Gemini Pro, Gemini Pro Vision)
  - Safety settings configuration

- [ ] Create `providers/google/chat.go`:
  - GenerateContent API integration
  - Multi-turn conversation support
  - Function calling capabilities
  - Content filtering and safety

- [ ] Create `providers/google/embeddings.go`:
  - Gecko embeddings integration
  - Batch embedding processing
  - Task-specific embeddings

- [ ] Environment variables:
  ```
  GOOGLE_API_KEY=your_google_key
  GOOGLE_BASE_URL=https://generativelanguage.googleapis.com
  ```

### 6. OpenRouter Provider
- [ ] Create `providers/openrouter/client.go`:
  - OpenRouter API integration
  - Model marketplace access
  - Cost optimization features
  - Model availability checking

- [ ] Create `providers/openrouter/models.go`:
  - Dynamic model discovery
  - Model pricing information
  - Model capability metadata
  - Model selection optimization

- [ ] Environment variables:
  ```
  OPENROUTER_API_KEY=your_openrouter_key
  OPENROUTER_BASE_URL=https://openrouter.ai/api/v1
  ```

### 7. Ollama Local Provider
- [ ] Create `providers/ollama/client.go`:
  - Local Ollama server communication
  - Model management (pull, list, delete)
  - Health checking for local server
  - Automatic model downloading

- [ ] Create `providers/ollama/chat.go`:
  - Chat API integration
  - Streaming support
  - Model switching
  - Context management

- [ ] Create `providers/ollama/embeddings.go`:
  - Local embedding generation
  - Support for embedding models
  - Batch processing optimization

- [ ] Configuration:
  ```yaml
  ollama:
    base_url: "http://localhost:11434"
    timeout: "30s"
    auto_pull_models: true
    default_chat_model: "llama2"
    default_embedding_model: "nomic-embed-text"
  ```

### 8. Model Selection and Routing
- [ ] Create `routing/selector.go`:
  - Provider selection based on request type
  - Model selection within providers
  - Cost optimization algorithms
  - Performance-based selection

- [ ] Selection strategies:
  - Round-robin load balancing
  - Cost-optimized selection
  - Performance-based routing
  - Feature-based selection (streaming, context length)

- [ ] Create `data/model_mappings.yaml`:
  ```yaml
  model_mappings:
    chat_models:
      fast:
        - provider: openai
          model: gpt-3.5-turbo
        - provider: anthropic
          model: claude-3-haiku
      balanced:
        - provider: openai
          model: gpt-4
        - provider: anthropic
          model: claude-3-sonnet
      powerful:
        - provider: openai
          model: gpt-4-turbo
        - provider: anthropic
          model: claude-3-opus
    
    embedding_models:
      - provider: openai
        model: text-embedding-ada-002
        dimensions: 1536
      - provider: google
        model: embedding-gecko-001
        dimensions: 768
  ```

### 9. Load Balancing and Fallback
- [ ] Create `routing/load_balancer.go`:
  - Distribute requests across providers
  - Health-based routing
  - Circuit breaker implementation
  - Request queuing and throttling

- [ ] Create `routing/fallback.go`:
  - Automatic failover between providers
  - Graceful degradation strategies
  - Priority-based fallback chains
  - Error recovery mechanisms

### 10. Configuration System Enhancement
- [ ] Extend configuration for multiple providers:
  ```yaml
  providers:
    openai:
      enabled: true
      api_key_env: "OPENAI_KEY"
      base_url: "https://api.openai.com/v1"
      models:
        chat: ["gpt-3.5-turbo", "gpt-4", "gpt-4-turbo"]
        moderation: ["text-moderation-latest"]
        embedding: ["text-embedding-ada-002"]
      rate_limits:
        requests_per_minute: 3500
        tokens_per_minute: 90000
    
    anthropic:
      enabled: true
      api_key_env: "ANTHROPIC_API_KEY"
      base_url: "https://api.anthropic.com"
      models:
        chat: ["claude-3-haiku", "claude-3-sonnet", "claude-3-opus"]
      rate_limits:
        requests_per_minute: 1000
        tokens_per_minute: 40000
    
    # Similar configs for other providers...
  
  routing:
    default_chat_provider: "openai"
    default_moderation_provider: "openai"
    default_embedding_provider: "openai"
    fallback_enabled: true
    load_balancing: "round_robin" # round_robin, cost_optimized, performance
  ```

### 11. Database Schema Updates
- [ ] Add provider tracking to logs:
  ```sql
  ALTER TABLE log_entries ADD COLUMN provider_used TEXT;
  ALTER TABLE log_entries ADD COLUMN model_used TEXT;
  ALTER TABLE log_entries ADD COLUMN cost_estimate REAL;
  ALTER TABLE log_entries ADD COLUMN provider_response_time INTEGER;
  ```

- [ ] Provider statistics table:
  ```sql
  CREATE TABLE provider_stats (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      provider_name TEXT NOT NULL,
      model_name TEXT NOT NULL,
      request_count INTEGER DEFAULT 0,
      success_count INTEGER DEFAULT 0,
      error_count INTEGER DEFAULT 0,
      total_tokens INTEGER DEFAULT 0,
      total_cost REAL DEFAULT 0.0,
      avg_response_time REAL DEFAULT 0.0,
      last_updated DATETIME DEFAULT CURRENT_TIMESTAMP
  );
  ```

### 12. Unified Middleware Integration
- [ ] Update middleware to work with any provider:
  - Provider-agnostic request processing
  - Unified response handling
  - Provider-specific error handling
  - Consistent logging across providers

- [ ] Update relevance filtering:
  - Support different embedding providers
  - Cross-provider similarity calculations
  - Provider-specific optimization

### 13. Cost Management and Analytics
- [ ] Cost tracking system:
  - Per-provider cost calculation
  - Token usage monitoring
  - Cost alerts and budgeting
  - Usage optimization recommendations

- [ ] Analytics dashboard data:
  - Provider performance comparison
  - Model accuracy metrics
  - Cost per request analysis
  - User preference insights

### 14. Testing and Validation
- [ ] Unit tests for each provider:
  - Mock implementations for testing
  - Provider interface compliance
  - Error handling validation
  - Performance benchmarking

- [ ] Integration tests:
  - Cross-provider functionality
  - Fallback mechanism testing
  - Load balancing validation
  - End-to-end provider switching

- [ ] Test configurations:
  - Mock provider for testing
  - Test model configurations
  - Provider health simulation
  - Error scenario testing

### 15. Documentation and Examples
- [ ] Provider setup documentation:
  - API key configuration guide
  - Model selection recommendations
  - Performance tuning tips
  - Cost optimization strategies

- [ ] Integration examples:
  - Multi-provider configurations
  - Custom routing strategies
  - Provider-specific optimizations
  - Migration guides from single provider

## Success Criteria

- [ ] All five providers (OpenAI, Anthropic, Google, OpenRouter, Ollama) work seamlessly
- [ ] Users can configure different providers for different functions
- [ ] Automatic fallback works when providers fail
- [ ] Load balancing distributes requests efficiently
- [ ] Cost tracking provides accurate usage information
- [ ] Performance remains acceptable across all providers
- [ ] Configuration changes can be made without code changes
- [ ] All existing features work with any provider

## Provider-Specific Success Criteria

### OpenAI
- [ ] All existing functionality migrated to new structure
- [ ] Streaming chat completions working
- [ ] Moderation API integrated
- [ ] Embeddings API working

### Anthropic
- [ ] Claude 3 models accessible
- [ ] Streaming responses working
- [ ] System prompts properly handled
- [ ] Context window management

### Google
- [ ] Gemini Pro integration complete
- [ ] Safety settings configurable
- [ ] Gecko embeddings working
- [ ] Multi-modal support (future)

### OpenRouter
- [ ] Model marketplace accessible
- [ ] Dynamic model discovery
- [ ] Cost optimization working
- [ ] Model switching seamless

### Ollama
- [ ] Local server communication
- [ ] Model management (pull/delete)
- [ ] Offline functionality
- [ ] Local embeddings working

## Performance Targets

- [ ] Provider switching: < 5ms overhead
- [ ] Fallback activation: < 100ms detection
- [ ] Load balancing: < 1ms routing decision
- [ ] Cross-provider consistency: > 95% compatible responses
- [ ] Provider health checks: < 500ms per provider

## Next Phase

After completion, proceed to Phase 5: Admin API & Management for building comprehensive administration interfaces and management capabilities.