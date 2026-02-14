package platform

import (
	"context"
	"fmt"

	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
)

// ConfigureFirewall opens necessary ports for the Minecraft server.
func ConfigureFirewall(ctx context.Context, runner CommandRunner, plat Platform, port int, serverType string) {
	output := ui.Default()
	output.Step("Configuring Firewall")

	if plat.OS == "macos" {
		output.Info("macOS firewall: you may need to allow Java in System Settings > Privacy & Security > Firewall")
		return
	}

	portTCP := fmt.Sprintf("%d/tcp", port)

	if runner.CommandExists("ufw") {
		if err := runner.RunSudo(ctx, "ufw", "allow", portTCP, "comment", "Minecraft Server"); err != nil {
			output.Warn("Failed to configure UFW: %v", err)
			return
		}
		output.Success("UFW: opened port %s", portTCP)
		if serverType == "paper" {
			if err := runner.RunSudo(ctx, "ufw", "allow", "19132/udp", "comment", "Minecraft Bedrock (Geyser)"); err == nil {
				output.Success("UFW: opened port 19132/udp (Geyser/Bedrock)")
			}
		}
	} else if runner.CommandExists("firewall-cmd") {
		if err := runner.RunSudo(ctx, "firewall-cmd", "--permanent", "--add-port="+portTCP); err != nil {
			output.Warn("Failed to configure firewalld: %v", err)
			return
		}
		if serverType == "paper" {
			_ = runner.RunSudo(ctx, "firewall-cmd", "--permanent", "--add-port=19132/udp")
		}
		_ = runner.RunSudo(ctx, "firewall-cmd", "--reload")
		output.Success("Firewalld: opened port %s", portTCP)
	} else {
		output.Warn("No known firewall detected. You may need to manually open port %d", port)
	}
}
