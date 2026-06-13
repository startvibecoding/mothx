# Simple Agent Example

This example demonstrates how to use the public `agent` package of VibeCoding to construct and run a basic AI agent with streaming outputs.

## How it Works

The example utilizes the fluent `Builder` API provided by the `agent` package:
1. It resolves a provider by name (`WithProviderByName`), mapping to the built-in OpenAI implementation.
2. It sets the desired model (`gpt-4o-mini`) and mode (`agent`).
3. It initializes the execution context in the current directory (`WithWorkDir`).
4. It calls `a.Run()` which returns a channel of `Event` objects containing text/thinking streams, tool calls, results, and execution statuses.

## How to Run

1. Make sure you are in the root directory of the project.
2. Set your OpenAI API key:
   ```bash
   export OPENAI_API_KEY="your-api-key"
   ```
3. Run the example:
   ```bash
   go run example/simple_agent/main.go
   ```
