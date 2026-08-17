package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
)

func TestFabricDownloadRenameFailureRestoresExistingJAR(t *testing.T) {
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	mux.HandleFunc("/installers", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]fabricInstaller{{URL: srv.URL + "/installer.jar"}})
	})
	mux.HandleFunc("/installer.jar", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("installer"))
	})

	originalURL := fabricInstallerVersionsURL
	fabricInstallerVersionsURL = srv.URL + "/installers"
	defer func() { fabricInstallerVersionsURL = originalURL }()

	dir := t.TempDir()
	jarPath := filepath.Join(dir, "server.jar")
	if err := os.WriteFile(jarPath, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	runner := platform.NewMockRunner()
	src := filepath.Join(dir, "fabric-server-launch.jar")
	runner.ErrorMap["mv ["+src+" "+jarPath+"]"] = errors.New("rename failed")

	err := Download(t.Context(), "fabric", "latest", dir, runner, ui.New(false))
	if err == nil {
		t.Fatal("expected rename failure")
	}
	if !strings.Contains(err.Error(), "renaming Fabric server launcher") {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := os.ReadFile(jarPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "existing" {
		t.Fatalf("server.jar = %q, want previous contents", got)
	}
	if _, err := os.Stat(jarPath + ".bak"); !os.IsNotExist(err) {
		t.Fatalf("unexpected backup after restoration: %v", err)
	}
}
