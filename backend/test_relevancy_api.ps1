# QT-1 Relevancy API Test Script
# This demonstrates how to manage relevancy settings via API

$baseUrl = "http://localhost:8080"

Write-Host "🔧 QT-1 Relevancy API Test Script" -ForegroundColor Cyan
Write-Host "====================================" -ForegroundColor Cyan

# 1. Get current relevancy configuration
Write-Host "`n1. Getting current relevancy configuration..." -ForegroundColor Yellow
try {
    $response = Invoke-RestMethod -Uri "$baseUrl/api/relevancy/config" -Method GET
    Write-Host "✅ Current relevancy config:" -ForegroundColor Green
    $response | ConvertTo-Json -Depth 3
} catch {
    Write-Host "❌ Failed to get relevancy config: $($_.Exception.Message)" -ForegroundColor Red
}

# 2. Get all keywords organized by relevancy level
Write-Host "`n2. Getting keywords organized by relevancy..." -ForegroundColor Yellow
try {
    $keywords = Invoke-RestMethod -Uri "$baseUrl/api/relevancy/keywords" -Method GET
    Write-Host "✅ Keywords by category:" -ForegroundColor Green
    $keywords | ConvertTo-Json -Depth 3
} catch {
    Write-Host "❌ Failed to get keywords: $($_.Exception.Message)" -ForegroundColor Red
}

# 3. Test relevancy with sample content
Write-Host "`n3. Testing relevancy with sample content..." -ForegroundColor Yellow

$testCases = @(
    @{ content = "How do I implement machine learning in Python?"; expected = "relevant" }
    @{ content = "What's the weather like today?"; expected = "irrelevant" }
    @{ content = "Can you help me with my React application?"; expected = "relevant" }
    @{ content = "I want to cook pasta for dinner."; expected = "irrelevant" }
)

foreach ($test in $testCases) {
    try {
        $body = @{ content = $test.content } | ConvertTo-Json
        $result = Invoke-RestMethod -Uri "$baseUrl/api/relevancy/test" -Method POST -Body $body -ContentType "application/json"
        
        $status = if ($result.relevant) { "✅ RELEVANT" } else { "❌ IRRELEVANT" }
        Write-Host "Content: '$($test.content)'" -ForegroundColor Cyan
        Write-Host "Result: $status (score: $($result.relevancy_score.ToString('F3')), blocked: $($result.blocked))" -ForegroundColor White
        Write-Host "Reason: $($result.reason)" -ForegroundColor Gray
        Write-Host ""
    } catch {
        Write-Host "❌ Failed to test content: $($_.Exception.Message)" -ForegroundColor Red
    }
}

# 4. Add a custom keyword
Write-Host "`n4. Adding custom keyword..." -ForegroundColor Yellow
try {
    $body = @{ 
        keyword = "blockchain"
        score = 0.9 
    } | ConvertTo-Json
    
    $result = Invoke-RestMethod -Uri "$baseUrl/api/relevancy/keywords" -Method POST -Body $body -ContentType "application/json"
    Write-Host "✅ Added keyword: $($result.message)" -ForegroundColor Green
} catch {
    Write-Host "❌ Failed to add keyword: $($_.Exception.Message)" -ForegroundColor Red
}

# 5. Test with the new keyword
Write-Host "`n5. Testing with new keyword..." -ForegroundColor Yellow
try {
    $body = @{ content = "I want to learn about blockchain technology and cryptocurrency." } | ConvertTo-Json
    $result = Invoke-RestMethod -Uri "$baseUrl/api/relevancy/test" -Method POST -Body $body -ContentType "application/json"
    
    $status = if ($result.relevant) { "✅ RELEVANT" } else { "❌ IRRELEVANT" }
    Write-Host "Content: 'I want to learn about blockchain technology and cryptocurrency.'" -ForegroundColor Cyan
    Write-Host "Result: $status (score: $($result.relevancy_score.ToString('F3')))" -ForegroundColor White
    Write-Host "Matched keywords: $($result.details.matched_keywords -join ', ')" -ForegroundColor Gray
    Write-Host ""
} catch {
    Write-Host "❌ Failed to test with new keyword: $($_.Exception.Message)" -ForegroundColor Red
}

# 6. Update relevancy configuration
Write-Host "`n6. Updating relevancy configuration..." -ForegroundColor Yellow
try {
    $body = @{
        threshold = 0.4  # Change threshold to be less strict
        weight = 0.5     # Increase weight in moderation pipeline
    } | ConvertTo-Json
    
    $result = Invoke-RestMethod -Uri "$baseUrl/api/relevancy/config" -Method POST -Body $body -ContentType "application/json"
    Write-Host "✅ Updated config: $($result.message)" -ForegroundColor Green
} catch {
    Write-Host "❌ Failed to update config: $($_.Exception.Message)" -ForegroundColor Red
}

# 7. Remove the custom keyword
Write-Host "`n7. Removing custom keyword..." -ForegroundColor Yellow
try {
    $result = Invoke-RestMethod -Uri "$baseUrl/api/relevancy/keywords?keyword=blockchain" -Method DELETE
    Write-Host "✅ Removed keyword: $($result.message)" -ForegroundColor Green
} catch {
    Write-Host "❌ Failed to remove keyword: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host "`n🎉 Relevancy API test complete!" -ForegroundColor Green
Write-Host "`nExample curl commands:" -ForegroundColor Cyan
Write-Host @"
# Get relevancy config
curl -X GET http://localhost:8080/api/relevancy/config

# Test content relevancy
curl -X POST http://localhost:8080/api/relevancy/test \
  -H "Content-Type: application/json" \
  -d '{"content":"How do I code in Python?"}'

# Add a keyword
curl -X POST http://localhost:8080/api/relevancy/keywords \
  -H "Content-Type: application/json" \
  -d '{"keyword":"docker","score":0.8}'

# Remove a keyword
curl -X DELETE http://localhost:8080/api/relevancy/keywords?keyword=docker

# Update threshold
curl -X POST http://localhost:8080/api/relevancy/config \
  -H "Content-Type: application/json" \
  -d '{"threshold":0.4}'
"@ -ForegroundColor Gray 