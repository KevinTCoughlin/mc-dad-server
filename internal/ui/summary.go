package ui

import (
	"fmt"
	"strings"
)

// InstallSummary holds data for the install completion summary.
type InstallSummary struct {
	ServerDir    string
	ServerType   string
	Port         int
	BedrockPort  int
	Memory       string
	GCType       string
	Whitelist    bool
	Difficulty   string
	GameMode     string
	ChatFilter   bool
	PlayitSetup  bool
	LicenseLabel string
	InitSystem   string
}

// PrintInstallSummary displays the completion summary after install.
func (u *UI) PrintInstallSummary(s *InstallSummary) {
	divider := strings.Repeat("═", 54)

	fmt.Println()
	fmt.Println(u.colorize(colorGreen+colorBold, divider))
	fmt.Println(u.colorize(colorGreen+colorBold, "  MC Dad Server - Installation Complete!"))
	fmt.Println(u.colorize(colorGreen+colorBold, divider))
	fmt.Println()

	fmt.Printf("  %s  %s\n", u.Bold("Server Directory:"), s.ServerDir)
	fmt.Printf("  %s       %s\n", u.Bold("Server Type:"), s.ServerType)
	fmt.Printf("  %s              %d (Java)\n", u.Bold("Port:"), s.Port)
	if s.ServerType == "paper" {
		fmt.Printf("  %s     %d (Geyser)\n", u.Bold("Bedrock Port:"), s.BedrockPort)
	}
	fmt.Printf("  %s            %s\n", u.Bold("Memory:"), s.Memory)
	fmt.Printf("  %s                %s\n", u.Bold("GC:"), strings.ToUpper(s.GCType))
	fmt.Printf("  %s         %v\n", u.Bold("Whitelist:"), s.Whitelist)
	fmt.Printf("  %s        %s\n", u.Bold("Difficulty:"), s.Difficulty)
	fmt.Printf("  %s         %s\n", u.Bold("Game Mode:"), s.GameMode)
	fmt.Printf("  %s          %s\n", u.Bold("License:"), s.LicenseLabel)
	fmt.Println()

	if s.ServerType == "paper" {
		fmt.Println(u.colorize(colorCyan+colorBold, "  Plugins Installed:"))
		fmt.Println("    Geyser + Floodgate  (Bedrock cross-play)")
		fmt.Println("    Parkour             (obstacle courses)")
		fmt.Println("    WorldEdit           (fast building)")
		fmt.Println("    Multiverse-Core     (multiple worlds)")
		if s.ChatFilter {
			fmt.Println("    ChatSentry          (chat filter)")
		}
		fmt.Println()
	}

	fmt.Println(u.colorize(colorCyan+colorBold, "  Quick Start:"))
	fmt.Printf("    Start server:      %s\n", u.Bold("mc-dad-server start"))
	fmt.Printf("    Stop server:       %s\n", u.Bold("mc-dad-server stop"))
	fmt.Printf("    Server status:     %s\n", u.Bold("mc-dad-server status"))
	fmt.Printf("    View console:      %s\n", u.Bold("screen -r minecraft"))
	fmt.Printf("    Backup world:      %s\n", u.Bold("mc-dad-server backup"))
	fmt.Println()

	if s.ServerType == "paper" {
		fmt.Println(u.colorize(colorCyan+colorBold, "  Bedrock Cross-Play (iPad/Switch/Phone):"))
		fmt.Printf("    Kids on Bedrock connect to your IP on port %s\n", u.Bold(fmt.Sprintf("%d", s.BedrockPort)))
		fmt.Println("    No Minecraft Java account needed (Floodgate)")
		fmt.Println()
		fmt.Println(u.colorize(colorCyan+colorBold, "  Parkour Setup (first time):"))
		fmt.Printf("    %s\n", u.Bold("mc-dad-server setup-parkour"))
		fmt.Println()
	}

	if s.ServerType == "paper" && s.ChatFilter {
		fmt.Println(u.colorize(colorCyan+colorBold, "  Chat Filter:"))
		fmt.Printf("    Blocked words list: %s\n", u.Bold(s.ServerDir+"/blocked-words.txt"))
		fmt.Println("    Edit the list to customize for your family")
		fmt.Println()
	}

	if s.InitSystem == "systemd" {
		fmt.Println(u.colorize(colorCyan+colorBold, "  Systemd:"))
		fmt.Println("    sudo systemctl start minecraft")
		fmt.Println("    sudo systemctl status minecraft")
		fmt.Println()
	}

	if s.PlayitSetup {
		fmt.Println(u.colorize(colorCyan+colorBold, "  Multiplayer (No Port Forwarding):"))
		fmt.Printf("    Run %s and follow the setup link\n", u.Bold("playit"))
		fmt.Println()
	}

	fmt.Printf("  %s Your kids connect with: %s (same machine)\n",
		u.colorize(colorYellow+colorBold, "Tip:"),
		u.Bold(fmt.Sprintf("localhost:%d", s.Port)))
	fmt.Printf("       Or your %s (same network)\n",
		u.Bold(fmt.Sprintf("local IP:%d", s.Port)))
	fmt.Println()
	fmt.Println(u.colorize(colorGreen+colorBold, divider))
	fmt.Println()
}
