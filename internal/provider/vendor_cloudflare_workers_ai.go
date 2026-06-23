package provider

func init() {
	RegisterVendorAdapter(simpleVendorAdapter{
		name:       "cloudflare-workers-ai",
		domains:    []string{"api.cloudflare.com"},
		defaultAPI: "openai-chat",
	})
}
