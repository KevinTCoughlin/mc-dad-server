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
			URL string `json:"url"`
		} `json:"server"`
	} `json:"downloads"`
}

// VanillaDownloadURL resolves the download URL for a Vanilla server JAR.
func VanillaDownloadURL(ctx context.Context, version string) (string, error) {
	body, err := httpGet(ctx, "https://piston-meta.mojang.com/mc/game/version_manifest_v2.json")
	if err != nil {
		return "", fmt.Errorf("fetching version manifest: %w", err)
	}

	var manifest versionManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return "", fmt.Errorf("parsing version manifest: %w", err)
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
		return "", fmt.Errorf("minecraft version %q not found", version)
	}

	metaBody, err := httpGet(ctx, versionURL)
	if err != nil {
		return "", fmt.Errorf("fetching version metadata: %w", err)
	}

	var meta versionMeta
	if err := json.Unmarshal(metaBody, &meta); err != nil {
		return "", fmt.Errorf("parsing version metadata: %w", err)
	}

	if meta.Downloads.Server.URL == "" {
		return "", fmt.Errorf("no server download for version %s", version)
	}

	return meta.Downloads.Server.URL, nil
}
