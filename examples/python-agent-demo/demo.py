import os
from openai import OpenAI

# 1. Initialize the OpenAI Client
# But instead of pointing to OpenAI's servers, we point it to our Local Go Sidecar Proxy!
# The Go Sidecar handles the complex mTLS, ECDSA signing, and DPI-evasion natively.
client = OpenAI(
    base_url="http://127.0.0.1:8080/v1",
    api_key="sk-no-key-needed" # The Zero-Trust Swarm authenticates via mTLS identity, not API keys!
)

print("🤖 [Python Agent] Initializing...")
print("🤖 [Python Agent] Delegating task to the Zero-Trust Swarm...\n")

try:
    # 2. Delegate the task
    # The `model` field maps exactly to the target agent's capability in the Swarm Registry.
    response = client.chat.completions.create(
        model="math-solver", # We request an agent with the 'math-solver' capability
        messages=[
            {"role": "system", "content": "You are a mathematical solver agent."},
            {"role": "user", "content": "Evaluate 100 * (50 + 2)"}
        ],
        max_tokens=150 # Enforce a strict token/cost budget over the Swarm!
    )

    # 3. Receive the mathematically secure response
    swarm_response = response.choices[0].message.content
    print("\n✅ [Python Agent] Task executed successfully!")
    print(f"✅ [Python Agent] Swarm Response: {swarm_response}")

except Exception as e:
    print(f"\n❌ [Python Agent] Failed to delegate task: {e}")
    print("Hint: Ensure the `run-demo.ps1` script is running in the background so the IdP, Registry, Agent, and Sidecar are active!")
