package provider

import (
	"fmt"
	"sync/atomic"
)

var toolCallFallbackCounter uint64

// NextToolCallFallbackID returns a process-wide unique fallback ID for tool calls.
func NextToolCallFallbackID(prefix string) string {
	n := atomic.AddUint64(&toolCallFallbackCounter, 1)
	return fmt.Sprintf("%s_%d", prefix, n)
}
