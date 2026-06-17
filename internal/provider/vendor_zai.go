package provider

func init() {
	RegisterVendorAdapter(simpleVendorAdapter{
		name:           "zai",
		domains:        []string{"api.z.ai", "open.bigmodel.cn"},
		thinkingFormat: "zai",
	})
}
