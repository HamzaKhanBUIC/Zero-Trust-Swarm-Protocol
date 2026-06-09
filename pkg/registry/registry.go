// Package registry provides the service registry for zero-trust swarm discovery.
// It tracks active agent endpoints, capabilities, and health, verified via mTLS.
package registry

import (
	"sync"
	"time"
)

// AgentRecord holds registration details for a single agent.
type AgentRecord struct {
	AgentID      string    `json:"agent_id"`
	Address      string    `json:"address"`
	Capabilities []string  `json:"capabilities"`
	LastSeen     time.Time `json:"last_seen"`
}

// SwarmRegistry is a thread-safe registry of active agents.
type SwarmRegistry struct {
	mu     sync.RWMutex
	agents map[string]*AgentRecord
}

// NewSwarmRegistry creates a new empty registry.
type QueryResponse struct {
	Agents []*AgentRecord `json:"agents"`
}

func NewSwarmRegistry() *SwarmRegistry {
	r := &SwarmRegistry{
		agents: make(map[string]*AgentRecord),
	}
	go r.cleanupRoutine()
	return r
}

// cleanupRoutine periodically sweeps the registry and permanently deletes stale agents
// from memory to prevent Out-Of-Memory (OOM) crashes during long uptimes.
func (r *SwarmRegistry) cleanupRoutine() {
	for {
		time.Sleep(1 * time.Minute)
		r.mu.Lock()
		now := time.Now()
		for id, rec := range r.agents {
			if now.Sub(rec.LastSeen) > 1*time.Minute {
				delete(r.agents, id)
			}
		}
		r.mu.Unlock()
	}
}

// Register registers or updates an agent in the registry.
func (r *SwarmRegistry) Register(agentID, addr string, caps []string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.agents[agentID] = &AgentRecord{
		AgentID:      agentID,
		Address:      addr,
		Capabilities: caps,
		LastSeen:     time.Now(),
	}
}

// Deregister removes an agent from the registry.
func (r *SwarmRegistry) Deregister(agentID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.agents, agentID)
}

// Query retrieves active agents matching a specific capability.
// If query is empty, returns all active agents.
func (r *SwarmRegistry) Query(capability string) []*AgentRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []*AgentRecord
	now := time.Now()

	for _, rec := range r.agents {
		// Prune inactive agents (older than 1 minute)
		if now.Sub(rec.LastSeen) > 1*time.Minute {
			continue
		}

		if capability == "" {
			results = append(results, rec)
			continue
		}

		for _, cap := range rec.Capabilities {
			if cap == capability {
				results = append(results, rec)
				break
			}
		}
	}

	return results
}
