# Test API endpoints
Write-Host "Testing QT-1 Middleware API endpoints..."

# Test login
Write-Host "`nTesting login..."
$loginData = @{
    username = "admin"
    password = "AdminPassword123!"
} | ConvertTo-Json

try {
    $loginResponse = Invoke-RestMethod -Uri "http://localhost:8080/api/auth/login" -Method POST -ContentType "application/json" -Body $loginData
    Write-Host "Login successful!" -ForegroundColor Green
    Write-Host "Full login response: $($loginResponse | ConvertTo-Json -Depth 5)"
    
    # Try different token field names
    $token = $null
    if ($loginResponse.token) {
        $token = $loginResponse.token
        Write-Host "Found token in .token field"
    } elseif ($loginResponse.data -and $loginResponse.data.token) {
        $token = $loginResponse.data.token
        Write-Host "Found token in .data.token field"
    } elseif ($loginResponse.access_token) {
        $token = $loginResponse.access_token
        Write-Host "Found token in .access_token field"
    } else {
        Write-Host "No token found in response!" -ForegroundColor Red
        Write-Host "Available fields: $($loginResponse.PSObject.Properties.Name -join ', ')"
    }
    
    Write-Host "Token: $token"
} catch {
    Write-Host "Login failed: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

if (-not $token) {
    Write-Host "Cannot continue without token" -ForegroundColor Red
    exit 1
}

# Test users endpoint
Write-Host "`nTesting /api/users endpoint..."
try {
    $headers = @{
        "Authorization" = "Bearer $token"
        "Content-Type" = "application/json"
    }
    $usersResponse = Invoke-RestMethod -Uri "http://localhost:8080/api/users" -Method GET -Headers $headers
    Write-Host "Users endpoint successful!" -ForegroundColor Green
    Write-Host "Response: $($usersResponse | ConvertTo-Json -Depth 3)"
} catch {
    Write-Host "Users endpoint failed: $($_.Exception.Message)" -ForegroundColor Red
}

# Test IP protection endpoint
Write-Host "`nTesting /api/ip-protection/status endpoint..."
try {
    $ipResponse = Invoke-RestMethod -Uri "http://localhost:8080/api/ip-protection/status" -Method GET -Headers $headers
    Write-Host "IP Protection endpoint successful!" -ForegroundColor Green
    Write-Host "Response: $($ipResponse | ConvertTo-Json -Depth 3)"
} catch {
    Write-Host "IP Protection endpoint failed: $($_.Exception.Message)" -ForegroundColor Red
}

# Test providers endpoint
Write-Host "`nTesting /api/providers endpoint..."
try {
    $providersResponse = Invoke-RestMethod -Uri "http://localhost:8080/api/providers" -Method GET -Headers $headers
    Write-Host "Providers endpoint successful!" -ForegroundColor Green
    Write-Host "Response: $($providersResponse | ConvertTo-Json -Depth 3)"
} catch {
    Write-Host "Providers endpoint failed: $($_.Exception.Message)" -ForegroundColor Red
}

# Test rules endpoint
Write-Host "`nTesting /api/rules endpoint..."
try {
    $rulesResponse = Invoke-RestMethod -Uri "http://localhost:8080/api/rules" -Method GET -Headers $headers
    Write-Host "Rules endpoint successful!" -ForegroundColor Green
    Write-Host "Response: $($rulesResponse | ConvertTo-Json -Depth 3)"
} catch {
    Write-Host "Rules endpoint failed: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host "`nAPI testing complete!" 