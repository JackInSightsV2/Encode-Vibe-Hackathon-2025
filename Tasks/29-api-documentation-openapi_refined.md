# API Documentation & OpenAPI Specification - Refined Implementation Cycles

## Overview
Break down comprehensive API documentation into 3 manageable cycles, each delivering testable functionality with clear outcomes.

---

## **Cycle 18A: OpenAPI Specification and Code Annotation**
**Duration:** 6-7 hours | **Priority:** Critical

### Prerequisites
- Go development environment ready
- Understanding of OpenAPI 3.0 specification format
- Knowledge of Swagger/OpenAPI annotation patterns

### Implementation Tasks
- [ ] Install Swagger/OpenAPI dependencies for Go
- [ ] Create comprehensive OpenAPI 3.0 specification file
- [ ] Add Swagger annotations to all Go API handlers
- [ ] Implement automatic spec generation from code
- [ ] Add detailed request/response schemas
- [ ] Configure authentication specifications

### Code Deliverables
```yaml
# api/openapi.yaml
openapi: 3.0.3
info:
  title: QT-1 Middleware API
  description: |
    Responsible AI Middleware API for content moderation and intelligent routing.
    
    ## Authentication
    Most endpoints require API key authentication via the `Authorization` header.
    
    ## Rate Limiting
    API calls are rate limited to 1000 requests per hour per API key.
    
    ## Error Handling
    All errors follow RFC 7807 Problem Details format.
  version: 1.0.0
  contact:
    name: QT-1 Support Team
    email: support@qt1-middleware.com
    url: https://docs.qt1-middleware.com
  license:
    name: MIT
    url: https://opensource.org/licenses/MIT
  x-logo:
    url: https://qt1-middleware.com/logo.png

servers:
  - url: https://api.qt1-middleware.com/v1
    description: Production server
  - url: https://staging-api.qt1-middleware.com/v1
    description: Staging server
  - url: http://localhost:8080/v1
    description: Development server

paths:
  /chat:
    post:
      tags:
        - Chat
      summary: Process chat request
      description: |
        Process a chat request through the QT-1 middleware with content moderation
        and intelligent provider routing.
      operationId: processChat
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/ChatRequest'
            examples:
              basic_chat:
                summary: Basic chat message
                value:
                  user_id: "user_123"
                  session_id: "session_456"
                  message: "Hello, how are you today?"
                  provider: "openai"
                  model: "gpt-3.5-turbo"
      responses:
        '200':
          description: Chat response generated successfully
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ChatResponse'
              examples:
                success_response:
                  summary: Successful chat response
                  value:
                    response: "Hello! I'm doing well, thank you for asking. How can I help you today?"
                    provider: "openai"
                    model: "gpt-3.5-turbo"
                    moderation_passed: true
                    response_time_ms: 1250
                    tokens_used: 45
        '400':
          $ref: '#/components/responses/BadRequest'
        '401':
          $ref: '#/components/responses/Unauthorized'
        '429':
          $ref: '#/components/responses/RateLimited'
        '500':
          $ref: '#/components/responses/InternalError'

  /health:
    get:
      tags:
        - System
      summary: Health check endpoint
      description: Returns the current health status of the middleware
      operationId: getHealth
      responses:
        '200':
          description: System is healthy
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/HealthResponse'

components:
  securitySchemes:
    ApiKeyAuth:
      type: apiKey
      in: header
      name: Authorization
      description: API key authentication (prefix with 'Bearer ')

  schemas:
    ChatRequest:
      type: object
      required:
        - user_id
        - session_id
        - message
      properties:
        user_id:
          type: string
          description: Unique identifier for the user
          example: "user_123"
          minLength: 1
          maxLength: 100
        session_id:
          type: string
          description: Session identifier for conversation context
          example: "session_456"
          minLength: 1
          maxLength: 100
        message:
          type: string
          description: User message content
          example: "Hello, how are you?"
          minLength: 1
          maxLength: 4000
        provider:
          type: string
          description: Preferred AI provider (optional, auto-selected if not specified)
          enum: [openai, anthropic, google, auto]
          example: "openai"
        model:
          type: string
          description: Specific model to use (optional)
          example: "gpt-3.5-turbo"
        temperature:
          type: number
          description: Model temperature for response variability
          minimum: 0
          maximum: 2
          example: 0.7
        max_tokens:
          type: integer
          description: Maximum tokens in response
          minimum: 1
          maximum: 4000
          example: 150

    ChatResponse:
      type: object
      properties:
        response:
          type: string
          description: Generated response from AI provider
          example: "Hello! I'm doing well, thank you for asking."
        provider:
          type: string
          description: AI provider used for generation
          example: "openai"
        model:
          type: string
          description: Specific model used
          example: "gpt-3.5-turbo"
        moderation_passed:
          type: boolean
          description: Whether content passed moderation checks
          example: true
        response_time_ms:
          type: integer
          description: Response time in milliseconds
          example: 1250
        tokens_used:
          type: integer
          description: Number of tokens consumed
          example: 45
        cost_estimate:
          type: number
          description: Estimated cost in USD
          example: 0.0023

    HealthResponse:
      type: object
      properties:
        status:
          type: string
          enum: [healthy, degraded, unhealthy]
          example: "healthy"
        timestamp:
          type: string
          format: date-time
          example: "2023-12-07T15:30:00Z"
        uptime_seconds:
          type: integer
          example: 3600
        version:
          type: string
          example: "1.0.0"
        checks:
          type: array
          items:
            $ref: '#/components/schemas/HealthCheck'

    HealthCheck:
      type: object
      properties:
        name:
          type: string
          example: "database"
        status:
          type: string
          enum: [healthy, degraded, unhealthy]
        response_time_ms:
          type: integer
          example: 25

    ErrorResponse:
      type: object
      required:
        - type
        - title
        - status
      properties:
        type:
          type: string
          format: uri
          description: Problem type URI
          example: "https://api.qt1-middleware.com/problems/invalid-request"
        title:
          type: string
          description: Human-readable problem summary
          example: "Invalid request format"
        status:
          type: integer
          description: HTTP status code
          example: 400
        detail:
          type: string
          description: Detailed error description
          example: "The 'message' field is required and cannot be empty"
        instance:
          type: string
          format: uri
          description: URI reference for this problem occurrence
          example: "/chat/request-123"

  responses:
    BadRequest:
      description: Bad request - invalid input
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/ErrorResponse'
    
    Unauthorized:
      description: Unauthorized - invalid or missing API key
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/ErrorResponse'
    
    RateLimited:
      description: Rate limit exceeded
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/ErrorResponse'
      headers:
        X-RateLimit-Limit:
          schema:
            type: integer
            example: 1000
        X-RateLimit-Remaining:
          schema:
            type: integer
            example: 0
        X-RateLimit-Reset:
          schema:
            type: integer
            example: 1609459200
    
    InternalError:
      description: Internal server error
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/ErrorResponse'

security:
  - ApiKeyAuth: []

tags:
  - name: Chat
    description: Chat processing and AI interactions
  - name: System
    description: System health and status endpoints
  - name: Analytics
    description: Usage analytics and metrics
  - name: Configuration
    description: System configuration management
```

