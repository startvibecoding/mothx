package provider

func init() {
	RegisterVendorAdapter(simpleVendorAdapter{name: "vercel-ai-gateway", domains: []string{"ai-gateway.vercel.sh"}})
}
