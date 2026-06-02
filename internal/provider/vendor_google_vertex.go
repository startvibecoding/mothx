package provider

func init() {
	RegisterVendorAdapter(simpleVendorAdapter{
		name:       "google-vertex",
		domains:    []string{"aiplatform.googleapis.com"},
		defaultAPI: "google-vertex",
	})
}
