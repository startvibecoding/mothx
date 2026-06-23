package provider

func init() {
	RegisterVendorAdapter(simpleVendorAdapter{
		name:       "mistral",
		domains:    []string{"api.mistral.ai"},
		defaultAPI: "openai-chat",
	})
}
