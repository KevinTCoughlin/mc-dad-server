package plugins

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
	"github.com/KevinTCoughlin/mc-dad-server/internal/verify"
)

func jarBody() []byte { return []byte("jar-bytes") }

// newGeyserServer serves the GeyserMC builds API shape plus the download.
func newGeyserServer(t *testing.T, sha string) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/v2/projects/geyser/versions/latest/builds/latest", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/downloads/spigot") {
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"build": 812,
			"downloads": map[string]any{
				"spigot": map[string]any{"name": "Geyser-Spigot.jar", "sha256": sha},
			},
		})
	})
	mux.HandleFunc("/v2/projects/geyser/versions/latest/builds/latest/downloads/spigot",
		func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write(jarBody())
		})

	return httptest.NewServer(mux)
}

func TestGeyserSource(t *testing.T) {
	srv := newGeyserServer(t, "abc123")
	defer srv.Close()

	geyserAPIBase = srv.URL + "/v2"

	src, err := geyserSource(t.Context(), "geyser")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if src.expected.SHA256 != "abc123" {
		t.Fatalf("sha256 = %q, want abc123", src.expected.SHA256)
	}
	if !strings.HasSuffix(src.url, "/downloads/spigot") {
		t.Fatalf("unexpected url %q", src.url)
	}
}

func TestHangarSourceStripsQuotesFromVersion(t *testing.T) {
	mux := http.NewServeMux()
	// The real endpoint returns a bare JSON string, quotes included.
	mux.HandleFunc("/api/v1/projects/WorldEdit/latestrelease", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`"7.3.0"`))
	})
	mux.HandleFunc("/api/v1/projects/WorldEdit/versions/7.3.0", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"downloads": map[string]any{
				"PAPER": map[string]any{
					"fileInfo": map[string]any{"name": "worldedit.jar", "sha256Hash": "deadbeef"},
				},
			},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	hangarAPIBase = srv.URL + "/api/v1"

	src, err := hangarSource(t.Context(), "WorldEdit")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if src.expected.SHA256 != "deadbeef" {
		t.Fatalf("sha256 = %q, want deadbeef", src.expected.SHA256)
	}
	want := srv.URL + "/api/v1/projects/WorldEdit/versions/7.3.0/PAPER/download"
	if src.url != want {
		t.Fatalf("url = %q, want %q (quotes must be stripped from the version)", src.url, want)
	}
}

func TestGithubSourcePicksJarAndSize(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/A5H73Y/Parkour/releases/latest", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"assets": []map[string]any{
				{"name": "checksums.txt", "size": 12, "browser_download_url": "https://example.invalid/checksums.txt"},
				{"name": "Parkour-7.2.1.jar", "size": 9, "browser_download_url": "https://example.invalid/Parkour.jar"},
			},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	githubAPIBase = srv.URL

	src, err := githubSource(t.Context(), "A5H73Y", "Parkour")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if src.url != "https://example.invalid/Parkour.jar" {
		t.Fatalf("picked the wrong asset: %q", src.url)
	}
	if src.expected.Size != 9 {
		t.Fatalf("size = %d, want 9", src.expected.Size)
	}
	if src.expected.SHA256 != "" {
		t.Fatal("GitHub publishes no digest; SHA256 should be empty")
	}
}

func TestGithubSourceNoJarAsset(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/releases/latest", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"assets": []map[string]any{{"name": "notes.md", "size": 1, "browser_download_url": "https://example.invalid/n"}},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	githubAPIBase = srv.URL

	if _, err := githubSource(t.Context(), "o", "r"); err == nil {
		t.Fatal("expected an error when no .jar asset exists")
	}
}

func TestDownloadVerifiedRejectsBadChecksum(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(jarBody())
	}))
	defer srv.Close()

	dir := t.TempDir()
	dest := filepath.Join(dir, "Plugin.jar")

	src := source{url: srv.URL, expected: verify.Expected{SHA256: strings.Repeat("0", 64)}}
	if err := downloadVerified(t.Context(), src, dest); err == nil {
		t.Fatal("expected a checksum error")
	}

	// A rejected download must not land at dest, and must not leave debris
	// that a later run could mistake for a good plugin.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no files left behind, got %v", entries)
	}
}

func TestDownloadVerifiedRejectsBadSize(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(jarBody())
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "Plugin.jar")
	src := source{url: srv.URL, expected: verify.Expected{Size: 999}}

	if err := downloadVerified(t.Context(), src, dest); err == nil {
		t.Fatal("expected a size error")
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatal("a size-mismatched download must not land at dest")
	}
}

func TestDownloadVerifiedAcceptsGoodChecksum(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(jarBody())
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "Plugin.jar")
	src := source{url: srv.URL, expected: verify.Expected{Size: int64(len(jarBody()))}}

	if err := downloadVerified(t.Context(), src, dest); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, jarBody()) {
		t.Fatalf("content = %q, want %q", got, jarBody())
	}
}

func TestInstallPluginSkipsExisting(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Plugin.jar"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	resolved := false
	r := resolver{
		name:     "Plugin",
		filename: "Plugin.jar",
		resolve: func(context.Context) (source, error) {
			resolved = true
			return source{}, nil
		},
	}

	installPlugin(t.Context(), r, dir, ui.New(false))

	if resolved {
		t.Fatal("an already-installed plugin must not hit the upstream API")
	}
	got, err := os.ReadFile(filepath.Join(dir, "Plugin.jar"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "old" {
		t.Fatalf("existing plugin was overwritten: %q", got)
	}
}

func TestInstallPluginSurvivesResolveFailure(t *testing.T) {
	dir := t.TempDir()
	r := resolver{
		name:     "Plugin",
		filename: "Plugin.jar",
		resolve: func(context.Context) (source, error) {
			return source{}, errors.New("upstream down")
		},
	}

	// A failing plugin is reported, not fatal — installPlugin returns nothing
	// and InstallAll continues with the rest of the set.
	installPlugin(t.Context(), r, dir, ui.New(false))

	if _, err := os.Stat(filepath.Join(dir, "Plugin.jar")); !os.IsNotExist(err) {
		t.Fatal("nothing should have been written")
	}
}
