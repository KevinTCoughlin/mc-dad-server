package server

import (
	"os"
	"path/filepath"
)

// AcceptEULA writes the eula.txt file accepting the Minecraft EULA.
func AcceptEULA(serverDir string) error {
	return os.WriteFile(filepath.Join(serverDir, "eula.txt"), []byte("eula=true\n"), 0o644)
}
