#!/usr/bin/env pwsh

# Test script for Relevancy AI Provider API endpoints
# Run this after starting the backend server

$BASE_URL = "http://localhost:8080"

Write-Host "🤖 Testing Relevancy AI Provider API Endpoints" -ForegroundColor Cyan
Write-Host "=" * 50

# Test 1: Get AI Provider Configuration
Write-Host "`n1️⃣ Getting AI Provider Configuration..." -ForegroundColor Yellow
try {
    $response = Invoke-RestMethod -Uri "$BASE_URL/api/relevancy/ai-provider" -Method GET
    Write-Host "✅ Success!" -ForegroundColor Green
    Write-Host "Current Configuration:" -ForegroundColor White
    $response.data | ConvertTo-Json -Depth 3 | Write-Host
} catch {
    Write-Host "❌ Failed: $($_.Exception.Message)" -ForegroundColor Red
}

# Test 2: Configure AI Provider (OpenAI)
Write-Host "`n2️⃣ Configuring OpenAI Provider..." -ForegroundColor Yellow
$aiConfig = @{
    ai_enabled = $true
    provider = "openai"
    model = "gpt-4"
    api_key = "sk-test-key-12345"
    endpoint = ""
    max_tokens = 2048
    temperature = 0.1
    system_prompt = "Analyze the following content and determine if it is relevant to programming, software development, AI, or technology topics. Respond with a relevancy score between 0.0 (completely irrelevant) and 1.0 (highly relevant)."
    hybrid_mode = $true
    ai_weight = 0.7
}

try {
    $response = Invoke-RestMethod -Uri "$BASE_URL/api/relevancy/ai-provider" -Method POST -Body ($aiConfig | ConvertTo-Json) -ContentType "application/json"
    Write-Host "✅ Success!" -ForegroundColor Green
    Write-Host "Configuration saved:" -ForegroundColor White
    $response.data | ConvertTo-Json -Depth 3 | Write-Host
} catch {
    Write-Host "❌ Failed: $($_.Exception.Message)" -ForegroundColor Red
}

# Test 3: Test AI Provider with Relevant Content
Write-Host "`n3️⃣ Testing AI Provider with Programming Content..." -ForegroundColor Yellow
$testRequest = @{
    content = "How do I implement a binary search algorithm in Python? I need to optimize my code for better performance."
    provider = "openai"
    model = "gpt-4"
    api_key = "sk-test-key-12345"
}

try {
    $response = Invoke-RestMethod -Uri "$BASE_URL/api/relevancy/ai-test" -Method POST -Body ($testRequest | ConvertTo-Json) -ContentType "application/json"
    Write-Host "✅ Success!" -ForegroundColor Green
    Write-Host "Test Results:" -ForegroundColor White
    $response.data | ConvertTo-Json -Depth 3 | Write-Host
} catch {
    Write-Host "❌ Failed: $($_.Exception.Message)" -ForegroundColor Red
}

# Test 4: Test AI Provider with Irrelevant Content
Write-Host "`n4️⃣ Testing AI Provider with Cooking Content..." -ForegroundColor Yellow
$testRequest2 = @{
    content = "What's the best recipe for chocolate chip cookies? I love baking on weekends."
    provider = "openai"
    model = "gpt-4"
    api_key = "sk-test-key-12345"
}

try {
    $response = Invoke-RestMethod -Uri "$BASE_URL/api/relevancy/ai-test" -Method POST -Body ($testRequest2 | ConvertTo-Json) -ContentType "application/json"
    Write-Host "✅ Success!" -ForegroundColor Green
    Write-Host "Test Results:" -ForegroundColor White
    $response.data | ConvertTo-Json -Depth 3 | Write-Host
} catch {
    Write-Host "❌ Failed: $($_.Exception.Message)" -ForegroundColor Red
}

# Test 5: Configure Anthropic Provider
Write-Host "`n5️⃣ Configuring Anthropic Provider..." -ForegroundColor Yellow
$anthropicConfig = @{
    ai_enabled = $true
    provider = "anthropic"
    model = "claude-3-sonnet"
    api_key = "sk-ant-test-key-67890"
    endpoint = ""
    max_tokens = 1024
    temperature = 0.2
    system_prompt = "You are an expert at determining content relevancy for programming and technology topics. Analyze the given content and provide a relevancy score."
    hybrid_mode = $false
    ai_weight = 1.0
}

try {
    $response = Invoke-RestMethod -Uri "$BASE_URL/api/relevancy/ai-provider" -Method POST -Body ($anthropicConfig | ConvertTo-Json) -ContentType "application/json"
    Write-Host "✅ Success!" -ForegroundColor Green
    Write-Host "Anthropic configuration saved:" -ForegroundColor White
    $response.data | ConvertTo-Json -Depth 3 | Write-Host
} catch {
    Write-Host "❌ Failed: $($_.Exception.Message)" -ForegroundColor Red
}

