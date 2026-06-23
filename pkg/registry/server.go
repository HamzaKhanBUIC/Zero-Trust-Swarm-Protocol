package registry

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"sync"
	"time"
)

//go:embed dist/*
var dashboardFS embed.FS

// Server handles the Swarm Visualizer dashboard HTTP and SSE endpoints.
type Server struct {
	Registry *SwarmRegistry
	clients  map[chan string]bool
	mu       sync.RWMutex
}

// NewServer creates a new HTTP server for the registry dashboard.
func NewServer(reg *SwarmRegistry) *Server {
	return &Server{
		Registry: reg,
		clients:  make(map[chan string]bool),
	}
}

// BroadcastEvent sends an event to all connected SSE clients.
func (s *Server) BroadcastEvent(eventType, agentID, message string) {
	ev := map[string]string{
		"time":     time.Now().Format(http.TimeFormat),
		"type":     eventType,
		"agent_id": agentID,
		"message":  message,
	}
	bytes, _ := json.Marshal(ev)

	s.mu.RLock()
	defer s.mu.RUnlock()
	for client := range s.clients {
		select {
		case client <- string(bytes):
		default:
			// Non-blocking send
		}
	}
}

// Start listens on the given address.
func (s *Server) Start(addr string) {
	mux := http.NewServeMux()

	// Serve the embedded static Vite app
	subFS, err := fs.Sub(dashboardFS, "dist")
	if err != nil {
		log.Printf("Warning: Failed to load dashboard dist directory: %v", err)
	} else {
		mux.Handle("/", http.FileServer(http.FS(subFS)))
	}

	// API Endpoint for Agent Data
	mux.HandleFunc("/api/agents", func(w http.ResponseWriter, r *http.Request) {
		agents := s.Registry.Query("")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(QueryResponse{Agents: agents})
	})

	// Server-Sent Events Endpoint
	mux.HandleFunc("/api/events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		clientChan := make(chan string, 10)
		s.mu.Lock()
		s.clients[clientChan] = true
		s.mu.Unlock()

		defer func() {
			s.mu.Lock()
			delete(s.clients, clientChan)
			s.mu.Unlock()
			close(clientChan)
		}()

		// Send initial connected event
		fmt.Fprintf(w, "data: %s\n\n", `{"type":"info","agent_id":"system","message":"Connected to event stream"}`)
		flusher.Flush()

		for {
			select {
			case <-r.Context().Done():
				return
			case msg := <-clientChan:
				fmt.Fprintf(w, "data: %s\n\n", msg)
				flusher.Flush()
			}
		}
	})

	log.Printf("🌐 Swarm Visualizer Dashboard running at http://%s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Printf("Dashboard server stopped: %v", err)
	}
}
