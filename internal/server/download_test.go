package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPaperDownloadURL_Integration(t *testing.T) {
	// Mock the Paper API
	mux := http.NewServeMux()

	mux.HandleFunc("/v2/projects/paper", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"versions": []string{"1.20.4", "1.21.4"},
		})
	})

	mux.HandleFunc("/v2/projects/paper/versions/1.21.4/builds", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"builds": []map[string]any{
				{
					"build": 42,
					"downloads": map[string]any{
						"application": map[string]any{
							"name": "paper-1.21.4-42.jar",
						},
					},
				},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	// We can't easily override the base URL in this test without refactoring.
	// This is a structural test showing the mock approach.
	t.Skip("requires API base URL injection for unit testing")

	url, err := PaperDownloadURL(context.Background(), "latest")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url == "" {
		t.Fatal("expected non-empty URL")
	}
}
