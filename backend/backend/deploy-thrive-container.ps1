#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Minimal deployment script for Thrive 3.0 FastAPI backend to Azure Container Instances (ACI).
.DESCRIPTION
    Builds the Docker image located in ../backend, pushes it to Azure Container Registry (ACR),
    and deploys / updates an Azure Container Instance within the specified resource group.

    The script is intentionally simple — no KeyVault, HTTPS custom-domain checks, or repository cleanup.
.PARAMETER ResourceGroupName
    Azure resource group that contains (or will contain) the ACI instance.
.PARAMETER ContainerName
    Name for the ACI instance & ACR repository.
.PARAMETER RegistryName
    Existing Azure Container Registry name (e.g. "biwebcreg").
.PARAMETER SubscriptionId
    Azure subscription Id to target.
.PARAMETER Location
    Azure region. Default: uksouth.
#>
param(
    [string]$ResourceGroupName = "BIS-UKS-RG-P08",
    [string]$ContainerName     = "thrive-backend",
    [string]$RegistryName      = "thriveacr",
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
# backend directory (contains Dockerfile)
$BackendDir = (Get-Item $PSScriptRoot).Parent.FullName      # .../backend
# repo root is one level above backend
$RepoRoot   = (Get-Item $BackendDir).Parent.FullName        # .../Thrive-3.0-v2-Backend

$Dockerfile = Join-Path $BackendDir "Dockerfile"            # .../backend/Dockerfile
$ContextDir = $RepoRoot                                     # send whole repo as build-context so Dockerfile paths work

if (-not (Test-Path $Dockerfile)) {
    Write-ErrorX "Dockerfile not found: $Dockerfile"; exit 1
}

# --- 1. Build Docker image ---
$timestamp = (Get-Date -Format "yyyyMMdd-HHmmss")
$imageTag  = "v$timestamp"
$imageName = "$RegistryName.azurecr.io/$ContainerName`:$imageTag"

Write-Info "Building image $imageName…"
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
$envArgs = @(
    "EXPO_PUBLIC_SUPABASE_URL=$Env:EXPO_PUBLIC_SUPABASE_URL",
    "EXPO_PUBLIC_SUPABASE_ANON_KEY=$Env:EXPO_PUBLIC_SUPABASE_ANON_KEY",
    "SUPABASE_SERVICE_ROLE_KEY=$Env:SUPABASE_SERVICE_ROLE_KEY"
) -join " "

# Fetch ACR admin password for `registry-password` argument
$RegistryPwd = (az acr credential show --name $RegistryName --query 'passwords[0].value' -o tsv)

Exec "az container create --resource-group $ResourceGroupName --name $ContainerName --image $imageName --registry-login-server $RegistryName.azurecr.io --registry-username $RegistryName --registry-password $RegistryPwd --dns-name-label $ContainerName --ports 8000 --cpu 1 --memory 1 --os-type Linux --restart-policy Always --environment-variables $envArgs --location $Location"

# --- 6. Output details ---
Write-Info "Fetching container IP…"
$ip = az container show --resource-group $ResourceGroupName --name $ContainerName --query "ipAddress.ip" -o tsv
Write-Host "\nDeployment complete! Access your API at: http://$ip:8000/health" -ForegroundColor Green
