# Test additional API endpoints
Write-Host "Testing additional QT-1 Middleware API endpoints..."

# Login first
$loginData = @{
    username = "admin"
    password = "AdminPassword123!"
} | ConvertTo-Json

$loginResponse = Invoke-RestMethod -Uri "http://localhost:8080/api/auth/login" -Method POST -ContentType "application/json" -Body $loginData
$token = $loginResponse.data.token
$headers = @{
    "Authorization" = "Bearer $token"
    "Content-Type" = "application/json"
}

# Test DDoS Protection
Write-Host "`nTesting /api/ddos-protection/status endpoint..."
try {
    $ddosResponse = Invoke-RestMethod -Uri "http://localhost:8080/api/ddos-protection/status" -Method GET -Headers $headers
    Write-Host "DDoS Protection endpoint successful!" -ForegroundColor Green
    Write-Host "Response: $($ddosResponse | ConvertTo-Json -Depth 3)"
} catch {
    Write-Host "DDoS Protection endpoint failed: $($_.Exception.Message)" -ForegroundColor Red
}

# Test Rate Limits
Write-Host "`nTesting /api/rate-limits endpoint..."
try {
    $rateLimitsResponse = Invoke-RestMethod -Uri "http://localhost:8080/api/rate-limits" -Method GET -Headers $headers
    Write-Host "Rate Limits endpoint successful!" -ForegroundColor Green
    Write-Host "Response: $($rateLimitsResponse | ConvertTo-Json -Depth 3)"
} catch {
    Write-Host "Rate Limits endpoint failed: $($_.Exception.Message)" -ForegroundColor Red
}

# Test Optimizer
Write-Host "`nTesting /api/optimizer/metrics endpoint..."
try {
    $optimizerResponse = Invoke-RestMethod -Uri "http://localhost:8080/api/optimizer/metrics" -Method GET -Headers $headers
    Write-Host "Optimizer endpoint successful!" -ForegroundColor Green
    Write-Host "Response: $($optimizerResponse | ConvertTo-Json -Depth 3)"
} catch {
    Write-Host "Optimizer endpoint failed: $($_.Exception.Message)" -ForegroundColor Red
}

# Test Configuration
Write-Host "`nTesting /api/config/status endpoint..."
try {
    $configResponse = Invoke-RestMethod -Uri "http://localhost:8080/api/config/status" -Method GET -Headers $headers
    Write-Host "Configuration endpoint successful!" -ForegroundColor Green
    Write-Host "Response: $($configResponse | ConvertTo-Json -Depth 3)"
} catch {
    Write-Host "Configuration endpoint failed: $($_.Exception.Message)" -ForegroundColor Red
}

# Test Kill Switch
Write-Host "`nTesting /api/killswitch/status endpoint..."
try {
    $killswitchResponse = Invoke-RestMethod -Uri "http://localhost:8080/api/killswitch/status" -Method GET -Headers $headers
    Write-Host "Kill Switch endpoint successful!" -ForegroundColor Green
    Write-Host "Response: $($killswitchResponse | ConvertTo-Json -Depth 3)"
} catch {
    Write-Host "Kill Switch endpoint failed: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host "`nAdditional API testing complete!" 