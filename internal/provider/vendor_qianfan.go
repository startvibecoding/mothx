package provider

func init() {
	RegisterVendorAdapter(simpleVendorAdapter{
		name:       "qianfan",
		domains:    []string{"qianfan.baidubce.com", "aip.baidubce.com"},
		defaultAPI: "openai-chat",
	})
}
