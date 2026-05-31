#!/bin/bash
set -e

echo -e "\033[1;36m==========================================================\033[0m"
echo -e "\033[1;36m   Zero-Trust Swarm Protocol - 1-Click Local Demo   \033[0m"
echo -e "\033[1;36m==========================================================\033[0m"
echo ""

# Tidy up modules
echo -e "\033[1;33m📦 Ensuring Go modules are ready...\033[0m"
go mod tidy

# Start IdP Daemon
echo -e "\033[1;32m🚀 Starting IdP Daemon (Workload Identity) in background...\033[0m"
go run ./cmd/idp-daemon &
IDP_PID=$!
sleep 3

# Start Registry
echo -e "\033[1;32m🚀 Starting Swarm Registry in background...\033[0m"
go run ./cmd/registry --listen 127.0.0.1:9000 &
REG_PID=$!
sleep 3

# Start Agent Alpha
echo -e "\033[1;32m🚀 Starting Agent Alpha (math-solver) in background...\033[0m"
go run ./cmd/agent --id agent-alpha --listen 127.0.0.1:9101 --caps math-solver --registry 127.0.0.1:9000 &
ALPHA_PID=$!
sleep 3

# Start Agent Beta
echo -e "\033[1;32m🚀 Starting Agent Beta to delegate task to math-solver...\033[0m"
go run ./cmd/agent --id agent-beta --listen 127.0.0.1:9102 --registry 127.0.0.1:9000 --target-cap math-solver --task "Evaluate 3 * (5 + 7)"

echo ""
echo -e "\033[1;36m✅ Demo Complete! The Swarm successfully executed the zero-trust workflow.\033[0m"
echo -e "\033[1;33m⚠️  Cleaning up background processes...\033[0m"
kill $ALPHA_PID $REG_PID $IDP_PID
echo -e "\033[1;32mCleanup complete. Goodbye!\033[0m"
