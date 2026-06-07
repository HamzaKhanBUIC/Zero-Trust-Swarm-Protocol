# Pre-build binaries
$env:Path += ";C:\Users\Hamza Imran\go\bin"
go build -o idp-daemon.exe ./cmd/idp-daemon
go build -o registry.exe ./cmd/registry
go build -o agent.exe ./cmd/agent
go build -o sidecar.exe ./cmd/sidecar

# Start IdP
Start-Process -NoNewWindow -FilePath ".\idp-daemon.exe"
Start-Sleep -Seconds 3

# Start Registry
Start-Process -NoNewWindow -FilePath ".\registry.exe" -ArgumentList "--listen", "127.0.0.1:9000"
Start-Sleep -Seconds 3

# Start Agent Alpha (math-solver)
Start-Process -NoNewWindow -FilePath ".\agent.exe" -ArgumentList "--id", "agent-alpha", "--listen", "127.0.0.1:9101", "--caps", "math-solver", "--registry", "127.0.0.1:9000"
Start-Sleep -Seconds 3

# Start Sidecar Proxy
Start-Process -NoNewWindow -FilePath ".\sidecar.exe" -ArgumentList "--id", "agent-sidecar", "--listen", "127.0.0.1:8080", "--registry", "127.0.0.1:9000"
Start-Sleep -Seconds 3

# Run REST request to test the Sidecar
Write-Host "Sending HTTP REST request to Sidecar Proxy..." -ForegroundColor Cyan
$jsonPayload = @{
    model = "math-solver"
    messages = @(
        @{ role = "user"; content = "Evaluate 100 * (50 + 2)" }
    )
    max_tokens = 150
} | ConvertTo-Json

$response = Invoke-RestMethod -Uri "http://127.0.0.1:8080/v1/chat/completions" -Method Post -ContentType "application/json" -Body $jsonPayload
$response | ConvertTo-Json -Depth 5 | Write-Host

Write-Host "`nDemo Complete!" -ForegroundColor Green