```go
// backend/api/handlers.go with Swagger annotations
package api

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/swaggo/swag"
)

// ChatRequest represents a chat request
type ChatRequest struct {
    UserID      string  `json:"user_id" binding:"required" example:"user_123"`
    SessionID   string  `json:"session_id" binding:"required" example:"session_456"`
    Message     string  `json:"message" binding:"required" example:"Hello, how are you?"`
    Provider    string  `json:"provider,omitempty" example:"openai"`
    Model       string  `json:"model,omitempty" example:"gpt-3.5-turbo"`
    Temperature float64 `json:"temperature,omitempty" example:"0.7"`
    MaxTokens   int     `json:"max_tokens,omitempty" example:"150"`
}

// ChatResponse represents a chat response
type ChatResponse struct {
    Response         string  `json:"response" example:"Hello! I'm doing well, thank you."`
    Provider         string  `json:"provider" example:"openai"`
    Model           string  `json:"model" example:"gpt-3.5-turbo"`
    ModerationPassed bool    `json:"moderation_passed" example:"true"`
    ResponseTimeMs   int     `json:"response_time_ms" example:"1250"`
    TokensUsed       int     `json:"tokens_used" example:"45"`
    CostEstimate     float64 `json:"cost_estimate" example:"0.0023"`
}

// ProcessChat godoc
// @Summary      Process chat request
// @Description  Process a chat request through the QT-1 middleware with content moderation and intelligent provider routing
// @Tags         Chat
// @Accept       json
// @Produce      json
// @Param        request body ChatRequest true "Chat request payload"
// @Success      200 {object} ChatResponse "Chat response generated successfully"
// @Failure      400 {object} ErrorResponse "Bad request - invalid input"
// @Failure      401 {object} ErrorResponse "Unauthorized - invalid API key"
// @Failure      429 {object} ErrorResponse "Rate limit exceeded"
// @Failure      500 {object} ErrorResponse "Internal server error"
// @Security     ApiKeyAuth
// @Router       /chat [post]
func (h *Handler) ProcessChat(c *gin.Context) {
    var req ChatRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, ErrorResponse{
            Type:   "https://api.qt1-middleware.com/problems/invalid-request",
            Title:  "Invalid request format",
            Status: 400,
            Detail: err.Error(),
        })
        return
    }

    // Process chat logic here...
    
    response := ChatResponse{
        Response:         "Generated response here",
        Provider:         req.Provider,
        Model:           req.Model,
        ModerationPassed: true,
        ResponseTimeMs:   1250,
        TokensUsed:       45,
        CostEstimate:     0.0023,
    }

    c.JSON(http.StatusOK, response)
}

// GetHealth godoc
// @Summary      Health check endpoint
// @Description  Returns the current health status of the middleware
// @Tags         System
// @Produce      json
// @Success      200 {object} HealthResponse "System is healthy"
// @Success      503 {object} HealthResponse "System is unhealthy"
// @Router       /health [get]
func (h *Handler) GetHealth(c *gin.Context) {
    // Health check logic here...
    
    response := HealthResponse{
        Status:        "healthy",
        Timestamp:     time.Now(),
        UptimeSeconds: 3600,
        Version:       "1.0.0",
        Checks: []HealthCheck{
            {
                Name:           "database",
                Status:         "healthy",
                ResponseTimeMs: 25,
            },
        },
    }

    c.JSON(http.StatusOK, response)
}
```

