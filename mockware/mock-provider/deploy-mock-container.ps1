#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Deployment script for QT-1 Mock Provider to Azure Container Instances (ACI).
.DESCRIPTION
    Builds the Docker image for the mock-provider, pushes it to Azure Container Registry (ACR),
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
    [string]$ContainerName     = "qt1-mock-provider",
    [string]$RegistryName      = "qt1acr",
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
# Current directory is mock-provider (contains Dockerfile)
$MockProviderDir = $PSScriptRoot                            # .../mockware/mock-provider
$Dockerfile = Join-Path $MockProviderDir "Dockerfile"       # .../mockware/mock-provider/Dockerfile
$ContextDir = $MockProviderDir                              # Use mock-provider as build context

if (-not (Test-Path $Dockerfile)) {
    Write-ErrorX "Dockerfile not found: $Dockerfile"; exit 1
}

# --- 1. Build Docker image ---
$timestamp = (Get-Date -Format "yyyyMMdd-HHmmss")
$imageTag  = "v$timestamp"
$imageName = "$RegistryName.azurecr.io/$ContainerName`:$imageTag"

Write-Info "Building mock-provider image $imageName…"
# Clear any existing local images first to ensure fresh build
try { 
    docker rmi qt1-mock-provider:latest 2>$null 
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

# --- 1. Azure login / subscription ---
Write-Info "Logging into Azure…"
Exec "az login"
Exec "az account set --subscription $SubscriptionId"

# --- 2. Ensure resource-group ---
Write-Info "Ensuring resource-group $ResourceGroupName…"
$rgExists = (az group exists --name $ResourceGroupName | ConvertFrom-Json)
if (-not $rgExists) { Exec "az group create --name $ResourceGroupName --location $Location" }

# --- 2b. Ensure Azure Container Registry ---
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

# --- 3. Push image ---
Write-Info "Logging into ACR $RegistryName…"
Exec "az acr login --name $RegistryName"

Write-Info "Pushing image…"
Exec "docker push $imageName"

# --- 4. Delete any existing container instance ---
Write-Info "Removing existing container (if any)…"
az container delete --yes --resource-group $ResourceGroupName --name $ContainerName 1>$null 2>$null

# --- 5. Deploy new container instance ---
Write-Info "Creating container instance…"

# Mock provider specific environment variables
$envArgs = @(
    "PORT=8081",
    "NODE_ENV=production",
    "RESPONSE_DELAY_MIN=100",
    "RESPONSE_DELAY_MAX=500",
    "ERROR_RATE=0.02",
    "RATE_LIMIT_MAX=1000",
    "RATE_LIMIT_WINDOW=60000",
    "TOKEN_RATE_INPUT=0.0001",
    "TOKEN_RATE_OUTPUT=0.0002",
    "ENABLE_STREAMING=true",
    "ENABLE_DETAILED_LOGGING=true"
) -join " "

# Fetch ACR admin password for `registry-password` argument
$RegistryPwd = (az acr credential show --name $RegistryName --query 'passwords[0].value' -o tsv)

Exec "az container create --resource-group $ResourceGroupName --name $ContainerName --image $imageName --registry-login-server $RegistryName.azurecr.io --registry-username $RegistryName --registry-password $RegistryPwd --dns-name-label $ContainerName --ports 8081 --cpu 1 --memory 1 --os-type Linux --restart-policy Always --environment-variables $envArgs --location $Location"

# --- 6. Output details ---
Write-Info "Fetching container details…"
$ip = az container show --resource-group $ResourceGroupName --name $ContainerName --query "ipAddress.ip" -o tsv
$fqdn = az container show --resource-group $ResourceGroupName --name $ContainerName --query "ipAddress.fqdn" -o tsv

Write-Host "\n🚀 Mock Provider Deployment Complete!" -ForegroundColor Green
Write-Host "Access your Mock Provider at:" -ForegroundColor Green
Write-Host "  • Health Check: http://$fqdn:8081/health" -ForegroundColor Yellow
Write-Host "  • OpenAI API:   http://$fqdn:8081/v1/chat/completions" -ForegroundColor Yellow
Write-Host "  • Direct IP:    http://$ip:8081" -ForegroundColor Yellow
Write-Host "\nTesting endpoints:" -ForegroundColor Green
Write-Host "  curl http://$fqdn:8081/health" -ForegroundColor Gray
