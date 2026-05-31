package policy

import (
	"testing"

	"github.com/hamza-imran/zero-trust-swarm/pkg/protocol"
)

func TestEngine_Evaluate(t *testing.T) {
	engine := NewEngine([]Rule{
		{
			Effect:     Allow,
			Principals: []string{"spiffe://swarm.local/agent/math-*", "spiffe://swarm.local/registry"},
			Actions:    []protocol.MessageType{protocol.TypeTask, protocol.TypePing},
		},
		{
			Effect:     Deny,
			Principals: []string{"spiffe://swarm.local/agent/math-bad"},
			Actions:    []protocol.MessageType{protocol.TypeTask},
		},
	})

	tests := []struct {
		name      string
		principal string
		action    protocol.MessageType
		want      bool
	}{
		{
			name:      "Allowed agent and action",
			principal: "spiffe://swarm.local/agent/math-1",
			action:    protocol.TypeTask,
			want:      true,
		},
		{
			name:      "Allowed registry and action",
			principal: "spiffe://swarm.local/registry",
			action:    protocol.TypePing,
			want:      true,
		},
		{
			name:      "Denied by specific deny rule",
			principal: "spiffe://swarm.local/agent/math-bad",
			action:    protocol.TypeTask,
			want:      false,
		},
		{
			name:      "Unknown principal",
			principal: "spiffe://swarm.local/agent/hacker",
			action:    protocol.TypeTask,
			want:      false,
		},
		{
			name:      "Allowed principal but unknown action",
			principal: "spiffe://swarm.local/agent/math-1",
			action:    protocol.TypeResult,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := engine.Evaluate(tt.principal, tt.action); got != tt.want {
				t.Errorf("Engine.Evaluate() = %v, want %v", got, tt.want)
			}
		})
	}
}
