import requests
import json
from openai import OpenAI
from typing import Optional, Dict, Any

class SwarmClient:
    """
    Python SDK for the Zero-Trust Swarm Protocol.
    Interacts with the local Go Sidecar proxy.
    """

    def __init__(self, sidecar_url: str = "http://127.0.0.1:8080"):
        self.sidecar_url = sidecar_url
        # Seamlessly wrap the OpenAI client
        self.openai = OpenAI(
            base_url=f"{self.sidecar_url}/v1",
            api_key="sk-no-key-needed"
        )

    def ask(self, capability: str, prompt: str, max_tokens: int = 150) -> str:
        """
        Synchronously delegate a task to the swarm using the OpenAI ChatCompletions schema.
        This blocks until the peer responds.
        """
        response = self.openai.chat.completions.create(
            model=capability,
            messages=[{"role": "user", "content": prompt}],
            max_tokens=max_tokens
        )
        return response.choices[0].message.content

    def delegate_async(self, target_id: str, payload: str) -> Dict[str, Any]:
        """
        Asynchronously delegate a generic payload to a specific agent.
        The sidecar will persistently queue this message and deliver it via mTLS in the background.
        """
        res = requests.post(
            f"{self.sidecar_url}/v1/tasks/async",
            json={"target_id": target_id, "payload": payload}
        )
        res.raise_for_status()
        return res.json()
