# Contributing to Zero-Trust Swarm Protocol

Welcome to the **Zero-Trust Swarm Protocol** (SwarmTLS). We are building the foundational transport and security layer for the future of autonomous, multi-agent artificial intelligence.

Because this project sits at the critical intersection of Network Security, Cryptography, and AI, we have high standards for contributions.

## 🎯 Current Engineering Priorities
As outlined in the README, our primary focus areas are:
1. **Post-Quantum Cryptography (PQC)**: We are looking for cryptographers to help implement Kyber/Dilithium for mTLS and payload signing.
2. **QUIC / WebRTC Transport**: Moving away from raw TCP to support seamless NAT traversal for agents running on edge devices (laptops, phones) behind residential routers.
3. **Zero-Knowledge Proofs (ZKPs)**: Implementing capability attestation without revealing the raw SPIFFE ID.

## 🛠️ Development Guidelines

### 1. Security First
- This is a security protocol. Any changes to `transport.SwarmTLS`, the ECDSA payload signer, or the SPIFFE IdP logic must be accompanied by comprehensive tests and a mathematical justification if cryptography is altered.
- **Never commit private keys or certificates.** All tests must use dynamically generated keys.

### 2. Code Style (Go)
- We strictly adhere to standard Go formatting. Run `go fmt` and `go vet` before submitting a PR.
- Use explicit error handling. Avoid `panic()` in agent routines unless the error is fundamentally unrecoverable at boot.

### 3. OpenTelemetry Integration
- If you add a new logical component or task type to the swarm, you **must** instrument it with an OpenTelemetry Span. We rely on distributed tracing to debug the swarm's behavior across multiple nodes.

## 🚀 Submitting a Pull Request
1. Fork the repository.
2. Create a feature branch (`git checkout -b feature/quic-transport`).
3. Write your code and tests.
4. Open a Pull Request against `main`. Provide a clear description of the problem solved, architecture changed, and attach any benchmark results if performance was affected.

---
*By contributing, you agree that your code will be released under the MIT License.*
