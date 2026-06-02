package provider

func init() {
	RegisterVendorAdapter(simpleVendorAdapter{
		name:       "google-gemini",
		domains:    []string{"generativelanguage.googleapis.com"},
		defaultAPI: "google-gemini",
	})
}
