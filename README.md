<div align="center">

<img src="https://capsule-render.vercel.app/api?type=waving&color=0:000000,50:00ADD8,100:000000&height=180&section=header&text=Zero-Trust%20Swarm%20Protocol&fontSize=50&fontColor=ffffff&animation=twinkling" width="100%" alt="SwarmTLS Header" />

# 🐝🛡️ Zero-Trust Swarm Protocol

[![Open Source](https://img.shields.io/badge/Open%20Source-100%25-green?style=for-the-badge&logo=open-source-initiative)](https://github.com/HamzaKhanBUIC/Zero-Trust-Swarm-Protocol)
[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=for-the-badge&logo=go)](https://go.dev/)
[![mTLS 1.3](https://img.shields.io/badge/Transport-mTLS_1.3-4B0082?style=for-the-badge&logo=lock)](https://datatracker.ietf.org/doc/html/rfc8446)
[![ECDSA Signatures](https://img.shields.io/badge/Integrity-ECDSA_Signed-FF8C00?style=for-the-badge&logo=shield)](https://en.wikipedia.org/wiki/Elliptic_Curve_Digital_Signature_Algorithm)
[![SPIFFE](https://img.shields.io/badge/Identity-SPIFFE-green?style=for-the-badge&logo=spiffe)](https://spiffe.io/)

> 👋 **Welcome to the Zero-Trust Swarm!** This project is entirely **Open Source**. We invite developers, security engineers, and AI researchers from around the world to clone the repo, try out the 1-click demo, and build the future of secure autonomous AI orchestration with us. Jump down to the [Contributing](#-contributing) section to get involved!

</div>

## 🚀 The Vision

*A mathematically secure, decentralized transport layer for autonomous AI agents operating in hostile networks.*

---

</div>

## 🤯 Why This Breaks the "Agentic Trend"

Most modern AI agent frameworks (LangChain, AutoGen, CrewAI) focus almost exclusively on application-layer reasoning. They assume a perfectly safe, monolithic execution environment or a trusted VPC. When they communicate over a network, they typically pass unencrypted, unsigned JSON over standard HTTP/REST or WebSockets.

**This is fundamentally broken for autonomous systems.** If agents are to be truly autonomous, they will operate in hostile environments. They cannot blindly execute an instruction just because it hit their open port.

The Zero-Trust Swarm Protocol shifts the paradigm from *"smart agents in a dumb, trusting network"* to *"smart agents in a mathematically secure, adversarial network."* By enforcing identity, authorization, and integrity at the lowest possible layer (raw TCP + mTLS + ECDSA), we ensure that an AI agent only executes tasks from mathematically verified peers, making the swarm completely immune to spoofing, MITM, and unauthorized instruction injection.

---

## 🏗️ Architecture Deep Dive

A technical breakdown for distributed systems and security engineers.

### 1. The Core Philosophy
The Zero-Trust Swarm Protocol abandons perimeter-based security and static secrets in favor of ephemeral cryptographic identities, mutually authenticated transport (mTLS 1.3), and decentralized gossip routing.

### 2. Ephemeral Workload Identity (IdP)
Instead of distributing static X.509 `.pem` and `.key` files, the system employs a localized Workload Identity Provider (`idp-daemon`) heavily inspired by the SPIFFE standard.

*   **Bootstrapping**: The `idp-daemon` generates an RSA-4096 Root CA entirely in memory on startup.
*   **Attestation**: When an agent binary executes, it hits the `idp-daemon` over a strict `127.0.0.1` loopback API, asserting its identity (`--id`).
*   **Issuance**: The IdP dynamically generates a leaf certificate with the Subject Alternative Name (SAN) formatted as a SPIFFE URI (e.g., `spiffe://swarm.local/agent/math-solver`). The certificate and private key are streamed back to the agent and held strictly in RAM.
*   **Security Posture**: Because no private keys touch the disk, local credential harvesting malware cannot steal an agent's identity.

### 3. Deep-Packet Inspection (DPI) Evasion
Enterprise firewalls and consumer AVs (like Norton 360) frequently execute Man-in-the-Middle (MITM) attacks on local TLS connections to inspect traffic by sniffing the `ClientHello` bytes. This breaks strict mTLS.

To bypass this without disabling security software, we implemented a custom `net.Listener` and `net.Conn` wrapper (`transport.SwarmTLS`):
*   **Prefix Masking**: Before the TLS handshake begins, the client transmits a custom 11-byte plaintext prefix: `SWARM-mTLS\n`.
*   **AV Confusion**: Standard DPI engines parsing the TCP stream fail to recognize the initial bytes as a valid TLS `ClientHello` (which typically starts with `0x16 0x03`). The AV assumes it's an unknown proprietary protocol and ignores the socket.
*   **The Handshake**: The receiving agent reads and strips the 11-byte prefix, and seamlessly hands the remaining raw byte stream to the standard Go `crypto/tls` engine, successfully completing an un-intercepted mTLS 1.3 handshake.

### 4. Embedded Zero-Trust Authorization (ABAC)
Once the mTLS tunnel is established, encryption and identity are guaranteed, but authorization is not. We embed an extremely lightweight, regex-capable rule engine directly into the agents.

```go
policy.Rule{
    Effect:     policy.Allow,
    Principals: []string{"spiffe://swarm.local/agent/*"},
    Actions:    []protocol.MessageType{protocol.TypeTask, protocol.TypeGossip},
}
```
If a peer connects and successfully completes the mTLS handshake, their verified SPIFFE ID is instantly matched against the local authorization engine. It operates on a strict Default-Deny basis. If the peer is not authorized, the connection is aggressively dropped.

### 5. Cryptographic Payload Integrity (ECDSA)
To prevent man-in-the-middle payload manipulation, every `protocol.Message` is marshaled to JSON. The sending agent takes the SHA-256 hash of the JSON payload and signs it using its ephemeral ECDSA private key. The receiving agent mathematically verifies the ECDSA signature against the sender's public key (extracted from the validated mTLS peer certificate state) before passing the payload to the application layer.

### 6. Decentralized Gossip Sync (Epidemic Routing)
While the Swarm Registry acts as an initial bootstrap node, the swarm itself is decentralized.
*   Agents maintain an internal `PeerState` routing table.
*   Every 10 seconds, a background Goroutine randomly selects a known peer and dials them via mTLS, passing a `TypeGossip` message containing its routing table.
*   **Result**: Even if the Swarm Registry is destroyed or disconnected, agents can dynamically discover new nodes and continue delegating tasks organically.

### 7. Distributed Observability (OpenTelemetry)
In a highly asynchronous swarm, tracing an agent's reasoning chain across multiple physical binaries is impossible with standard logging.
*   We integrated `go.opentelemetry.io/otel`.
*   When an agent creates a task, it generates a parent `TraceID` and `SpanID`.
*   These are injected directly into the secure `protocol.Message` wire format.
*   The receiving agent extracts the context and links its own internal execution span (`ExecuteTaskOnPeer`) directly to the parent trace, yielding flawless, cross-process execution graphs of the swarm's "thought process."

---

## 🚀 Quick Start (Docker - Recommended)

The entire Zero-Trust Swarm (Identity Provider, Registry, 1 Agent, and the Python Sidecar Proxy) can be launched instantly via Docker:

```bash
docker-compose up --build
```
*The Sidecar proxy will be bound to `localhost:8080`. You can immediately run the Python Agent demo!*

## 🛠️ Manual Installation (Go 1.22+)

1. Start the Identity Provider Daemon:
```bash
./idp-daemon.exe
```

2. Start the Swarm Registry (Bootstrap Node):
```bash
./registry.exe
```

3. Launch an Agent:
```bash
./agent.exe --id "spiffe://swarm.local/agent/math-solver"
```

### Common Errors
*   **Cause**: An agent attempted to query the registry for a specific capability (e.g., `math-solver`), but no agent registered with that capability within the heartbeat window (20 seconds).
*   **Fix**: Ensure your capable agent is actively running and successfully heartbeat-pinging the registry.

---

## 🤝 Future Horizons & Contributing

This project is open-source and built for the future of multi-agent orchestration. We are actively looking for contributions in the following areas:

*   **Post-Quantum Cryptography (PQC)**: Upgrading the mTLS handshakes and ECDSA signatures to lattice-based cryptography (Kyber/Dilithium).
*   **NAT Traversal & QUIC**: Moving from raw TCP to QUIC/WebRTC for seamless global peer discovery without central relays.
*   **Zero-Knowledge Attestation**: Allowing an agent to prove its capability to the registry using ZKPs without revealing its exact SPIFFE ID.

If you want to build the future of secure autonomous AI, check out [CONTRIBUTING.md](CONTRIBUTING.md) and open a Pull Request!

## 🛡️ License

MIT License. See `LICENSE` for more information.

<div align="center">
  <i>Built for the future of multi-agent orchestration.</i>
</div>