```bash
#!/bin/bash
# scripts/generate-docs.sh
set -e

echo "🔧 Generating OpenAPI documentation..."

# Install swag if not present
if ! command -v swag &> /dev/null; then
    echo "Installing swag..."
    go install github.com/swaggo/swag/cmd/swag@latest
fi

# Generate docs from annotations
cd backend
swag init -g main.go -o ./docs

# Validate the generated spec
echo "📋 Validating OpenAPI specification..."
if command -v swagger &> /dev/null; then
    swagger validate docs/swagger.yaml
else
    echo "⚠️  swagger-cli not found, skipping validation"
fi

echo "✅ Documentation generated successfully!"
echo "📖 Swagger UI: http://localhost:8080/swagger/index.html"
```

### Testing Requirements
- [ ] Test OpenAPI spec validation with swagger-cli
- [ ] Test annotation coverage for all endpoints
- [ ] Test example values in documentation
- [ ] Test authentication specification
- [ ] Test generated documentation accuracy

### Acceptance Criteria
- [ ] OpenAPI 3.0 specification validates without errors
- [ ] All API endpoints covered with comprehensive annotations
- [ ] Request/response examples provided for all endpoints
- [ ] Authentication and authorization clearly documented
- [ ] Error responses follow consistent format
- [ ] Documentation generates automatically from code

### Risk Mitigation
- Use established annotation patterns from swaggo documentation
- Test spec validation in CI pipeline
- Keep examples synchronized with actual API behavior

---

## **Cycle 18B: Interactive Documentation and SDK Generation**
**Duration:** 5-6 hours | **Priority:** High

### Prerequisites
- Cycle 18A completed and tested
- Understanding of Swagger UI customization
- Knowledge of SDK generation tools

