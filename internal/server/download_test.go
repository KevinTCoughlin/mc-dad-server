package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPaperDownloadURL_Latest(t *testing.T) {
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

	old := paperAPIBase
	paperAPIBase = srv.URL + "/v3"
	defer func() { paperAPIBase = old }()

	url, err := PaperDownloadURL(context.Background(), "latest")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "https://fill-data.papermc.io/v1/objects/abc123/paper-1.21.4-42.jar"
	if url != expected {
		t.Fatalf("got %q, want %q", url, expected)
	}
}

func TestPaperDownloadURL_SpecificVersion(t *testing.T) {
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

	old := paperAPIBase
	paperAPIBase = srv.URL + "/v3"
	defer func() { paperAPIBase = old }()

	url, err := PaperDownloadURL(context.Background(), "1.20.4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "https://fill-data.papermc.io/v1/objects/def456/paper-1.20.4-10.jar"
	if url != expected {
		t.Fatalf("got %q, want %q", url, expected)
	}
}

func TestPaperDownloadURL_NoVersions(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/v3/projects/paper/versions", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"versions": []any{},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	old := paperAPIBase
	paperAPIBase = srv.URL + "/v3"
	defer func() { paperAPIBase = old }()

	_, err := PaperDownloadURL(context.Background(), "latest")
	if err == nil {
		t.Fatal("expected error for empty versions")
	}
	if !strings.Contains(err.Error(), "no Paper versions found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPaperDownloadURL_NoDownload(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/v3/projects/paper/versions/1.21.4/builds/latest", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"downloads": map[string]any{},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	old := paperAPIBase
	paperAPIBase = srv.URL + "/v3"
	defer func() { paperAPIBase = old }()

	_, err := PaperDownloadURL(context.Background(), "1.21.4")
	if err == nil {
		t.Fatal("expected error for missing download")
	}
	if !strings.Contains(err.Error(), "no download found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPaperDownloadURL_UserAgent(t *testing.T) {
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

	old := paperAPIBase
	paperAPIBase = srv.URL + "/v3"
	defer func() { paperAPIBase = old }()

	_, err := PaperDownloadURL(context.Background(), "1.21.4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
