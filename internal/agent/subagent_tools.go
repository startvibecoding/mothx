package agent

import "github.com/startvibecoding/vibecoding/internal/tools"

// RegisterSubAgentTools registers the built-in sub-agent tools when multi-agent
// mode is enabled. It is safe to call more than once; Registry.Register replaces
// existing tools without duplicating their order.
func RegisterSubAgentTools(registry *tools.Registry, manager *AgentManager) {
	if registry == nil || manager == nil {
		return
	}
	registry.Register(NewSubAgentSpawnTool(manager))
	registry.Register(NewSubAgentStatusTool(manager))
	registry.Register(NewSubAgentSendTool(manager))
	registry.Register(NewSubAgentDestroyTool(manager))
}
