package server

import (
	"context"
	"encoding/json"
	"fmt"
)

type versionManifest struct {
	Latest struct {
		Release string `json:"release"`
	} `json:"latest"`
	Versions []struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	} `json:"versions"`
}

type versionMeta struct {
	Downloads struct {
		Server struct {
			URL  string `json:"url"`
			SHA1 string `json:"sha1"`
		} `json:"server"`
	} `json:"downloads"`
}

// VanillaArtifact resolves the download URL and published SHA-1 for a Vanilla
// server JAR.
func VanillaArtifact(ctx context.Context, version string) (Artifact, error) {
	body, err := httpGet(ctx, "https://piston-meta.mojang.com/mc/game/version_manifest_v2.json")
	if err != nil {
		return Artifact{}, fmt.Errorf("fetching version manifest: %w", err)
	}

	var manifest versionManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return Artifact{}, fmt.Errorf("parsing version manifest: %w", err)
	}

	if version == "latest" {
		version = manifest.Latest.Release
	}

	var versionURL string
	for _, v := range manifest.Versions {
		if v.ID == version {
			versionURL = v.URL
			break
		}
	}
	if versionURL == "" {
		return Artifact{}, fmt.Errorf("minecraft version %q not found", version)
	}

	metaBody, err := httpGet(ctx, versionURL)
	if err != nil {
		return Artifact{}, fmt.Errorf("fetching version metadata: %w", err)
	}

	var meta versionMeta
	if err := json.Unmarshal(metaBody, &meta); err != nil {
		return Artifact{}, fmt.Errorf("parsing version metadata: %w", err)
	}

	if meta.Downloads.Server.URL == "" {
		return Artifact{}, fmt.Errorf("no server download for version %s", version)
	}

	return Artifact{URL: meta.Downloads.Server.URL, SHA1: meta.Downloads.Server.SHA1}, nil
}
