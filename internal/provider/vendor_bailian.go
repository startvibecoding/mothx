package provider

func init() {
	RegisterVendorAdapter(simpleVendorAdapter{
		name:    "bailian",
		domains: []string{"dashscope.aliyuncs.com", "token-plan.cn-beijing.maas.aliyuncs.com", "coding.dashscope.aliyuncs.com"},
	})
}
