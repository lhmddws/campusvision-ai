#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Download ONNX models for face-recognition with mirror/proxy support.
.DESCRIPTION
    Tries multiple mirrors (hf-mirror.com, huggingface.co) with automatic
    fallback and retry. Supports proxy for restricted networks.
.EXAMPLE
    # Default (try all mirrors, 3 retries)
    .\scripts\download-models.ps1

    # Use specific mirror with proxy
    .\scripts\download-models.ps1 -Mirror "https://hf-mirror.com" -Proxy "http://127.0.0.1:7890"

    # More retries for slow networks
    .\scripts\download-models.ps1 -Retries 5
#>
param(
    [string]$Mirror = "",
    [string]$Proxy = "",
    [int]$Retries = 3,
    [int]$Timeout = 300,
    [switch]$Help
)

if ($Help) {
    Get-Help $PSCommandPath
    exit 0
}

$ProjectRoot = Split-Path -Parent (Split-Path -Parent $PSCommandPath)
Set-Location $ProjectRoot

$env:HF_ENDPOINT = if ($Mirror) { $Mirror } else { "https://hf-mirror.com" }

$argsList = @("--retries", $Retries, "--timeout", $Timeout)
if ($Mirror) { $argsList += @("--mirror", $Mirror) }
if ($Proxy) { $argsList += @("--proxy", $Proxy) }

Write-Host "╔══════════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║  CampusVision - Model Downloader             ║" -ForegroundColor Cyan
Write-Host "╠══════════════════════════════════════════════╣" -ForegroundColor Cyan
Write-Host "║  Mirror: $($env:HF_ENDPOINT)" -ForegroundColor Cyan
Write-Host "║  Retries: $Retries | Timeout: ${Timeout}s" -ForegroundColor Cyan
if ($Proxy) { Write-Host "║  Proxy: $Proxy" -ForegroundColor Cyan }
Write-Host "╚══════════════════════════════════════════════╝" -ForegroundColor Cyan

python -m app.download_models @argsList

if ($LASTEXITCODE -eq 0) {
    Write-Host "`n✅ Models downloaded successfully!" -ForegroundColor Green
    Write-Host "  Location: $ProjectRoot\app\models\" -ForegroundColor Green
    Get-ChildItem "$ProjectRoot\app\models\*.onnx" | ForEach-Object {
        Write-Host "  - $($_.Name) ($('{0:N1} MB' -f ($_.Length / 1MB)))" -ForegroundColor Green
    }
} else {
    Write-Host "`n❌ Download failed. Try:" -ForegroundColor Red
    Write-Host "  1. With proxy:   .\scripts\download-models.ps1 -Proxy http://127.0.0.1:7890" -ForegroundColor Yellow
    Write-Host "  2. Direct mirror: .\scripts\download-models.ps1 -Mirror https://huggingface.co" -ForegroundColor Yellow
    Write-Host "  3. Manual: Download *.onnx files and place in app\models\" -ForegroundColor Yellow
}
