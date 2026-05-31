// Package gossip implements a lightweight, pure-Go peer-to-peer discovery protocol over mTLS.
package gossip

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/hamza-imran/zero-trust-swarm/pkg/protocol"
	"github.com/hamza-imran/zero-trust-swarm/pkg/registry"
	"github.com/hamza-imran/zero-trust-swarm/pkg/transport"
)

// GossipEngine manages peer discovery and synchronization.
type GossipEngine struct {
	AgentID   string
	Address   string
	SwarmTLS  *transport.SwarmTLS
	PrivKey   *ecdsa.PrivateKey
	PeerState map[string]registry.AgentRecord
	mu        sync.RWMutex
}

// NewGossipEngine creates a new decentralized gossip engine.
func NewGossipEngine(agentID, addr string, swarmTLS *transport.SwarmTLS, privKey *ecdsa.PrivateKey) *GossipEngine {
	return &GossipEngine{
		AgentID:   agentID,
		Address:   addr,
		SwarmTLS:  swarmTLS,
		PrivKey:   privKey,
		PeerState: make(map[string]registry.AgentRecord),
	}
}

// UpdateLocal updates the agent's own state or explicitly adds a seed peer.
func (g *GossipEngine) UpdateLocal(rec registry.AgentRecord) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.PeerState[rec.AgentID] = rec
}

// MergeState merges an incoming state map with the local state.
func (g *GossipEngine) MergeState(incoming map[string]registry.AgentRecord) {
	g.mu.Lock()
	defer g.mu.Unlock()
	mergedCount := 0
	for id, rec := range incoming {
		if _, exists := g.PeerState[id]; !exists {
			g.PeerState[id] = rec
			mergedCount++
			fmt.Printf("🌐 [GOSSIP DISCOVERY] Discovered new peer: %s at %s\n", id, rec.Address)
		}
	}
}

// Start initiates the periodic gossip loop.
func (g *GossipEngine) Start() {
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for range ticker.C {
			g.gossipRandomPeer()
		}
	}()
}

// gossipRandomPeer selects a random known peer and sends them our state.
func (g *GossipEngine) gossipRandomPeer() {
	g.mu.RLock()
	peers := make([]registry.AgentRecord, 0, len(g.PeerState))
	for id, rec := range g.PeerState {
		// Don't gossip with ourselves
		if id != g.AgentID {
			peers = append(peers, rec)
		}
	}
	stateBytes, _ := json.Marshal(g.PeerState)
	g.mu.RUnlock()

	if len(peers) == 0 {
		return // No peers to gossip with yet
	}

	// Select random peer
	target := peers[rand.Intn(len(peers))]

	// Dial peer over mTLS
	conn, err := g.SwarmTLS.Dial(target.Address)
	if err != nil {
		// log.Printf("⚠️ Gossip failed to dial %s: %v", target.AgentID, err)
		return
	}
	defer conn.Close()

	msg := protocol.NewMessage(protocol.TypeGossip, g.AgentID, target.AgentID, string(stateBytes))
	msg.Sign(g.PrivKey)

	if err := protocol.WriteMessage(conn, msg); err != nil {
		log.Printf("⚠️ Gossip write failed to %s: %v", target.AgentID, err)
	}
}
