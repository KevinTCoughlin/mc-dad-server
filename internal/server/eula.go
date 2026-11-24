package server

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
)

// EULAURL is the Minecraft EULA the operator agrees to by installing.
const EULAURL = "https://aka.ms/MinecraftEULA"

// AcceptEULA writes eula.txt accepting the Minecraft EULA.
func AcceptEULA(serverDir string) error {
	return EnsureEULA(serverDir, true)
}

// EnsureEULA checks eula.txt. When the EULA is already accepted it returns
// nil. When accept is false and the file does not agree, it returns an error
// that names the EULA and the flag that records agreement. When accept is
// true it writes eula=true atomically.
func EnsureEULA(serverDir string, accept bool) error {
	path := filepath.Join(serverDir, "eula.txt")
	if eulaAccepted(path) {
		return nil
	}
	if !accept {
		return fmt.Errorf("the Minecraft EULA is not accepted. Read %s, then rerun with --accept-eula", EULAURL)
	}
	body := fmt.Sprintf("# By changing the setting below to TRUE you are indicating your agreement to the Minecraft EULA (%s).\n# Written by mc-dad-server.\neula=true\n", EULAURL)
	if err := platform.WriteAtomic(path, []byte(body), 0o644); err != nil {
		return fmt.Errorf("could not write %s to accept the Minecraft EULA (%s): %w", path, EULAURL, err)
	}
	return nil
}

func eulaAccepted(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(key), "eula") && strings.EqualFold(strings.TrimSpace(value), "true") {
			return true
		}
	}
	return false
}
