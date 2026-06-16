package provider

func init() {
	RegisterVendorAdapter(simpleVendorAdapter{name: "xai", domains: []string{"api.x.ai"}})
}
