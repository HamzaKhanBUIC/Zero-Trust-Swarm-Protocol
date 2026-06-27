export interface SwarmEvent {
  time: string;
  type: string;
  agent_id: string;
  message: string;
}

export interface AgentRecord {
  agent_id: string;
  address: string;
  capabilities: string[];
  last_seen: string;
}

export class SwarmAPI {
  /**
   * Fetches active agent nodes from the registry.
   */
  static async fetchAgents(): Promise<AgentRecord[]> {
    const res = await fetch('/api/agents');
    if (!res.ok) {
      throw new Error(`Failed to fetch agents: ${res.statusText}`);
    }
    const data = await res.json();
    return data.agents || [];
  }

  /**
   * Establishes a Server-Sent Events (SSE) connection to listen for live swarm events.
   * Returns the EventSource instance, and takes a callback for onMessage events.
   */
  static subscribeToEvents(onMessage: (event: SwarmEvent) => void, onError?: (err: Event) => void): EventSource {
    const sse = new EventSource('/api/events');
    
    sse.onmessage = (e) => {
      try {
        const ev: SwarmEvent = JSON.parse(e.data);
        onMessage(ev);
      } catch (err) {
        console.error("Failed to parse SSE event data:", err);
      }
    };

    if (onError) {
      sse.onerror = (e) => {
        onError(e);
      };
    }

    return sse;
  }
}
