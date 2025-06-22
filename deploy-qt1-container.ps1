#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Deployment script for QT-1 Backend and Frontend to Azure Container Instances (ACI).
.DESCRIPTION
    Builds the Docker image for the QT-1 middleware (Go backend + React frontend), pushes it to Azure Container Registry (ACR),
    and deploys / updates an Azure Container Instance within the specified resource group.
.PARAMETER ResourceGroupName
    Azure resource group that contains (or will contain) the ACI instance.
.PARAMETER ContainerName
    Name for the ACI instance & ACR repository.
.PARAMETER RegistryName
    Existing Azure Container Registry name.
.PARAMETER SubscriptionId
    Azure subscription Id to target.
.PARAMETER Location
    Azure region. Default: uksouth.
#>
param(
    [string]$ResourceGroupName = "BIS-UKS-RG-P06",
    [string]$ContainerName     = "qt1-middleware",
    [string]$RegistryName      = "qt1acr1",
    [string]$SubscriptionId    = "e9f582ab-e625-4713-b459-904e482457bb",
    [string]$Location          = "uksouth"
)

# Utilities
function Write-Info   { param($m) Write-Host "[INFO]  $m"   -ForegroundColor Cyan }
function Write-Warn   { param($m) Write-Host "[WARN]  $m"   -ForegroundColor Yellow }
function Write-ErrorX { param($m) Write-Host "[ERROR] $m"  -ForegroundColor Red   }
function Exec($cmd)   {
    Write-Host ">> $cmd" -ForegroundColor DarkGray
    Invoke-Expression $cmd
    if ($LASTEXITCODE -ne 0) { Write-ErrorX "Command failed: $cmd"; exit 1 }
}

# --- 0. Paths ---
# Current directory should be the project root (contains Dockerfile, backend/, frontend/)
$ProjectRootDir = $PSScriptRoot                             # Project root directory
$Dockerfile = Join-Path $ProjectRootDir "Dockerfile"        # ./Dockerfile
$ContextDir = $ProjectRootDir                               # Use project root as build context

if (-not (Test-Path $Dockerfile)) {
    Write-ErrorX "Dockerfile not found: $Dockerfile"; exit 1
}

if (-not (Test-Path (Join-Path $ProjectRootDir "backend"))) {
    Write-ErrorX "Backend directory not found: $ProjectRootDir/backend"; exit 1
}

if (-not (Test-Path (Join-Path $ProjectRootDir "frontend"))) {
    Write-ErrorX "Frontend directory not found: $ProjectRootDir/frontend"; exit 1
}

# --- 1. Build Docker image ---
$timestamp = (Get-Date -Format "yyyyMMdd-HHmmss")
$imageTag  = "v$timestamp"
$imageName = "$RegistryName.azurecr.io/$ContainerName`:$imageTag"

Write-Info "Building QT-1 middleware image $imageName…"
# Clear any existing local images first to ensure fresh build
try { 
    docker rmi qt1-middleware:latest 2>$null 
    Write-Info "Removed existing local image"
} catch { 
    Write-Info "No existing image to remove" 
}
Exec "docker build --no-cache --pull -t $imageName -f $Dockerfile $ContextDir"

# Clean up dangling local images to keep disk usage low
Exec "docker image prune -f"

# Prompt for deployment
$deployChoice = Read-Host "Docker image built successfully. Do you want to deploy to Azure? (Y/N)"
if ($deployChoice -notmatch '^(?i)y$') {
    Write-Warn "Deployment skipped by user. Exiting."
    exit 0
}

Write-Info "User chose to deploy. Continuing…"

# --- 2. Azure login / subscription ---
Write-Info "Logging into Azure…"
Exec "az login"
Exec "az account set --subscription $SubscriptionId"

# --- 3. Ensure resource-group ---
Write-Info "Ensuring resource-group $ResourceGroupName…"
$rgExists = (az group exists --name $ResourceGroupName | ConvertFrom-Json)
if (-not $rgExists) { Exec "az group create --name $ResourceGroupName --location $Location" }

