package idp

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/hamza-imran/zero-trust-swarm/pkg/pki"
)

// MinTokenLen is the shortest bootstrap token the IdP accepts.
const MinTokenLen = 16

// ErrDenied is returned when a request names an identity the IdP may not issue.
var ErrDenied = errors.New("identity request denied")

// Allowlist is the set of identities the IdP may issue. An identity may be
// bound to a bootstrap token; the caller must then present that token.
type Allowlist struct {
	tokens map[string]string // agent ID -> token ("" means no token required)
}

// ParseAllowlist parses a comma-separated list of "name" or "name=token"
// entries, for example "swarm-registry=s3cret...,agent-alpha".
func ParseAllowlist(s string) (*Allowlist, error) {
	a := &Allowlist{tokens: make(map[string]string)}
	for _, entry := range strings.Split(s, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		id, token, hasToken := strings.Cut(entry, "=")
		if !pki.ValidAgentID(id) {
			return nil, fmt.Errorf("invalid agent ID %q in allowlist", id)
		}
		if hasToken && len(token) < MinTokenLen {
			return nil, fmt.Errorf("token for %q is shorter than %d characters", id, MinTokenLen)
		}
		if _, dup := a.tokens[id]; dup {
			return nil, fmt.Errorf("agent ID %q listed twice", id)
		}
		a.tokens[id] = token
	}
	if len(a.tokens) == 0 {
		return nil, errors.New("allowlist is empty")
	}
	return a, nil
}

// Authorize returns nil only if req names an allowed identity and carries
// that identity's token, when one is configured.
func (a *Allowlist) Authorize(req IdentityRequest) error {
	want, ok := a.tokens[req.AgentID]
	if !ok {
		return fmt.Errorf("%w: %q is not on the allowlist", ErrDenied, req.AgentID)
	}
	if want != "" && subtle.ConstantTimeCompare([]byte(req.Token), []byte(want)) != 1 {
		return fmt.Errorf("%w: bad token for %q", ErrDenied, req.AgentID)
	}
	return nil
}

// RequiresTokens reports whether every identity on the list needs a token.
func (a *Allowlist) RequiresTokens() bool {
	for _, token := range a.tokens {
		if token == "" {
			return false
		}
	}
	return true
}

// IsLoopback reports whether the listen address addr only accepts
// connections from this host.
func IsLoopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
