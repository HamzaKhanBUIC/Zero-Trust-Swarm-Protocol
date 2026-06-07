# Python Agent Swarm Demo

This directory demonstrates how any Python-based AI framework (like standard Python, LangChain, CrewAI, or AutoGen) can seamlessly integrate with the Go-based **Zero-Trust Swarm Protocol**.

We achieve this using the **Localhost Sidecar Proxy**.

## The Magic
Instead of pointing the `openai` client to OpenAI's cloud servers, we point the `base_url` to our local Go sidecar proxy. The Go sidecar instantly wraps the JSON payload in a deep-packet-inspection evading mTLS tunnel, signs it with an ephemeral ECDSA identity key, and routes it to the correct agent in the swarm.

Zero Python cryptography required. 100% mathematically secure networking.

## Running the Demo

1. Open a terminal in the root of the repository and launch the Swarm and the Sidecar using the PowerShell demo script:
   ```powershell
   .\test-llm.ps1
   ```
   *(This starts the IdP Daemon, the Registry, Agent Alpha, and the Sidecar in the background).*

2. In a new terminal, navigate to this folder and install the standard `openai` pip package:
   ```bash
   pip install -r requirements.txt
   ```

3. Run the Python agent!
   ```bash
   python demo.py
   ```

You will immediately see your Python code delegate a task natively over the mTLS swarm!
