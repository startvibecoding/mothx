package provider

func init() {
	RegisterVendorAdapter(simpleVendorAdapter{name: "huggingface", domains: []string{"router.huggingface.co"}})
}
