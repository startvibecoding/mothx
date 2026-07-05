package provider

func init() {
	RegisterVendorAdapter(simpleVendorAdapter{
		name:    "jd-plan",
		domains: []string{"agentrs.jd.com"},
	})
}
