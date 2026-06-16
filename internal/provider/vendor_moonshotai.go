package provider

func init() {
	RegisterVendorAdapter(simpleVendorAdapter{name: "moonshotai", domains: []string{"api.moonshot.ai"}})
}
