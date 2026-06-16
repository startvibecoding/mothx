package provider

func init() {
	RegisterVendorAdapter(simpleVendorAdapter{name: "cerebras", domains: []string{"api.cerebras.ai"}})
}
