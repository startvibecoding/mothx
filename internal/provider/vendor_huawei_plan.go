package provider

func init() {
	RegisterVendorAdapter(simpleVendorAdapter{
		name:    "huawei-plan",
		domains: []string{"api.modelarts-maas.com/plan/"},
	})
}
