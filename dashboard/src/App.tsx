import { useState, useEffect } from 'react';
import ReactFlow, {
  Background,
  Controls,
  useNodesState,
  useEdgesState,
  MarkerType,
  type Edge,
  type Node,
} from 'reactflow';
import 'reactflow/dist/style.css';
import { Activity, ShieldCheck, Server, Cpu, Key } from 'lucide-react';
import { SwarmAPI, type SwarmEvent } from './api';


const initialNodes: Node[] = [
  {
    id: 'swarm-registry',
    type: 'default',
    position: { x: 250, y: 50 },
    data: {
      label: (
        <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '8px' }}>
          <Server size={32} color="#8b5cf6" />
          <div style={{ fontWeight: 'bold' }}>Swarm Registry</div>
          <div style={{ fontSize: '10px', color: '#94a3b8' }}>Bootstrap Node (mTLS Enforced)</div>
        </div>
      ),
    },
    className: 'registry',
  },
];

function App() {
  const [nodes, setNodes, onNodesChange] = useNodesState(initialNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);
  const [events, setEvents] = useState<SwarmEvent[]>([]);
  const [agentCount, setAgentCount] = useState(0);

  // Poll registry for active nodes
  useEffect(() => {
    const fetchNodes = async () => {
      try {
        const agents = await SwarmAPI.fetchAgents();
        setAgentCount(agents.length);
        
        const newNodes: Node[] = [...initialNodes];
        const newEdges: Edge[] = [];

        agents.forEach((agent, index) => {
          // Layout in a semi-circle around the registry
          const angle = (index / Math.max(agents.length, 1)) * Math.PI + Math.PI;
          const radius = 250;
          const x = 250 + Math.cos(angle) * radius;
          const y = 300 + Math.sin(angle) * radius;

          newNodes.push({
            id: agent.agent_id,
            position: { x, y },
            data: {
              label: (
                <div>
                  <div className="node-header">
                    <Cpu size={20} color="#38bdf8" />
                    Agent Node
                  </div>
                  <div className="node-id">{agent.agent_id}</div>
                  <div className="node-caps">
                    {agent.capabilities.map(c => (
                      <span key={c} className="cap-badge">{c}</span>
                    ))}
                  </div>
                </div>
              )
            }
          });

          newEdges.push({
            id: `e-reg-${agent.agent_id}`,
            source: agent.agent_id,
            target: 'swarm-registry',
            animated: true,
            style: { stroke: 'rgba(56, 189, 248, 0.4)', strokeWidth: 2 },
            markerEnd: { type: MarkerType.ArrowClosed, color: 'rgba(56, 189, 248, 0.4)' }
          });
        });

        setNodes(newNodes);
        setEdges(newEdges);
      } catch (err) {
        console.error("Failed to fetch nodes:", err);
      }
    };

    fetchNodes();
    const interval = setInterval(fetchNodes, 5000);
    return () => clearInterval(interval);
  }, [setNodes, setEdges]);

  // Connect to SSE for live event stream
  useEffect(() => {
    const sse = SwarmAPI.subscribeToEvents((ev) => {
      setEvents(prev => [ev, ...prev].slice(0, 50)); // Keep last 50 events
    });
    return () => sse.close();
  }, []);

  return (
    <div style={{ width: '100vw', height: '100vh', position: 'relative' }}>
      
      {/* Top Header */}
      <header className="dashboard-header">
        <div className="dashboard-title">
          <ShieldCheck size={28} color="#38bdf8" />
          Zero-Trust Swarm Visualizer
        </div>
        <div className="dashboard-stats">
          <div className="stat-badge">
            <div className="pulse-dot"></div>
            <span>{agentCount} Active Agents</span>
          </div>
          <div className="stat-badge">
            <Key size={16} color="#8b5cf6" />
            <span>mTLS 1.3 Active</span>
          </div>
        </div>
      </header>

      {/* React Flow Canvas */}
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        fitView
        attributionPosition="bottom-right"
      >
        <Background color="#1e293b" gap={20} size={1} />
        <Controls />
      </ReactFlow>

      {/* Live Event Log Panel */}
      <div className="event-log-panel">
        <div className="event-log-header">
          <Activity size={18} color="#22c55e" />
          Swarm Event Stream
        </div>
        <div className="event-log-list">
          {events.length === 0 ? (
            <div style={{ textAlign: 'center', color: 'var(--text-muted)', marginTop: '20px' }}>
              Waiting for network activity...
            </div>
          ) : (
            events.map((ev, i) => (
              <div key={i} className={`event-item type-${ev.type.toLowerCase()}`}>
                <div className="event-time">{new Date(ev.time).toLocaleTimeString()}</div>
                <div style={{ fontWeight: 600 }}>{ev.type.toUpperCase()}</div>
                <div style={{ fontFamily: 'monospace', fontSize: '0.75rem', color: 'var(--text-muted)', wordBreak: 'break-all' }}>
                  {ev.agent_id}
                </div>
                <div style={{ marginTop: '4px' }}>{ev.message}</div>
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  );
}

export default App;