# Test 6: Test Custom Endpoint Configuration
Write-Host "`n6️⃣ Configuring Custom Endpoint..." -ForegroundColor Yellow
$customConfig = @{
    ai_enabled = $true
    provider = "custom"
    model = "llama-2-7b"
    api_key = "custom-api-key"
    endpoint = "https://api.custom-llm.com/v1/chat/completions"
    max_tokens = 512
    temperature = 0.3
    system_prompt = "Determine if this content is about programming, software, or technology."
    hybrid_mode = $true
    ai_weight = 0.8
}

try {
    $response = Invoke-RestMethod -Uri "$BASE_URL/api/relevancy/ai-provider" -Method POST -Body ($customConfig | ConvertTo-Json) -ContentType "application/json"
    Write-Host "✅ Success!" -ForegroundColor Green
    Write-Host "Custom endpoint configuration saved:" -ForegroundColor White
    $response.data | ConvertTo-Json -Depth 3 | Write-Host
} catch {
    Write-Host "❌ Failed: $($_.Exception.Message)" -ForegroundColor Red
}

# Test 7: Error Handling - Invalid Configuration
Write-Host "`n7️⃣ Testing Error Handling (Invalid Config)..." -ForegroundColor Yellow
$invalidConfig = @{
    ai_enabled = $true
    provider = ""  # Missing provider
    model = ""     # Missing model
    api_key = ""   # Missing API key
}

try {
    $response = Invoke-RestMethod -Uri "$BASE_URL/api/relevancy/ai-provider" -Method POST -Body ($invalidConfig | ConvertTo-Json) -ContentType "application/json"
    Write-Host "❌ Expected error but got success" -ForegroundColor Red
} catch {
    Write-Host "✅ Correctly handled error: $($_.Exception.Message)" -ForegroundColor Green
}

# Test 8: Disable AI Provider
Write-Host "`n8️⃣ Disabling AI Provider..." -ForegroundColor Yellow
$disableConfig = @{
    ai_enabled = $false
    provider = ""
    model = ""
    api_key = ""
    endpoint = ""
    max_tokens = 2048
    temperature = 0.1
    system_prompt = ""
    hybrid_mode = $true
    ai_weight = 0.7
}

try {
    $response = Invoke-RestMethod -Uri "$BASE_URL/api/relevancy/ai-provider" -Method POST -Body ($disableConfig | ConvertTo-Json) -ContentType "application/json"
    Write-Host "✅ Success!" -ForegroundColor Green
    Write-Host "AI Provider disabled:" -ForegroundColor White
    $response.data | ConvertTo-Json -Depth 3 | Write-Host
} catch {
    Write-Host "❌ Failed: $($_.Exception.Message)" -ForegroundColor Red
}

# Test 9: Final Configuration Check
Write-Host "`n9️⃣ Final Configuration Check..." -ForegroundColor Yellow
try {
    $response = Invoke-RestMethod -Uri "$BASE_URL/api/relevancy/ai-provider" -Method GET
    Write-Host "✅ Success!" -ForegroundColor Green
    Write-Host "Final Configuration:" -ForegroundColor White
    $response.data | ConvertTo-Json -Depth 3 | Write-Host
} catch {
    Write-Host "❌ Failed: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host "`n🎉 AI Provider API Testing Complete!" -ForegroundColor Cyan
Write-Host "=" * 50

Write-Host "`n📋 Summary of AI Provider Features:" -ForegroundColor White
Write-Host "• Configure multiple AI providers (OpenAI, Anthropic, Local, Custom)" -ForegroundColor Gray
Write-Host "• Test AI provider connectivity and performance" -ForegroundColor Gray
Write-Host "• Hybrid mode combining AI and local keyword scoring" -ForegroundColor Gray
Write-Host "• Customizable system prompts and model parameters" -ForegroundColor Gray
Write-Host "• Real-time AI relevancy testing with detailed feedback" -ForegroundColor Gray
Write-Host "• Simulated responses for testing without API keys" -ForegroundColor Gray

Write-Host "`n🔗 Frontend Integration:" -ForegroundColor White
Write-Host "• Navigate to Advanced Configuration > Relevancy Layer tab" -ForegroundColor Gray
Write-Host "• Scroll down to see '🤖 AI-Powered Relevancy' section" -ForegroundColor Gray
Write-Host "• Enable AI, configure provider, and test in real-time" -ForegroundColor Gray 