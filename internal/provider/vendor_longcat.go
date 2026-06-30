package provider

func init() {
	RegisterVendorAdapter(simpleVendorAdapter{
		name:    "longcat",
		domains: []string{"api.longcat.chat"},
	})
}
