package server

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
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

func TestFabricDownloadVerifiesStrongestChecksum(t *testing.T) {
	const jarBody = "installer"
	sha512Sum := hex.EncodeToString(mustSum512(jarBody))
	sha256Sum := hex.EncodeToString(mustSum256(jarBody))

	t.Run("sha512 match", func(t *testing.T) {
		dir := t.TempDir()
		var buf strings.Builder
		runner := serveFabric(t, jarBody, map[string]string{
			".sha512": sha512Sum + "\n",
			".sha256": strings.Repeat("ab", 32),
		}, nil)
		err := FabricDownload(t.Context(), "latest", dir, runner, ui.NewWriter(&buf, false))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(buf.String(), "checksum verified (sha512)") {
			t.Fatalf("output = %q", buf.String())
		}
		if !ran(runner, "java") {
			t.Fatal("java was not run after a matching checksum")
		}
	})

	t.Run("sha512 missing falls back to sha256", func(t *testing.T) {
		dir := t.TempDir()
		var buf strings.Builder
		runner := serveFabric(t, jarBody, map[string]string{
			".sha256": sha256Sum + "  fabric-installer.jar\n",
		}, nil)
		err := FabricDownload(t.Context(), "1.21.4", dir, runner, ui.NewWriter(&buf, false))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(buf.String(), "checksum verified (sha256)") {
			t.Fatalf("output = %q", buf.String())
		}
	})

	t.Run("mismatch removes the jar", func(t *testing.T) {
		dir := t.TempDir()
		runner := serveFabric(t, jarBody, map[string]string{
			".sha512": strings.Repeat("ab", 64) + "\n",
		}, nil)
		err := FabricDownload(t.Context(), "latest", dir, runner, ui.New(false))
		if err == nil || !strings.Contains(err.Error(), "sha512 checksum mismatch") {
			t.Fatalf("error = %v", err)
		}
		if ran(runner, "java") {
			t.Fatal("java ran after a checksum mismatch")
		}
		if _, statErr := os.Stat(filepath.Join(dir, "fabric-installer.jar")); !os.IsNotExist(statErr) {
			t.Fatalf("installer jar still present: %v", statErr)
		}
	})

	t.Run("checksum HTTP 500 fails closed", func(t *testing.T) {
		dir := t.TempDir()
		runner := serveFabric(t, jarBody, nil, map[string]int{".sha512": http.StatusInternalServerError})
		err := FabricDownload(t.Context(), "latest", dir, runner, ui.New(false))
		if err == nil || !strings.Contains(err.Error(), "HTTP 500") {
			t.Fatalf("error = %v", err)
		}
		if ran(runner, "java") {
			t.Fatal("java ran after a checksum fetch failure")
		}
	})

	t.Run("no published checksum warns", func(t *testing.T) {
		dir := t.TempDir()
		var buf strings.Builder
		runner := serveFabric(t, jarBody, nil, nil)
		err := FabricDownload(t.Context(), "latest", dir, runner, ui.NewWriter(&buf, false))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(buf.String(), "downloading unverified") {
			t.Fatalf("output = %q", buf.String())
		}
		if !ran(runner, "java") {
			t.Fatal("java was not run when no checksum was published")
		}
	})
}

func serveFabric(t *testing.T, jarBody string, sums map[string]string, statuses map[string]int) *platform.MockRunner {
	t.Helper()
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	mux.HandleFunc("/installers", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]fabricInstaller{{
			URL:     srv.URL + "/installer.jar",
			Version: "1.1.2",
		}})
	})
	mux.HandleFunc("/installer.jar", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(jarBody))
	})
	for suffix, body := range sums {
		mux.HandleFunc("/installer.jar"+suffix, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(body))
		})
	}
	for suffix, status := range statuses {
		mux.HandleFunc("/installer.jar"+suffix, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
		})
	}

	original := fabricInstallerVersionsURL
	fabricInstallerVersionsURL = srv.URL + "/installers"
	t.Cleanup(func() { fabricInstallerVersionsURL = original })

	return platform.NewMockRunner()
}

func ran(runner *platform.MockRunner, name string) bool {
	for _, cmd := range runner.Commands {
		if cmd.Name == name {
			return true
		}
	}
	return false
}

func mustSum512(s string) []byte {
	sum := sha512.Sum512([]byte(s))
	return sum[:]
}

func mustSum256(s string) []byte {
	sum := sha256.Sum256([]byte(s))
	return sum[:]
}
