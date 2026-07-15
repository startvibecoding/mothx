package provider

func init() {
	RegisterVendorAdapter(simpleVendorAdapter{
		name:       "codeok",
		domains:    []string{"codeok.cc"},
		defaultAPI: "openai-responses",
	})
}
