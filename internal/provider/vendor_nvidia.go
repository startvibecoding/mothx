package provider

func init() {
	RegisterVendorAdapter(simpleVendorAdapter{name: "nvidia", domains: []string{"integrate.api.nvidia.com"}})
}
