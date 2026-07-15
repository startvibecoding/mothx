package provider

func init() {
	RegisterVendorAdapter(simpleVendorAdapter{
		name:       "yescode",
		domains:    []string{"co.yes.vg"},
		defaultAPI: "openai-responses",
	})
}
