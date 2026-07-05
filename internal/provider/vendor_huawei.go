package provider

func init() {
	RegisterVendorAdapter(simpleVendorAdapter{
		name:    "huawei",
		domains: []string{"api.modelarts-maas.com"},
	})
}
