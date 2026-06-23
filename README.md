<div align="center">
  <h1>⚡ Zero-Trust Swarm Protocol</h1>
  <p><strong>Secure, Scalable mTLS Inter-Agent Communication Network</strong></p>
</div>

---

## 📖 Overview

The **Zero-Trust Swarm Protocol** is a secure, decentralized architecture for autonomous AI agents to communicate via enforced mTLS 1.3. 

## 🚀 Features

1. **Swarm Visualizer Dashboard**: A React Flow based UI to monitor active agents, capabilities, and network events over Server-Sent Events (SSE).
2. **Persistent Task Queues**: Store-and-Forward background task queues powered by a pure-Go SQLite driver.
3. **Python SDK**: Native `swarm-mtls` Python SDK to synchronously or asynchronously communicate with the Go Swarm Registry.
4. **Expanded Sidecar API**: Local API to interface LLMs into the secure Swarm.

---

## 🛠️ Tech Stack

- **Backend**: Go (Golang)
- **Database**: SQLite (`glebarez/go-sqlite`)
- **Frontend Dashboard**: React, Vite, React Flow
- **SDK**: Python
- **Containerization**: Docker Compose

---

## 💻 Usage

To launch the automation engine, simply run:

```bash
docker-compose up --build
```

Then visit `http://127.0.0.1:3000` to view the Dashboard.

---

## ⚠️ Troubleshooting

**Windows Users (Screen Stuttering during `docker-compose up --build`):**
If you experience your mouse or screen stuttering while compiling the Go microservices on Windows, don't worry—your hardware is safe! This is a known issue where Docker Desktop's WSL2 backend (`vmmem`) temporarily consumes large amounts of RAM during heavy compilation tasks. 

**Fix:** Create a `.wslconfig` file in your Windows user directory (`C:\Users\YourName\.wslconfig`) with the following limits:
```ini
[wsl2]
memory=4GB
processors=2
```
