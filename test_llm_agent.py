import sys
import os

# Add the local sdk path so we can import swarm_mtls without pip install
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), 'sdk', 'python')))

from swarm_mtls.client import OllamaAgent

def main():
    print("Initializing Ollama Swarm Agent...")
    print("Ensure you have Ollama running locally (http://localhost:11434) with 'phi3' pulled.")
    
    # Create the agent natively configured for Phi-3 (or fallback to llama3 if installed)
    agent = OllamaAgent(
        name="code-reviewer", 
        host="127.0.0.1", 
        port=5000, 
        model_name="phi3"
    )
    
    # Start the secure server to listen for Swarm events
    agent.serve()

if __name__ == "__main__":
    main()
