$ErrorActionPreference = "Stop"

Write-Host "==========================================================" -ForegroundColor Cyan
Write-Host "   Zero-Trust Swarm Protocol - 1-Click Local Demo   " -ForegroundColor Cyan
Write-Host "==========================================================" -ForegroundColor Cyan
Write-Host ""

# Ensure we're in the right directory
$ScriptDir = Split-Path $MyInvocation.MyCommand.Path
Set-Location $ScriptDir

# Tidy up modules
Write-Host "[INIT] Ensuring Go modules are ready..." -ForegroundColor Yellow
go mod tidy

# Pre-build binaries to prevent startup race conditions
Write-Host "[INIT] Compiling Swarm Binaries..." -ForegroundColor Yellow
go build -o idp-daemon.exe ./cmd/idp-daemon
go build -o registry.exe ./cmd/registry
go build -o agent.exe ./cmd/agent

# Start IdP Daemon
Write-Host "[START] Starting IdP Daemon (Workload Identity) in background..." -ForegroundColor Green
Start-Process -NoNewWindow -FilePath ".\idp-daemon.exe"
Start-Sleep -Seconds 3

# Start Registry
Write-Host "[START] Starting Swarm Registry in background..." -ForegroundColor Green
Start-Process -NoNewWindow -FilePath ".\registry.exe" -ArgumentList "--listen", "127.0.0.1:9000"
Start-Sleep -Seconds 3

# Start Agent Alpha (Capability: math-solver)
Write-Host "[START] Starting Agent Alpha (math-solver) in background..." -ForegroundColor Green
Start-Process -NoNewWindow -FilePath ".\agent.exe" -ArgumentList "--id", "agent-alpha", "--listen", "127.0.0.1:9101", "--caps", "math-solver", "--registry", "127.0.0.1:9000"
Start-Sleep -Seconds 3

# Start Agent Beta (Task delegation)
Write-Host "[START] Starting Agent Beta to delegate task to math-solver..." -ForegroundColor Green
.\agent.exe --id agent-beta --listen 127.0.0.1:9102 --registry 127.0.0.1:9000 --target-cap math-solver --task "Evaluate 3 * (5 + 7)"

Write-Host ""
Write-Host "[SUCCESS] Demo Complete! The Swarm successfully executed the zero-trust workflow." -ForegroundColor Cyan
Write-Host "[NOTE] The background processes (IdP, Registry, Alpha) are still running in this terminal session." -ForegroundColor Yellow
Write-Host "To stop them, you can close this window or use Task Manager." -ForegroundColor Yellow
