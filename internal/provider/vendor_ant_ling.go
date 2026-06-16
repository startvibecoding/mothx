package provider

func init() {
	RegisterVendorAdapter(simpleVendorAdapter{name: "ant-ling", domains: []string{"api.ant-ling.com"}})
}
