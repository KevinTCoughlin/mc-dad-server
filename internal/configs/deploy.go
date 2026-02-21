package configs

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/KevinTCoughlin/mc-dad-server/internal/config"
)

func readEmbedded(name string) ([]byte, error) {
	return fs.ReadFile(embeddedFS, name)
}

// Deploy writes all embedded config files to the server directory,
// performing template substitution on server.properties.
func Deploy(cfg *config.ServerConfig) error {
	// Base configs (server root)
	baseConfigs := []string{"server.properties", "bukkit.yml", "spigot.yml"}
	for _, name := range baseConfigs {
		data, err := readEmbedded("embedded/configs/" + name)
		if err != nil {
			return fmt.Errorf("reading embedded %s: %w", name, err)
		}
		dest := filepath.Join(cfg.Dir, name)
		content := string(data)

		if name == "server.properties" {
			content = substituteProperties(content, cfg)
		}

		if err := os.WriteFile(dest, []byte(content), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", dest, err)
		}
	}

	// Paper configs go in config/ subdirectory
	configDir := filepath.Join(cfg.Dir, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	paperConfigs := []string{"paper-global.yml", "paper-world-defaults.yml"}
	for _, name := range paperConfigs {
		data, err := readEmbedded("embedded/configs/" + name)
		if err != nil {
			return fmt.Errorf("reading embedded %s: %w", name, err)
		}
		dest := filepath.Join(configDir, name)
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", dest, err)
		}
	}

	// Parkour plugin config (Paper only)
	if cfg.ServerType == "paper" {
		parkourDir := filepath.Join(cfg.Dir, "plugins", "Parkour")
		if err := os.MkdirAll(parkourDir, 0o755); err != nil {
			return fmt.Errorf("creating Parkour config dir: %w", err)
		}
		data, err := readEmbedded("embedded/configs/parkour-config.yml")
		if err != nil {
			return fmt.Errorf("reading embedded parkour config: %w", err)
		}
		dest := filepath.Join(parkourDir, "config.yml")
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", dest, err)
		}
	}

	return nil
}

func substituteProperties(content string, cfg *config.ServerConfig) string {
	replacer := strings.NewReplacer(
		"%%MC_PORT%%", fmt.Sprintf("%d", cfg.Port),
		"%%MC_MOTD%%", cfg.MOTD,
		"%%MC_DIFFICULTY%%", cfg.Difficulty,
		"%%MC_GAMEMODE%%", cfg.GameMode,
		"%%MC_MAX_PLAYERS%%", fmt.Sprintf("%d", cfg.MaxPlayers),
		"%%MC_WHITELIST%%", fmt.Sprintf("%v", cfg.Whitelist),
		"%%MC_RCON_PASSWORD%%", cfg.RCONPassword,
	)
	return replacer.Replace(content)
}

// DeployBlockedWords writes the embedded blocked words list to the server directory.
func DeployBlockedWords(serverDir string) error {
	data, err := readEmbedded("embedded/blocked-words.txt")
	if err != nil {
		return fmt.Errorf("reading embedded blocked-words.txt: %w", err)
	}
	return os.WriteFile(filepath.Join(serverDir, "blocked-words.txt"), data, 0o644)
}

// DeployChatSentryConfig writes the ChatSentry config to the plugins directory.
func DeployChatSentryConfig(serverDir string) error {
	sentryDir := filepath.Join(serverDir, "plugins", "ChatSentry")
	if err := os.MkdirAll(sentryDir, 0o755); err != nil {
		return err
	}

	data, err := readEmbedded("embedded/configs/chatsentry-config.yml")
	if err != nil {
		return fmt.Errorf("reading embedded chatsentry config: %w", err)
	}
	return os.WriteFile(filepath.Join(sentryDir, "config.yml"), data, 0o644)
}

// DeployStartScript renders and writes the start.sh script.
func DeployStartScript(cfg *config.ServerConfig) error {
	data, err := readEmbedded("embedded/templates/start.sh.tmpl")
	if err != nil {
		return fmt.Errorf("reading start.sh template: %w", err)
	}

	tmpl, err := template.New("start.sh").Parse(string(data))
	if err != nil {
		return fmt.Errorf("parsing start.sh template: %w", err)
	}

	dest := filepath.Join(cfg.Dir, "start.sh")
	f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("creating start.sh: %w", err)
	}
	defer func() { _ = f.Close() }()

	return tmpl.Execute(f, map[string]any{
		"Memory":    cfg.Memory,
		"GCType":    cfg.GCType,
		"EnableBun": cfg.EnableBun,
	})
}
