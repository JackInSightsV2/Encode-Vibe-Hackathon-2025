# Test OpenAI API Key
# Usage: .\test_openai_key.ps1 -apiKey "your-api-key-here"

param(
    [Parameter(Mandatory=$true)]
    [string]$apiKey
)

# API endpoint
$url = "https://api.openai.com/v1/chat/completions"

# Request headers
$headers = @{
    "Content-Type"  = "application/json"
    "Authorization" = "Bearer $apiKey"
}

# Request body
$body = @{
    model = "gpt-3.5-turbo"
    messages = @(
        @{
            role = "user"
            content = "Say this is a test!"
        }
    )
    max_tokens = 10
} | ConvertTo-Json

try {
    Write-Host "Testing OpenAI API key..."
    Write-Host "Key starts with: $($apiKey.Substring(0, [Math]::Min(10, $apiKey.Length)))..."
    Write-Host "Sending request to OpenAI API..."
    
    $response = Invoke-RestMethod -Uri $url -Method Post -Headers $headers -Body $body
    
    Write-Host "✅ Success! API key is valid." -ForegroundColor Green
    Write-Host "Response from OpenAI:"
    $response.choices[0].message.content
} 
catch {
    Write-Host "❌ Error occurred:" -ForegroundColor Red
    Write-Host $_.Exception.Message -ForegroundColor Red
    
    if ($_.Exception.Response) {
        $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
        $reader.BaseStream.Position = 0
        $reader.DiscardBufferedData()
        $responseBody = $reader.ReadToEnd()
        Write-Host "Response body: $responseBody" -ForegroundColor Red
    }
}

Write-Host "`nTo use this key in your application, set it in your environment:"
Write-Host "PowerShell: `$env:OPENAI_API_KEY = `"$apiKey`""
Write-Host "CMD: set OPENAI_API_KEY=$apiKey"
Write-Host "Or add it to your .env file: OPENAI_API_KEY=$apiKey"
