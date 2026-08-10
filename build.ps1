# GitHub Proxy Build Script (PowerShell)
# Usage: .\build.ps1 [-Version "v1.0.0"]
# Example: .\build.ps1 v1.0.0
#
# 流程：构建前端（web/ → cmd/github-proxy/public，供 embed 打包）→
#       交叉编译 Go 二进制（./cmd/github-proxy）→ 复制配置到 build/。

param(
    [string]$Version = "dev"
)

$ErrorActionPreference = "Stop"
$RootDir = $PSScriptRoot
$BuildDir = Join-Path $RootDir "build"

Write-Host "============================================" -ForegroundColor Cyan
Write-Host " GitHub Proxy Build Script" -ForegroundColor Cyan
Write-Host " Version: $Version" -ForegroundColor Cyan
Write-Host "============================================" -ForegroundColor Cyan
Write-Host ""

# 1. Create build directory
if (-not (Test-Path $BuildDir)) {
    New-Item -ItemType Directory -Path $BuildDir | Out-Null
} else {
    Get-ChildItem $BuildDir -ErrorAction SilentlyContinue | ForEach-Object {
        Remove-Item $_.FullName -Recurse -Force -ErrorAction SilentlyContinue
    }
}

# 2. Build frontend (输出到 cmd/github-proxy/public，由 //go:embed 打包)
Write-Host "[1/4] Building frontend..." -ForegroundColor Yellow
Push-Location (Join-Path $RootDir "web")
try {
    npm install --silent 2>&1 | Out-Null
    npm run build --silent 2>&1 | Out-Null
    Write-Host "  Frontend built -> cmd/github-proxy/public/" -ForegroundColor Green
} finally {
    Pop-Location
}
Write-Host ""

# 3. Cross-compile Go backends (模块根 = 仓库根)
$env:CGO_ENABLED = "0"
$BuildTime = (Get-Date).ToString("yyyy-MM-ddTHH-mm-ss")
$LdFlags = "-s -w -X main.Version=$Version -X main.BuildTime=$BuildTime"

# Linux amd64
Write-Host "[2/4] Building Linux amd64..." -ForegroundColor Yellow
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build "-ldflags=$LdFlags" -o "$BuildDir/github-proxy-linux-amd64" ./cmd/github-proxy
Write-Host "  Done -> build/github-proxy-linux-amd64" -ForegroundColor Green
Write-Host ""

# Windows amd64
Write-Host "[3/4] Building Windows amd64..." -ForegroundColor Yellow
$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build "-ldflags=$LdFlags" -o "$BuildDir/github-proxy-windows-amd64.exe" ./cmd/github-proxy
Write-Host "  Done -> build/github-proxy-windows-amd64.exe" -ForegroundColor Green
Write-Host ""

# 4. Copy config
Write-Host "[4/4] Copying config..." -ForegroundColor Yellow
Copy-Item (Join-Path $RootDir "config.toml") $BuildDir
Write-Host "  Config copied to build/" -ForegroundColor Green
Write-Host ""

Write-Host "============================================" -ForegroundColor Cyan
Write-Host " Build complete!" -ForegroundColor Green
Write-Host " Output: $BuildDir" -ForegroundColor Cyan
Write-Host ""
Write-Host "Files:" -ForegroundColor Cyan
Get-ChildItem $BuildDir | Format-Table Name, @{N='Size(MB)';E={[math]::Round($_.Length/1MB,2)}} -AutoSize
Write-Host "Docker build: docker build -t github-proxy:$Version ." -ForegroundColor Gray
Write-Host "============================================" -ForegroundColor Cyan