### Implementation Tasks
- [ ] Set up Swagger UI with custom branding
- [ ] Configure ReDoc for alternative documentation view
- [ ] Implement SDK generation for multiple languages
- [ ] Create API testing playground
- [ ] Add code examples in multiple languages
- [ ] Configure authentication testing in UI

### Code Deliverables
```go
// backend/main.go - Swagger UI setup
package main

import (
    "github.com/gin-gonic/gin"
    swaggerFiles "github.com/swaggo/files"
    ginSwagger "github.com/swaggo/gin-swagger"
    _ "./docs" // Import generated docs
)

// @title           QT-1 Middleware API
// @version         1.0
// @description     Responsible AI Middleware API for content moderation and intelligent routing
// @termsOfService  https://qt1-middleware.com/terms

// @contact.name   QT-1 Support
// @contact.url    https://qt1-middleware.com/support
// @contact.email  support@qt1-middleware.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      api.qt1-middleware.com
// @BasePath  /v1

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
// @description API key authentication (prefix with 'Bearer ')

func main() {
    r := gin.Default()

    // Custom Swagger UI configuration
    r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler,
        ginSwagger.URL("http://localhost:8080/swagger/doc.json"),
        ginSwagger.DocExpansion("none"),
        ginSwagger.DeepLinking(true),
        ginSwagger.DefaultModelsExpandDepth(-1),
    ))

    // ReDoc alternative documentation
    r.StaticFile("/redoc", "./static/redoc.html")

    // API routes
    v1 := r.Group("/v1")
    {
        v1.POST("/chat", handler.ProcessChat)
        v1.GET("/health", handler.GetHealth)
    }

    r.Run(":8080")
}
```

```html
<!-- static/redoc.html -->
<!DOCTYPE html>
<html>
<head>
    <title>QT-1 Middleware API Documentation</title>
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <link href="https://fonts.googleapis.com/css?family=Montserrat:300,400,700|Roboto:300,400,700" rel="stylesheet">
    <style>
        body { margin: 0; padding: 0; }
        redoc { 
            --redoc-font-family: 'Montserrat', sans-serif;
            --redoc-code-font-family: 'Monaco', 'Consolas', monospace;
        }
    </style>
</head>
<body>
    <redoc spec-url='/swagger/doc.json'></redoc>
    <script src="https://cdn.jsdelivr.net/npm/redoc@latest/bundles/redoc.standalone.js"></script>
</body>
</html>
```

```yaml
# sdk-generation/.openapi-generator/config.yaml
# Go SDK Configuration
go:
  packageName: qt1client
  packageUrl: github.com/qt1-middleware/go-client
  packageVersion: 1.0.0
  clientPackage: client
  packageCompany: QT-1
  authorName: QT-1 Team
  authorEmail: developers@qt1-middleware.com

# Python SDK Configuration  
python:
  packageName: qt1_client
  projectName: qt1-middleware-python-client
  packageVersion: 1.0.0
  packageCompany: QT-1
  authorName: QT-1 Team
  authorEmail: developers@qt1-middleware.com

# JavaScript SDK Configuration
javascript:
  projectName: qt1-middleware-js-client
  projectVersion: 1.0.0
  projectDescription: QT-1 Middleware JavaScript/TypeScript client
  authorName: QT-1 Team
  authorEmail: developers@qt1-middleware.com
  npmRepository: https://registry.npmjs.org/
```

