# Contributing to Zero-Trust Swarm Protocol

First off, thank you for considering contributing to the Zero-Trust Swarm Protocol! We are building the foundational networking layer for the future of autonomous, multi-agent AI systems.

## Getting Started
1. Fork the repository.
2. Clone your fork: `git clone https://github.com/your-username/Zero-Trust-Swarm-Protocol.git`
3. Create your feature branch: `git checkout -b feature/amazing-feature`
4. Start the swarm environment locally via Docker: `docker-compose up --build`
5. Test the Python SDK: `python test_agent.py`

## Areas of Contribution
We are actively looking for elite engineering contributions in:
* **React Frontend**: Building out the `Swarm Visualizer Dashboard` using React Flow and Server-Sent Events.
* **Python SDK LLM Integration**: Adding LangChain, OpenAI, and Gemini support to the native Python agent.
* **Post-Quantum Cryptography**: Upgrading mTLS from ECDSA to lattice-based schemes.
* **NAT Traversal**: Upgrading transport from raw TCP to WebRTC/QUIC for cross-network peer discovery.

## Code Standards
* All Go code must be formatted with `gofmt`.
* Cryptographic functions must rely on the standard `crypto` libraries unless implementing a highly vetted external PQC library.
* Any changes to `pkg/protocol` must be accompanied by updates to `pkg/protocol/protocol_test.go`.

## Submitting a Pull Request
1. Commit your changes: `git commit -m 'feat: Add amazing feature'`
2. Push to the branch: `git push origin feature/amazing-feature`
3. Open a Pull Request against the `main` branch.
4. Ensure the GitHub Actions CI pipeline passes successfully.
