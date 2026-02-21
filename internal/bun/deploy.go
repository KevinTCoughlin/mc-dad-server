package bun

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/KevinTCoughlin/mc-dad-server/internal/config"
	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
)

// DeployScripts writes the Bun scripting sidecar files to the server directory.
// Runtime files are always overwritten (framework updates). User scripts in
// scripts/ are preserved across re-installs.
func DeployScripts(cfg *config.ServerConfig) error {
	bunDir := filepath.Join(cfg.Dir, "bun-scripts")

	// Always deploy runtime/ (framework files)
	runtimeDir := filepath.Join(bunDir, "runtime")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		return fmt.Errorf("creating runtime dir: %w", err)
	}

	runtimeFiles := []string{
		"runtime/types.ts",
		"runtime/events.ts",
		"runtime/rcon.ts",
		"runtime/log-parser.ts",
		"runtime/players.ts",
		"runtime/scheduler.ts",
		"runtime/webhooks.ts",
		"runtime/server.ts",
		"runtime/index.ts",
	}

	for _, name := range runtimeFiles {
		data, err := fs.ReadFile(embeddedFS, "embedded/bun/"+name)
		if err != nil {
			return fmt.Errorf("reading embedded %s: %w", name, err)
		}
		dest := filepath.Join(bunDir, name)
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", dest, err)
		}
	}

	// Deploy scripts/example.ts only if scripts/ directory is empty or doesn't exist
	scriptsDir := filepath.Join(bunDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		return fmt.Errorf("creating scripts dir: %w", err)
	}

	if dirEmpty(scriptsDir) {
		data, err := fs.ReadFile(embeddedFS, "embedded/bun/scripts/example.ts")
		if err != nil {
			return fmt.Errorf("reading embedded example.ts: %w", err)
		}
		dest := filepath.Join(scriptsDir, "example.ts")
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			return fmt.Errorf("writing example.ts: %w", err)
		}
	}

	// Deploy tsconfig.json (static)
	data, err := fs.ReadFile(embeddedFS, "embedded/bun/tsconfig.json")
	if err != nil {
		return fmt.Errorf("reading embedded tsconfig.json: %w", err)
	}
	if err := os.WriteFile(filepath.Join(bunDir, "tsconfig.json"), data, 0o644); err != nil {
		return fmt.Errorf("writing tsconfig.json: %w", err)
	}

	// Deploy .env from template
	if err := deployTemplate(cfg, "embedded/bun/env.tmpl", filepath.Join(bunDir, ".env")); err != nil {
		return fmt.Errorf("deploying .env: %w", err)
	}

	// Deploy package.json from template
	if err := deployTemplate(cfg, "embedded/bun/package.json.tmpl", filepath.Join(bunDir, "package.json")); err != nil {
		return fmt.Errorf("deploying package.json: %w", err)
	}

	return nil
}

// InstallDependencies runs bun install in the bun-scripts directory.
func InstallDependencies(ctx context.Context, runner platform.CommandRunner, serverDir string) error {
	bunDir := filepath.Join(serverDir, "bun-scripts")
	return runner.Run(ctx, "bash", "-c", fmt.Sprintf("cd %s && bun install", bunDir))
}

// deployTemplate reads a Go template from the embedded FS, executes it with
// config data, and writes the result to dest.
func deployTemplate(cfg *config.ServerConfig, tmplPath, dest string) error {
	data, err := fs.ReadFile(embeddedFS, tmplPath)
	if err != nil {
		return fmt.Errorf("reading template %s: %w", tmplPath, err)
	}

	tmpl, err := template.New(filepath.Base(tmplPath)).Parse(string(data))
	if err != nil {
		return fmt.Errorf("parsing template %s: %w", tmplPath, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]string{
		"RCONPassword": cfg.RCONPassword,
		"RCONPort":     "25575",
		"ServerDir":    cfg.Dir,
	}); err != nil {
		return fmt.Errorf("executing template %s: %w", tmplPath, err)
	}

	return os.WriteFile(dest, buf.Bytes(), 0o644)
}

// dirEmpty returns true if the directory is empty or doesn't exist.
func dirEmpty(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return true
	}
	// Ignore hidden files like .gitkeep
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), ".") {
			return false
		}
	}
	return true
}
