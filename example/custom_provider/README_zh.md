# 自定义 Provider 示例

本示例展示了外部 Go 开发者如何实现自己的大语言模型（LLM）后端，并将其无缝地接入到 VibeCoding 的高级 Agent 框架中。

## 工作原理

VibeCoding 提供了一个公共的 `Provider` 接口：
```go
type Provider interface {
	Chat(ctx context.Context, params ChatParams) <-chan StreamEvent
	Name() string
	Models() []ModelInfo
	GetModel(id string) *ModelInfo
}
```

在本示例中，我们：
1. 实现了嵌套 `agent.BaseProvider` 的 `CustomProvider` 结构体。
2. 定义了一个自定义 Mock 模型（`mock-model`）。
3. 实现了 `Chat` 方法以输出 Mock 流式回复。
4. 结合关键词触发：
   - **文本流式输出**：模拟标准的一字一句（delta-by-delta）响应流。
   - **工具执行请求**：通过发送带 `ToolCallBlock` 的 `StreamToolCall` 事件，模拟 LLM 决策去读取文件（`README.md`）。
5. 用此自定义 Provider 运行 VibeCoding 的 Agent 循环。

请注意，VibeCoding 内置的工具注册表（Tool Registry）是如何自动拦截 LLM 的 `StreamToolCall` 事件、安全地在工作区执行实际的 `read` 工具，并将执行结果反馈给 Assistant 的！

## 如何运行

1. 确保你在项目的根目录下。
2. 运行该示例：
   ```bash
   go run example/custom_provider/main.go
   ```
