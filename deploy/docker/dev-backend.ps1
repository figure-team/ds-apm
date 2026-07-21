#!/usr/bin/env pwsh
# Fast backend dev loop: cross-compile the Go server on the Windows host and
# hot-swap it into the running `signoz` container — no Docker build, no yarn.
#
#   ./deploy/docker/dev-backend.ps1
#
# First run only: the base image must exist. If it doesn't, build it once with
#   docker compose -f deploy/docker/docker-compose.local.yaml up -d --build signoz
# then use this script for every subsequent change.

$ErrorActionPreference = 'Stop'

# Run from repo root (go.mod lives there) regardless of caller's location.
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot '..' '..')
Set-Location $repoRoot

$outDir = Join-Path $PSScriptRoot '.bin'
$outBin = Join-Path $outDir 'signoz'
New-Item -ItemType Directory -Force -Path $outDir | Out-Null

Write-Host '==> Cross-compiling signoz (linux/amd64) on host...' -ForegroundColor Cyan
$env:CGO_ENABLED = '0'
$env:GOOS        = 'linux'
$env:GOARCH      = 'amd64'
# Pin to the go.mod toolchain (matches golang:1.25-alpine in Dockerfile.local).
# Newer host Go (1.26+) breaks bytedance/sonic v1.14.1 (undefined GoMapIterator),
# so don't let GOTOOLCHAIN=auto fall through to the local toolchain.
$env:GOTOOLCHAIN = 'go1.25.7'
# 버전 미주입 시 사이드바에 <unset>이 노출되므로 Makefile과 같은 -X 주입을 재현한다.
$version   = (git describe --tags --always --dirty 2>$null); if (-not $version) { $version = 'dev' }
$hash      = (git rev-parse --short HEAD 2>$null); if (-not $hash) { $hash = 'unknown' }
$branch    = (git rev-parse --abbrev-ref HEAD 2>$null); if (-not $branch) { $branch = 'unknown' }
$timestamp = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
$verPkg    = 'github.com/SigNoz/signoz/pkg/version'
$ldflags   = "-s -w -X $verPkg.variant=community -X $verPkg.version=$version -X $verPkg.hash=$hash -X $verPkg.time=$timestamp -X $verPkg.branch=$branch"

# -v prints each package as it compiles so a long cold build shows progress
# instead of looking hung (go build is otherwise silent).
go build -tags timetzdata -v `
  -ldflags $ldflags `
  -o $outBin ./cmd/community
if ($LASTEXITCODE -ne 0) { throw 'go build failed' }

$compose = @('-f', 'deploy/docker/docker-compose.local.yaml')

Write-Host '==> Ensuring signoz container exists (no build)...' -ForegroundColor Cyan
docker compose @compose up -d --no-build signoz
if ($LASTEXITCODE -ne 0) { throw 'docker compose up failed (build the base image once with --build)' }

# Stop before copying: the running server holds /root/signoz open, so an
# in-place `docker cp` over the live binary would fail with "text file busy".
Write-Host '==> Stopping signoz to swap the binary...' -ForegroundColor Cyan
docker stop signoz | Out-Null

Write-Host '==> Copying freshly built binary into the container...' -ForegroundColor Cyan
docker cp $outBin signoz:/root/signoz
if ($LASTEXITCODE -ne 0) { throw 'docker cp failed' }

Write-Host '==> Starting signoz with the new binary...' -ForegroundColor Cyan
docker start signoz | Out-Null

Write-Host '==> Done. API on :8080, frontend dev server on :3301.' -ForegroundColor Green
