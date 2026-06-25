package provider

func init() {
	RegisterVendorAdapter(simpleVendorAdapter{
		name:       "volcengine",
		domains:    []string{"ark.cn-beijing.volces.com"},
		defaultAPI: "openai-chat",
	})
}
