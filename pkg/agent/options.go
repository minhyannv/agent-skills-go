package agent

import loggerpkg "github.com/minhyannv/agent-skills-go/pkg/logger"

// AgentOption configures optional runtime dependencies for AgentLoop.
type AgentOption func(*agentDeps)

type agentDeps struct {
	logger loggerpkg.Logger
}

// WithLogger injects a logger dependency.
func WithLogger(l loggerpkg.Logger) AgentOption {
	return func(d *agentDeps) {
		d.logger = l
	}
}
