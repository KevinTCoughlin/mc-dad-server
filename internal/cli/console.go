package cli

import (
	"github.com/KevinTCoughlin/mc-dad-server/internal/console"
	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
)

// ConsoleCmd opens an interactive console with live server log.
type ConsoleCmd struct{}

// Run starts the interactive console TUI.
func (cmd *ConsoleCmd) Run(globals *Globals, runner platform.CommandRunner) error {
	return console.Run(&console.Options{
		Dir:     globals.Dir,
		Session: globals.Session,
	}, runner)
}