```bash
#!/bin/bash
# scripts/generate-sdks.sh
set -e

echo "🛠️  Generating SDKs for QT-1 Middleware API..."

# Ensure OpenAPI Generator is available
if ! command -v openapi-generator &> /dev/null; then
    echo "Installing OpenAPI Generator..."
    npm install -g @openapitools/openapi-generator-cli
fi

# Create output directories
mkdir -p sdks/{go,python,javascript,java}

# Generate Go SDK
echo "📦 Generating Go SDK..."
openapi-generator generate \
    -i api/openapi.yaml \
    -g go \
    -o sdks/go \
    -c sdk-generation/.openapi-generator/config.yaml \
    --additional-properties=packageName=qt1client,isGoSubmodule=false

# Generate Python SDK
echo "🐍 Generating Python SDK..."
openapi-generator generate \
    -i api/openapi.yaml \
    -g python \
    -o sdks/python \
    -c sdk-generation/.openapi-generator/config.yaml \
    --additional-properties=packageName=qt1_client,projectName=qt1-middleware-python-client

# Generate JavaScript/TypeScript SDK
echo "📜 Generating JavaScript SDK..."
openapi-generator generate \
    -i api/openapi.yaml \
    -g typescript-fetch \
    -o sdks/javascript \
    -c sdk-generation/.openapi-generator/config.yaml \
    --additional-properties=npmName=qt1-middleware-client,supportsES6=true

# Generate Java SDK
echo "☕ Generating Java SDK..."
openapi-generator generate \
    -i api/openapi.yaml \
    -g java \
    -o sdks/java \
    --additional-properties=groupId=com.qt1,artifactId=qt1-middleware-client,apiPackage=com.qt1.api,modelPackage=com.qt1.model

echo "✅ SDK generation complete!"
echo "📁 SDKs available in ./sdks/ directory"
```

```typescript
// examples/typescript-example.ts
import { Configuration, ChatApi, ChatRequest } from 'qt1-middleware-client';

// Configure API client
const configuration = new Configuration({
    basePath: 'https://api.qt1-middleware.com/v1',
    apiKey: 'your-api-key-here'
});

const chatApi = new ChatApi(configuration);

async function example() {
    try {
        const chatRequest: ChatRequest = {
            user_id: 'user_123',
            session_id: 'session_456',
            message: 'Hello, how are you today?',
            provider: 'openai',
            model: 'gpt-3.5-turbo'
        };

        const response = await chatApi.processChat(chatRequest);
        console.log('Chat response:', response.data.response);
        console.log('Tokens used:', response.data.tokens_used);
        console.log('Cost estimate:', response.data.cost_estimate);
    } catch (error) {
        console.error('Error:', error.response?.data || error.message);
    }
}

example();
```

```python
# examples/python-example.py
import qt1_client
from qt1_client.rest import ApiException

# Configure API client
configuration = qt1_client.Configuration(
    host="https://api.qt1-middleware.com/v1",
    api_key={'ApiKeyAuth': 'your-api-key-here'}
)

# Create API instance
with qt1_client.ApiClient(configuration) as api_client:
    chat_api = qt1_client.ChatApi(api_client)
    
    # Create chat request
    chat_request = qt1_client.ChatRequest(
        user_id="user_123",
        session_id="session_456",
        message="Hello, how are you today?",
        provider="openai",
        model="gpt-3.5-turbo"
    )
    
    try:
        # Process chat
        response = chat_api.process_chat(chat_request)
        print(f"Chat response: {response.response}")
        print(f"Tokens used: {response.tokens_used}")
        print(f"Cost estimate: {response.cost_estimate}")
    except ApiException as e:
        print(f"Exception: {e}")
```

### Testing Requirements
- [ ] Test Swagger UI functionality and customization
- [ ] Test SDK generation for all target languages
- [ ] Test generated SDK examples
- [ ] Test authentication in interactive documentation
- [ ] Test API playground functionality

### Acceptance Criteria
- [ ] Swagger UI loads with custom branding and configuration
- [ ] ReDoc provides alternative documentation view
- [ ] SDKs generate successfully for Go, Python, JavaScript, Java
- [ ] Generated SDK examples execute without errors
- [ ] Interactive documentation allows API testing
- [ ] Authentication works in documentation UI

### Risk Mitigation
- Test SDK generation with CI pipeline
- Validate generated code compiles and runs
- Use established OpenAPI Generator templates

---

## **Cycle 18C: Documentation Website and Developer Portal**
**Duration:** 4-5 hours | **Priority:** Medium

### Prerequisites
- Cycles 18A and 18B completed
- Understanding of static site generation
- Knowledge of developer portal patterns

### Implementation Tasks
- [ ] Create comprehensive documentation website
- [ ] Add getting started guides and tutorials
- [ ] Implement search functionality
- [ ] Create code examples and cookbook
- [ ] Add changelog and migration guides
- [ ] Set up documentation deployment automation

