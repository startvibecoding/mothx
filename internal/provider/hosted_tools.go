package provider

const (
	HostedToolWebSearch                  = "web_search"
	HostedToolWebSearchAnthropicMessages = "web_search_20250305"
)

// HostedWebSearchToolType maps a hosted web_search tool to the provider-specific wire type.
// It is provider-neutral: the mapping depends on the tool's API family, not the vendor name.
func HostedWebSearchToolType(providerType, name string) string {
	if name != HostedToolWebSearch {
		return ""
	}
	switch providerType {
	case "responses", "openai-responses":
		return HostedToolWebSearch
	case "messages", "anthropic-messages":
		return HostedToolWebSearchAnthropicMessages
	default:
		return ""
	}
}
