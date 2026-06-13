# Custom Provider Example

This example demonstrates how external Go developers can implement their own LLM backend and plug it seamlessly into VibeCoding's advanced Agent framework.

## How it Works

VibeCoding provides a public `Provider` interface:
```go
type Provider interface {
	Chat(ctx context.Context, params ChatParams) <-chan StreamEvent
	Name() string
	Models() []ModelInfo
	GetModel(id string) *ModelInfo
}
```

In this example, we:
1. Implement a `CustomProvider` struct that embeds `agent.BaseProvider`.
2. Define a custom mock model (`mock-model`).
3. Implement the `Chat` method to stream mock replies.
4. Integrate keywords to trigger:
   - **Text Streaming**: Simulates a standard delta-by-delta response stream.
   - **Tool Execution Request**: Simulates the LLM deciding to read a file (`README.md`) by sending a `StreamToolCall` event with a `ToolCallBlock`.
5. Run the VibeCoding agent loop with this custom provider.

Notice how VibeCoding's built-in tool registry automatically intercepts the LLM's `StreamToolCall` event, runs the actual `read` tool against the workspace safely, and provides the result back to the assistant!

## How to Run

1. Make sure you are in the root directory of the project.
2. Run the example:
   ```bash
   go run example/custom_provider/main.go
   ```
