package hermes

import (
	"github.com/startvibecoding/vibecoding/internal/hermes/remotetui"
)

// ClientOptions configures the hermes client.
type ClientOptions struct {
	URL       string
	SessionID string
	AuthToken string
	Model     string
	WorkDir   string
}

// RunClient starts the hermes client with the Bubble Tea TUI.
func RunClient(opts ClientOptions) error {
	return remotetui.Run(remotetui.Options{
		URL:       opts.URL,
		SessionID: opts.SessionID,
		AuthToken: opts.AuthToken,
		Model:     opts.Model,
		WorkDir:   opts.WorkDir,
	})
}
