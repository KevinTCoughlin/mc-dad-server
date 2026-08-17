package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
)

func TestPaperDownloadURL_Latest(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()

	mux.HandleFunc("/v3/projects/paper/versions", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"versions": []map[string]any{
				{"version": map[string]any{"id": "1.21.4"}},
				{"version": map[string]any{"id": "1.20.4"}},
			},
		})
	})

	mux.HandleFunc("/v3/projects/paper/versions/1.21.4/builds/latest", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"downloads": map[string]any{
				"server:default": map[string]any{
					"name": "paper-1.21.4-42.jar",
					"url":  "https://fill-data.papermc.io/v1/objects/abc123/paper-1.21.4-42.jar",
					"checksums": map[string]any{
						"sha256": "abcdef1234567890",
					},
				},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	art, err := paperArtifact(context.Background(), "latest", srv.URL+"/v3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "https://fill-data.papermc.io/v1/objects/abc123/paper-1.21.4-42.jar"
	if art.URL != expected {
		t.Fatalf("got %q, want %q", art.URL, expected)
	}
}

func TestPaperDownloadURL_LatestSkipsRCVersions(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()

	mux.HandleFunc("/v3/projects/paper/versions", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"versions": []map[string]any{
				{"version": map[string]any{"id": "1.21.9-rc-1"}},
				{"version": map[string]any{"id": "1.21.8"}},
				{"version": map[string]any{"id": "1.21.7"}},
			},
		})
	})

	mux.HandleFunc("/v3/projects/paper/versions/1.21.8/builds/latest", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"downloads": map[string]any{
				"server:default": map[string]any{
					"name": "paper-1.21.8-1.jar",
					"url":  "https://fill-data.papermc.io/v1/objects/stable123/paper-1.21.8-1.jar",
					"checksums": map[string]any{
						"sha256": "stable123",
					},
				},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	art, err := paperArtifact(context.Background(), "latest", srv.URL+"/v3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "https://fill-data.papermc.io/v1/objects/stable123/paper-1.21.8-1.jar"
	if art.URL != expected {
		t.Fatalf("got %q, want %q", art.URL, expected)
	}
}

func TestPaperDownloadURL_SpecificVersion(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()

	mux.HandleFunc("/v3/projects/paper/versions/1.20.4/builds/latest", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"downloads": map[string]any{
				"server:default": map[string]any{
					"name": "paper-1.20.4-10.jar",
					"url":  "https://fill-data.papermc.io/v1/objects/def456/paper-1.20.4-10.jar",
					"checksums": map[string]any{
						"sha256": "def4567890abcdef",
					},
				},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	art, err := paperArtifact(context.Background(), "1.20.4", srv.URL+"/v3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "https://fill-data.papermc.io/v1/objects/def456/paper-1.20.4-10.jar"
	if art.URL != expected {
		t.Fatalf("got %q, want %q", art.URL, expected)
	}
}

func TestPaperDownloadURL_NoVersions(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()

	mux.HandleFunc("/v3/projects/paper/versions", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"versions": []any{},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	_, err := paperArtifact(context.Background(), "latest", srv.URL+"/v3")
	if err == nil {
		t.Fatal("expected error for empty versions")
	}
	if !strings.Contains(err.Error(), "no Paper versions found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPaperDownloadURL_NoStableVersions(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()

	mux.HandleFunc("/v3/projects/paper/versions", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"versions": []map[string]any{
				{"version": map[string]any{"id": "1.21.9-rc-1"}},
				{"version": map[string]any{"id": "1.21.9-beta-1"}},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	_, err := paperArtifact(context.Background(), "latest", srv.URL+"/v3")
	if err == nil {
		t.Fatal("expected error for missing stable versions")
	}
	if !strings.Contains(err.Error(), "no stable Paper versions found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPaperDownloadURL_NoDownload(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()

	mux.HandleFunc("/v3/projects/paper/versions/1.21.4/builds/latest", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"downloads": map[string]any{},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	_, err := paperArtifact(context.Background(), "1.21.4", srv.URL+"/v3")
	if err == nil {
		t.Fatal("expected error for missing download")
	}
	if !strings.Contains(err.Error(), "no download found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPaperDownloadURL_UserAgent(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()

	mux.HandleFunc("/v3/projects/paper/versions/1.21.4/builds/latest", func(w http.ResponseWriter, r *http.Request) {
		ua := r.Header.Get("User-Agent")
		if ua != paperUserAgent {
			t.Errorf("User-Agent = %q, want %q", ua, paperUserAgent)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"downloads": map[string]any{
				"server:default": map[string]any{
					"name": "paper-1.21.4-1.jar",
					"url":  "https://fill-data.papermc.io/v1/objects/xyz/paper-1.21.4-1.jar",
					"checksums": map[string]any{
						"sha256": "xyz",
					},
				},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	_, err := paperArtifact(context.Background(), "1.21.4", srv.URL+"/v3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFetchAndVerifyRetriesTransientHTTPFailure(t *testing.T) {
	setDownloadRetryDelay(t, 0)

	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if attempts.Add(1) < downloadAttempts {
			http.Error(w, "temporary outage", http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte("hello\n"))
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "server.jar")
	art := Artifact{URL: srv.URL, SHA256: helloSHA256}
	if err := fetchAndVerify(context.Background(), art, dest, ui.NewWriter(&bytes.Buffer{}, false)); err != nil {
		t.Fatalf("fetchAndVerify() error = %v", err)
	}
	if attempts.Load() != downloadAttempts {
		t.Fatalf("attempts = %d, want %d", attempts.Load(), downloadAttempts)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("reading dest: %v", err)
	}
	if string(got) != "hello\n" {
		t.Fatalf("dest = %q, want hello fixture", got)
	}
}

func TestFetchAndVerifyKeepsExistingJarOnChecksumFailure(t *testing.T) {
	setDownloadRetryDelay(t, 0)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("corrupt\n"))
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "server.jar")
	if err := os.WriteFile(dest, []byte("existing\n"), 0o644); err != nil {
		t.Fatalf("writing existing jar: %v", err)
	}

	art := Artifact{URL: srv.URL, SHA256: helloSHA256}
	err := fetchAndVerify(context.Background(), art, dest, ui.NewWriter(&bytes.Buffer{}, false))
	if err == nil {
		t.Fatal("expected checksum failure")
	}
	if !strings.Contains(err.Error(), "verifying server JAR") {
		t.Fatalf("error = %v, want verifying server JAR", err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("reading dest: %v", err)
	}
	if string(got) != "existing\n" {
		t.Fatalf("dest = %q, want existing jar to remain", got)
	}
}

func setDownloadRetryDelay(t *testing.T, delay time.Duration) {
	t.Helper()

	old := downloadRetryDelay
	downloadRetryDelay = delay
	t.Cleanup(func() {
		downloadRetryDelay = old
	})
}
