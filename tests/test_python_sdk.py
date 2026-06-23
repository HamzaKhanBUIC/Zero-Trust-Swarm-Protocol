import unittest
from unittest.mock import patch, MagicMock
from swarm_mtls.client import SwarmAgent

class TestSwarmAgent(unittest.TestCase):
    def setUp(self):
        self.agent = SwarmAgent(name="test-agent", host="127.0.0.1", port=5000)

    def test_capability_registration(self):
        @self.agent.capability
        def mock_capability(payload: str) -> str:
            return "success"
            
        self.assertIn("mock_capability", self.agent.capabilities)
        self.assertEqual(self.agent.capabilities["mock_capability"]("test"), "success")

if __name__ == '__main__':
    unittest.main()
