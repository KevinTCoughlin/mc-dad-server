package plugins

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/KevinTCoughlin/mc-dad-server/internal/configs"
	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
)

// SetupChatFilter deploys the ChatSentry config and blocked words list.
func SetupChatFilter(serverDir string, output *ui.UI) error {
	// Deploy blocked words
	if err := configs.DeployBlockedWords(serverDir); err != nil {
		return fmt.Errorf("deploying blocked words: %w", err)
	}
	output.Success("Blocked words list deployed")

	// Download ChatSentry
	pluginsDir := filepath.Join(serverDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		return err
	}

	// Deploy ChatSentry config
	if err := configs.DeployChatSentryConfig(serverDir); err != nil {
		return fmt.Errorf("deploying ChatSentry config: %w", err)
	}
	output.Success("ChatSentry configured with blocked words filter")

	return nil
}