### Code Deliverables
```markdown
<!-- docs/getting-started.md -->
# Getting Started with QT-1 Middleware API

## Quick Start

The QT-1 Middleware API provides responsible AI content moderation and intelligent provider routing. Get started in minutes.

### 1. Get Your API Key

Visit the [Developer Portal](https://developers.qt1-middleware.com) to create an account and generate your API key.

### 2. Make Your First Request

```bash
curl -X POST https://api.qt1-middleware.com/v1/chat \
  -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user_123",
    "session_id": "session_456", 
    "message": "Hello, how are you?"
  }'
```

### 3. Handle the Response

```json
{
  "response": "Hello! I'm doing well, thank you for asking. How can I help you today?",
  "provider": "openai",
  "model": "gpt-3.5-turbo",
  "moderation_passed": true,
  "response_time_ms": 1250,
  "tokens_used": 45,
  "cost_estimate": 0.0023
}
```

## Authentication

All API requests require authentication using an API key in the Authorization header:

```
Authorization: Bearer your-api-key-here
```

## Rate Limiting

API calls are limited to 1000 requests per hour per API key. Rate limit headers are included in all responses:

- `X-RateLimit-Limit`: Maximum requests per hour
- `X-RateLimit-Remaining`: Remaining requests in current window  
- `X-RateLimit-Reset`: Unix timestamp when rate limit resets

## Error Handling

All errors follow RFC 7807 Problem Details format:

```json
{
  "type": "https://api.qt1-middleware.com/problems/invalid-request",
  "title": "Invalid request format", 
  "status": 400,
  "detail": "The 'message' field is required and cannot be empty",
  "instance": "/chat/request-123"
}
```

## Next Steps

- [Explore the API Reference](./api-reference.md)
- [View SDK Documentation](./sdks/)
- [Check out Code Examples](./examples/)
- [Read Best Practices](./best-practices.md)
```

```javascript
// docs/.vitepress/config.js
export default {
  title: 'QT-1 Middleware Docs',
  description: 'Responsible AI Middleware API Documentation',
  
  themeConfig: {
    logo: '/logo.svg',
    
    nav: [
      { text: 'Guide', link: '/getting-started' },
      { text: 'API Reference', link: '/api-reference' },
      { text: 'SDKs', link: '/sdks/' },
      { text: 'Examples', link: '/examples/' }
    ],
    
    sidebar: {
      '/': [
        {
          text: 'Introduction',
          items: [
            { text: 'Getting Started', link: '/getting-started' },
            { text: 'Authentication', link: '/authentication' },
            { text: 'Rate Limiting', link: '/rate-limiting' },
            { text: 'Error Handling', link: '/error-handling' }
          ]
        },
        {
          text: 'API Reference',
          items: [
            { text: 'Chat API', link: '/api/chat' },
            { text: 'System API', link: '/api/system' },
            { text: 'Analytics API', link: '/api/analytics' }
          ]
        },
        {
          text: 'SDKs',
          items: [
            { text: 'JavaScript/TypeScript', link: '/sdks/javascript' },
            { text: 'Python', link: '/sdks/python' },
            { text: 'Go', link: '/sdks/go' },
            { text: 'Java', link: '/sdks/java' }
          ]
        },
        {
          text: 'Examples',
          items: [
            { text: 'Chat Bot', link: '/examples/chatbot' },
            { text: 'Content Moderation', link: '/examples/moderation' },
            { text: 'Analytics Integration', link: '/examples/analytics' }
          ]
        }
      ]
    },
    
    search: {
      provider: 'local'
    },
    
    socialLinks: [
      { icon: 'github', link: 'https://github.com/qt1-middleware/api' }
    ],
    
    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © 2023 QT-1 Middleware'
    }
  },

  head: [
    ['link', { rel: 'icon', href: '/favicon.ico' }],
    ['meta', { name: 'theme-color', content: '#3b82f6' }]
  ]
}
```

