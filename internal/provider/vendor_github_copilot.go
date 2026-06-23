package provider

func init() {
	RegisterVendorAdapter(simpleVendorAdapter{
		name:       "github-copilot",
		domains:    []string{"api.individual.githubcopilot.com", "api.githubcopilot.com"},
		defaultAPI: "openai-chat",
	})
}
