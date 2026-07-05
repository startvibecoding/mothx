package provider

func init() {
	RegisterVendorAdapter(simpleVendorAdapter{
		name:    "ctyun-plan",
		domains: []string{"wishub-x6.ctyun.cn"},
	})
}