```yaml
# .github/workflows/docs.yml
name: Deploy Documentation

on:
  push:
    branches: [main]
    paths: ['docs/**', 'api/**', 'backend/**/*.go']
  
  workflow_dispatch:

jobs:
  deploy-docs:
    runs-on: ubuntu-latest
    
    steps:
    - uses: actions/checkout@v4
      with:
        fetch-depth: 0
    
    - name: Setup Node.js
      uses: actions/setup-node@v4
      with:
        node-version: '18'
        cache: 'npm'
        cache-dependency-path: docs/package-lock.json
    
    - name: Setup Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Generate OpenAPI Spec
      run: |
        cd backend
        go install github.com/swaggo/swag/cmd/swag@latest
        swag init -g main.go -o ../docs/public/api
    
    - name: Install dependencies
      run: |
        cd docs
        npm ci
    
    - name: Build documentation
      run: |
        cd docs
        npm run build
    
    - name: Deploy to GitHub Pages
      uses: peaceiris/actions-gh-pages@v3
      with:
        github_token: ${{ secrets.GITHUB_TOKEN }}
        publish_dir: docs/.vitepress/dist
        cname: docs.qt1-middleware.com
```

```typescript
// docs/.vitepress/theme/SearchBox.vue
<template>
  <div class="search-box">
    <input
      ref="input"
      v-model="query"
      :placeholder="placeholder"
      @input="onInput"
      @focus="onFocus"
      @blur="onBlur"
      @keydown.up="onUp"
      @keydown.down="onDown"
      @keydown.enter="onEnter"
      @keydown.escape="onEscape"
    />
    
    <ul v-if="showSuggestions" class="suggestions">
      <li
        v-for="(suggestion, i) in suggestions"
        :key="i"
        :class="{ focused: i === focusIndex }"
        @click="go(i)"
      >
        <a :href="suggestion.path" @click.prevent>
          <span class="page-title">{{ suggestion.title }}</span>
          <span v-if="suggestion.header" class="header">{{ suggestion.header.title }}</span>
        </a>
      </li>
    </ul>
  </div>
</template>

<script>
import { ref, computed, watch, nextTick } from 'vue'
import { useRouter } from 'vue-router'

export default {
  setup() {
    const router = useRouter()
    const query = ref('')
    const focused = ref(false)
    const focusIndex = ref(0)
    
    const showSuggestions = computed(() => {
      return focused.value && suggestions.value.length
    })
    
    const suggestions = computed(() => {
      if (!query.value) return []
      
      // Search through documentation pages
      return searchPages(query.value).slice(0, 10)
    })
    
    function searchPages(searchQuery) {
      // Implementation for searching through documentation pages
      // This would typically integrate with a search index
      return []
    }
    
    return {
      query,
      focused,
      focusIndex,
      suggestions,
      showSuggestions,
      placeholder: 'Search docs...'
    }
  }
}
</script>
```

### Testing Requirements
- [ ] Test documentation website build and deployment
- [ ] Test search functionality
- [ ] Test code examples and copy-paste functionality
- [ ] Test responsive design on mobile devices
- [ ] Test documentation navigation and links

### Acceptance Criteria
- [ ] Documentation website deploys automatically on content changes
- [ ] Search functionality finds relevant documentation quickly
- [ ] Code examples are syntactically correct and executable
- [ ] Documentation is mobile-responsive and accessible
- [ ] Navigation is intuitive with clear information architecture
- [ ] Documentation stays synchronized with API changes

### Risk Mitigation
- Use established documentation frameworks (VitePress, Docusaurus)
- Implement automated testing for code examples
- Set up automated synchronization between API spec and docs

---

## **Integration Testing Checklist**
After all cycles complete:
- [ ] End-to-end test: API annotation → spec generation → SDK → documentation
- [ ] Cross-platform SDK testing (Go, Python, JavaScript, Java)
- [ ] Documentation accuracy and currency testing
- [ ] Interactive documentation functionality testing
- [ ] Developer onboarding flow testing with real users

## **Success Metrics**
- OpenAPI specification achieves 100% endpoint coverage
- Generated SDKs compile and execute without errors
- Interactive documentation allows successful API testing
- Documentation website loads within 2 seconds
- Developer onboarding time reduced by 60%
- API documentation receives 95%+ developer satisfaction rating