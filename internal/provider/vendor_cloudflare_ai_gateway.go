package provider

func init() {
	RegisterVendorAdapter(simpleVendorAdapter{
		name:       "cloudflare-ai-gateway",
		domains:    []string{"gateway.ai.cloudflare.com"},
		defaultAPI: "openai-chat",
	})
}
