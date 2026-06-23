package provider

func init() {
	// Amazon Bedrock uses OpenAI-compatible cross-region inference endpoints.
	// Users configure the base URL for their region, e.g.:
	//   https://bedrock-runtime.us-east-1.amazonaws.com/openai/v1
	// AWS SigV4 signing is handled by the user's proxy or API key configuration.
	RegisterVendorAdapter(simpleVendorAdapter{
		name:       "amazon-bedrock",
		domains:    []string{"bedrock-runtime", "bedrock-api"},
		defaultAPI: "openai-chat",
	})
}
