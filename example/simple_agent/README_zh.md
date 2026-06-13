# 简单 Agent 示例

本示例展示了如何使用 VibeCoding 的公共 `agent` 包来构建和运行一个支持流式输出的基础 AI Agent。

## 工作原理

本示例使用了 `agent` 包提供的链式 `Builder` API：
1. 通过名称解析 Provider（`WithProviderByName`），并映射到内置的 OpenAI 实现。
2. 设置所需的模型（`gpt-4o-mini`）和模式（`agent`）。
3. 将执行上下文初始化在当前目录（`WithWorkDir`）。
4. 调用 `a.Run()`，该方法返回一个 `Event` 对象的通道，其中包含文本/思考流、工具调用、执行结果和执行状态。

## 如何运行

1. 确保你在项目的根目录下。
2. 设置你的 OpenAI API 密钥：
   ```bash
   export OPENAI_API_KEY="your-api-key"
   ```
3. 运行该示例：
   ```bash
   go run example/simple_agent/main.go
   ```
