# GitHub Proxy Build Script (PowerShell)
# Usage: .\build.ps1 [-Version "v1.0.0"]
# Example: .\build.ps1 v1.0.0

param(
    [string]$Version = "dev"
)

$ErrorActionPreference = "Stop"
$BuildDir = Join-Path $PSScriptRoot "build"
$SrcDir = Join-Path $PSScriptRoot "src"

Write-Host "============================================" -ForegroundColor Cyan
Write-Host " GitHub Proxy Build Script" -ForegroundColor Cyan
Write-Host " Version: $Version" -ForegroundColor Cyan
Write-Host "============================================" -ForegroundColor Cyan
Write-Host ""

# 1. Create build directory
if (-not (Test-Path $BuildDir)) {
    New-Item -ItemType Directory -Path $BuildDir | Out-Null
}
else {
    Get-ChildItem $BuildDir | Remove-Item -Recurse -Force
}

# 2. Build frontend
Write-Host "[1/4] Building frontend..." -ForegroundColor Yellow
Push-Location "$SrcDir/frontend"
try {
    npm install --silent 2>&1 | Out-Null
    npm run build --silent 2>&1 | Out-Null
    Write-Host "  Frontend built -> src/public/" -ForegroundColor Green
} finally {
    Pop-Location
}
Write-Host ""

# 3. Cross-compile Go backends
Push-Location $SrcDir
try {
    $env:CGO_ENABLED = "0"
    $BuildTime = (Get-Date).ToString("yyyy-MM-ddTHH-mm-ss")
    $LdFlags = "-s -w -X main.Version=$Version -X main.BuildTime=$BuildTime"

    # Linux amd64
    Write-Host "[2/4] Building Linux amd64..." -ForegroundColor Yellow
    $env:GOOS = "linux"
    $env:GOARCH = "amd64"
    go build "-ldflags=$LdFlags" -o "$BuildDir/github-proxy-linux-amd64" .
    Write-Host "  Done -> build/github-proxy-linux-amd64" -ForegroundColor Green
    Write-Host ""

    # Windows amd64
    Write-Host "[3/4] Building Windows amd64..." -ForegroundColor Yellow
    $env:GOOS = "windows"
    $env:GOARCH = "amd64"
    go build "-ldflags=$LdFlags" -o "$BuildDir/github-proxy-windows-amd64.exe" .
    Write-Host "  Done -> build/github-proxy-windows-amd64.exe" -ForegroundColor Green
    Write-Host ""
} finally {
    Pop-Location
}


Write-Host "============================================" -ForegroundColor Cyan
Write-Host " Build complete!" -ForegroundColor Green
Write-Host " Output: $BuildDir" -ForegroundColor Cyan
Write-Host ""
Write-Host "Files:" -ForegroundColor Cyan
Get-ChildItem $BuildDir | Format-Table Name, @{N='Size(MB)';E={[math]::Round($_.Length/1MB,2)}} -AutoSize
Write-Host "Docker build: docker build -t github-proxy:$Version ." -ForegroundColor Gray
Write-Host "============================================" -ForegroundColor Cyan
