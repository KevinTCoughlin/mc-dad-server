package config

import (
	"fmt"
	"strings"
)

// BedrockPort is the default Geyser/Bedrock cross-play port.
const BedrockPort = 19132

// ServerConfig holds all configuration for a Minecraft server install.
type ServerConfig struct {
	Edition      string `json:"edition"`
	Dir          string `json:"dir"`
	Port         int    `json:"port"`
	Memory       string `json:"memory"`
	ServerType   string `json:"server_type"`
	MOTD         string `json:"motd"`
	MaxPlayers   int    `json:"max_players"`
	Difficulty   string `json:"difficulty"`
	GameMode     string `json:"gamemode"`
	GCType       string `json:"gc_type"`
	Whitelist    bool   `json:"whitelist"`
	ChatFilter   bool   `json:"chat_filter"`
	EnablePlayit bool   `json:"enable_playit"`
	Version      string `json:"version"`
	SessionName  string `json:"session_name"`
	MaxBackups   int    `json:"max_backups"`
	VoteDuration int    `json:"vote_duration"`
	VoteChoices  int    `json:"vote_choices"`

	// Generated at runtime
	RCONPassword string `json:"-"`
}

// DefaultConfig returns a ServerConfig with sensible defaults matching install.sh.
func DefaultConfig() *ServerConfig {
	return &ServerConfig{
		Edition:      "java",
		Dir:          "",
		Port:         25565,
		Memory:       "2G",
		ServerType:   "paper",
		MOTD:         "Dads Minecraft Server",
		MaxPlayers:   20,
		Difficulty:   "normal",
		GameMode:     "survival",
		GCType:       "g1gc",
		Whitelist:    true,
		ChatFilter:   true,
		EnablePlayit: true,
		Version:      "latest",
		SessionName:  "minecraft",
		MaxBackups:   5,
		VoteDuration: 30,
		VoteChoices:  5,
	}
}

var (
	validEditions     = map[string]bool{"java": true, "bedrock": true}
	validServerTypes  = map[string]bool{"paper": true, "fabric": true, "vanilla": true}
	validDifficulties = map[string]bool{"peaceful": true, "easy": true, "normal": true, "hard": true}
	validGameModes    = map[string]bool{"survival": true, "creative": true, "adventure": true}
	validGCTypes      = map[string]bool{"g1gc": true, "zgc": true}
)

// Validate checks that all config values are valid.
func (c *ServerConfig) Validate() error {
	if !validEditions[c.Edition] {
		return fmt.Errorf("invalid edition %q: must be java or bedrock", c.Edition)
	}
	if !validServerTypes[c.ServerType] {
		return fmt.Errorf("invalid server type %q: must be paper, fabric, or vanilla", c.ServerType)
	}
	if !validDifficulties[c.Difficulty] {
		return fmt.Errorf("invalid difficulty %q: must be peaceful, easy, normal, or hard", c.Difficulty)
	}
	if !validGameModes[c.GameMode] {
		return fmt.Errorf("invalid gamemode %q: must be survival, creative, or adventure", c.GameMode)
	}
	gc := strings.ToLower(c.GCType)
	if !validGCTypes[gc] {
		return fmt.Errorf("invalid gc type %q: must be g1gc or zgc", c.GCType)
	}
	c.GCType = gc

	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid port %d: must be 1-65535", c.Port)
	}
	if c.MaxPlayers < 1 {
		return fmt.Errorf("invalid max players %d: must be >= 1", c.MaxPlayers)
	}
	if c.MaxBackups < 1 {
		return fmt.Errorf("invalid max backups %d: must be >= 1", c.MaxBackups)
	}
	if c.Dir == "" {
		return fmt.Errorf("server directory must be set")
	}
	return nil
}