# --- 4. Ensure Azure Container Registry ---
Write-Info "Ensuring Azure Container Registry $RegistryName…"
$acrExists = $false
try {
    $null = az acr show --name $RegistryName --resource-group $ResourceGroupName --output none 2>$null
    if ($LASTEXITCODE -eq 0) { $acrExists = $true }
} catch { $acrExists = $false }

if (-not $acrExists) {
    Write-Info "Creating new ACR $RegistryName…"
    Exec "az acr create --resource-group $ResourceGroupName --name $RegistryName --sku Basic --location $Location --admin-enabled true"
} else {
    Write-Info "ACR $RegistryName already exists. Ensuring admin enabled…"
    $adminEnabled = az acr show --name $RegistryName --query "adminUserEnabled" -o tsv
    if ($adminEnabled -ne "true") { Exec "az acr update --name $RegistryName --admin-enabled true" }
}

# --- 5. Push image ---
Write-Info "Logging into ACR $RegistryName…"
Exec "az acr login --name $RegistryName"

Write-Info "Pushing image…"
Exec "docker push $imageName"

# --- 6. Delete any existing container instance ---
Write-Info "Removing existing container (if any)…"
az container delete --yes --resource-group $ResourceGroupName --name $ContainerName 1>$null 2>$null

# --- 7. Deploy new container instance ---
Write-Info "Creating container instance…"

# QT-1 Middleware specific environment variables
$envArgs = @(
    "PORT=8080",
    "QT1_HOST=0.0.0.0",
    "QT1_TARGET_URL=http://74.177.192.23:8081",
    "GIN_MODE=release",
    "DATABASE_URL=file:storage/qt1.db",
    "LOG_LEVEL=info",
    "CORS_ORIGINS=*",
    "CORS_ALLOW_ALL_ORIGINS=true",
    "SECURITY_IP_PROTECTION_ENABLED=false",
    "SECURITY_DDOS_PROTECTION_ENABLED=false",
    "SECURITY_RATE_LIMITING_ENABLED=false",
    "SECURITY_GEOBLOCKING_ENABLED=false",
    "SECURITY_BLOCK_SUSPICIOUS_IPS=false",
    "RATE_LIMIT_ENABLED=false",
    "DDOS_PROTECTION_ENABLED=false",
    "MODERATION_ENABLED=true",
    "MOCK_PROVIDER_BASE_URL=http://74.177.192.23:8081",
    "OPIK_URL=",
    "OPIK_WORKSPACE=default"
) -join " "

# Fetch ACR admin password for `registry-password` argument
$RegistryPwd = (az acr credential show --name $RegistryName --query 'passwords[0].value' -o tsv)

Exec "az container create --resource-group $ResourceGroupName --name $ContainerName --image $imageName --registry-login-server $RegistryName.azurecr.io --registry-username $RegistryName --registry-password $RegistryPwd --dns-name-label $ContainerName --ports 8080 --cpu 2 --memory 4 --os-type Linux --restart-policy Always --environment-variables $envArgs --location $Location"

# --- 8. Output details ---
Write-Info "Fetching container details…"
$ip = az container show --resource-group $ResourceGroupName --name $ContainerName --query "ipAddress.ip" -o tsv
$fqdn = az container show --resource-group $ResourceGroupName --name $ContainerName --query "ipAddress.fqdn" -o tsv

Write-Host "\n🚀 QT-1 Middleware Deployment Complete!" -ForegroundColor Green
Write-Host "Access your QT-1 Middleware at:" -ForegroundColor Green
Write-Host "  • Frontend Dashboard: http://$fqdn:8080" -ForegroundColor Yellow
Write-Host "  • API Health Check:   http://$fqdn:8080/health" -ForegroundColor Yellow
Write-Host "  • Chat API:           http://$fqdn:8080/chat" -ForegroundColor Yellow
Write-Host "  • Direct IP:          http://$ip:8080" -ForegroundColor Yellow
Write-Host "\nTesting endpoints:" -ForegroundColor Green
Write-Host "  curl http://$fqdn:8080/health" -ForegroundColor Gray
Write-Host "  curl -X POST http://$fqdn:8080/chat -H 'Content-Type: application/json' -d '{\"message\":\"Hello\"}'" -ForegroundColor Gray
