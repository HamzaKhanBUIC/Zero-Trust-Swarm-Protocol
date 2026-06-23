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


from http.server import BaseHTTPRequestHandler, HTTPServer
import threading

class SwarmAgent:
    """
    Python server that receives incoming tasks from the local Sidecar proxy.
    """
    def __init__(self, host="127.0.0.1", port=5000):
        self.host = host
        self.port = port
        self._handler_func = None

    def capability(self, func):
        """
        Decorator to register the function that processes incoming tasks.
        (For simplicity in this 1.0 architecture, we handle a single primary capability function per server).
        """
        self._handler_func = func
        return func

    def serve(self):
        """
        Starts the local HTTP server to listen for requests from the Sidecar.
        """
        agent = self
        
        class WebhookHandler(BaseHTTPRequestHandler):
            def do_POST(self):
                content_length = int(self.headers['Content-Length'])
                post_data = self.rfile.read(content_length)
                try:
                    data = json.loads(post_data.decode('utf-8'))
                    payload = data.get("payload", "")
                    
                    if agent._handler_func:
                        result = agent._handler_func(payload)
                    else:
                        result = f"Received: {payload} (No handler registered)"
                        
                    self.send_response(200)
                    self.send_header('Content-type', 'text/plain')
                    self.end_headers()
                    self.wfile.write(str(result).encode('utf-8'))
                except Exception as e:
                    self.send_response(500)
                    self.send_header('Content-type', 'text/plain')
                    self.end_headers()
                    self.wfile.write(f"Error processing task: {e}".encode('utf-8'))

            def log_message(self, format, *args):
                # Suppress default HTTP logging to keep console clean
                pass

        server_address = (self.host, self.port)
        httpd = HTTPServer(server_address, WebhookHandler)
        print(f"🐍 Python SwarmAgent HTTP listener running on http://{self.host}:{self.port}")
        httpd.serve_forever()
