// Package policy provides a declarative Zero-Trust RBAC/ABAC authorization engine.
// It enforces default-deny access control on all agent communications.
package policy

import (
	"strings"

	"github.com/hamza-imran/zero-trust-swarm/pkg/protocol"
)

// Effect determines whether a rule allows or denies an action.
type Effect string

const (
	Allow Effect = "ALLOW"
	Deny  Effect = "DENY"
)

// Rule defines an authorization policy.
type Rule struct {
	Effect     Effect
	Principals []string               // List of SPIFFE IDs or wildcards (e.g., "spiffe://swarm.local/agent/*")
	Actions    []protocol.MessageType // Allowed message types (e.g., protocol.TypeTask)
}

// Engine evaluates incoming requests against a set of rules.
type Engine struct {
	Rules []Rule
}

// NewEngine initializes a new policy engine with a set of rules.
func NewEngine(rules []Rule) *Engine {
	return &Engine{Rules: rules}
}

// Evaluate checks if the given principal is allowed to perform the action.
// It operates on a default-deny architecture.
func (e *Engine) Evaluate(principal string, action protocol.MessageType) bool {
	// If no rules exist, default deny
	if len(e.Rules) == 0 {
		return false
	}

	allowed := false

	for _, rule := range e.Rules {
		if !e.matchesPrincipal(principal, rule.Principals) {
			continue
		}

		if !e.matchesAction(action, rule.Actions) {
			continue
		}

		// Explicit Deny overrides Allow
		if rule.Effect == Deny {
			return false
		}

		if rule.Effect == Allow {
			allowed = true
		}
	}

	return allowed
}

// matchesPrincipal checks if the requested principal matches any of the allowed principals, supporting simple wildcards.
func (e *Engine) matchesPrincipal(principal string, allowedPrincipals []string) bool {
	for _, p := range allowedPrincipals {
		if p == "*" || p == principal {
			return true
		}
		// Basic suffix wildcard support (e.g., "spiffe://swarm.local/*")
		if strings.HasSuffix(p, "*") {
			prefix := strings.TrimSuffix(p, "*")
			if strings.HasPrefix(principal, prefix) {
				return true
			}
		}
	}
	return false
}

// matchesAction checks if the requested action matches the allowed actions.
func (e *Engine) matchesAction(action protocol.MessageType, allowedActions []protocol.MessageType) bool {
	for _, a := range allowedActions {
		if a == action {
			return true
		}
	}
	return false
}
