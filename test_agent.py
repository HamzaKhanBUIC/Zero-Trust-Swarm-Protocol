import sys
import os

# Ensure the SDK is on the path
sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__), 'sdk', 'python')))

from swarm_mtls.client import SwarmAgent

# Initialize the agent server on port 5000
print("Initializing Python Swarm Agent...")
agent = SwarmAgent(port=5000)

@agent.capability
def analyze_data(payload: str) -> str:
    print(f"\n[AI AGENT] Received incoming task payload: {payload}")
    return f"AI Analysis Result: '{payload}' is mathematically secure."

if __name__ == "__main__":
    print("Agent is ready. Waiting for tasks from the Sidecar...")
    agent.serve()
