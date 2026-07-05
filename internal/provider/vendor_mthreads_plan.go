package provider

func init() {
	RegisterVendorAdapter(simpleVendorAdapter{
		name:    "mthreads-plan",
		domains: []string{"coding-plan-endpoint.kuaecloud.net"},
	})
}
