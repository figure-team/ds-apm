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
go build -tags timetzdata `
  -ldflags '-s -w -X github.com/SigNoz/signoz/pkg/version.variant=community' `
  -o $outBin ./cmd/community
if ($LASTEXITCODE -ne 0) { throw 'go build failed' }

$compose = @(
  '-f', 'deploy/docker/docker-compose.local.yaml',
  '-f', 'deploy/docker/docker-compose.dev.yaml'
)

Write-Host '==> Ensuring signoz container is up (no build)...' -ForegroundColor Cyan
docker compose @compose up -d --no-build signoz
if ($LASTEXITCODE -ne 0) { throw 'docker compose up failed (build the base image once with --build)' }

Write-Host '==> Restarting signoz to load the new binary...' -ForegroundColor Cyan
docker restart signoz | Out-Null

Write-Host '==> Done. API on :8080, frontend dev server on :3301.' -ForegroundColor Green
