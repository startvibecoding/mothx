package provider

func init() {
	RegisterVendorAdapter(simpleVendorAdapter{name: "opencode", domains: []string{"opencode.ai"}})
}
