package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

// defaultPaperAPIBase is the base URL for the PaperMC Fill v3 downloads API.
const defaultPaperAPIBase = "https://fill.papermc.io/v3"

// paperUserAgent identifies this tool to the PaperMC Fill v3 API,
// which requires a descriptive User-Agent header on every request.
const paperUserAgent = "mc-dad-server (https://github.com/KevinTCoughlin/mc-dad-server)"

var stablePaperVersionPattern = regexp.MustCompile(`^\d+(\.\d+){1,2}$`)

// SupportedPaperVersion is the newest Paper release this installer will
// download. Minecraft 26.4 snapshot 5 requires explicit login
// acknowledgement packets, and this project does not migrate to that protocol.
const SupportedPaperVersion = "26.3"

// MinPaperBuild is the oldest Paper 26.3 build a new install will accept.
// Fill v3 build 228 calls EntityDamageByBlockEvent for fire damage, build 229
// stops a cancelled PotionSplashEvent from extinguishing fire, and build 233
// samples the full respawn-radius square. Newer 26.3 builds stay on this line.
const MinPaperBuild = 233

type paperVersionsResponse struct {
	Versions []struct {
		Version struct {
			ID string `json:"id"`
		} `json:"version"`
	} `json:"versions"`
}

type paperBuildResponse struct {
	ID        int `json:"id"`
	Downloads map[string]struct {
		Name      string `json:"name"`
		URL       string `json:"url"`
		Checksums struct {
			SHA256 string `json:"sha256"`
		} `json:"checksums"`
	} `json:"downloads"`
}

// PaperArtifact resolves the download URL and published SHA-256 for a Paper
// server JAR.
func PaperArtifact(ctx context.Context, version string) (Artifact, error) {
	return paperArtifact(ctx, version, defaultPaperAPIBase)
}

func paperArtifact(ctx context.Context, version, apiBase string) (Artifact, error) {
	if version == "latest" || version == "" {
		var err error
		version, err = paperLatestVersion(ctx, apiBase)
		if err != nil {
			return Artifact{}, err
		}
	} else if err := validateRequestedPaperVersion(version); err != nil {
		return Artifact{}, err
	}

	url := fmt.Sprintf("%s/projects/paper/versions/%s/builds/latest", apiBase, version)
	body, err := httpGet(ctx, url)
	if err != nil {
		return Artifact{}, fmt.Errorf("fetching Paper latest build: %w", err)
	}

	var build paperBuildResponse
	if err := json.Unmarshal(body, &build); err != nil {
		return Artifact{}, fmt.Errorf("parsing Paper latest build: %w", err)
	}

	if err := validatePaperBuild(version, build.ID); err != nil {
		return Artifact{}, err
	}

	dl, ok := build.Downloads["server:default"]
	if !ok || dl.URL == "" {
		return Artifact{}, fmt.Errorf("no download found for Paper %s latest build", version)
	}

	return Artifact{URL: dl.URL, SHA256: dl.Checksums.SHA256}, nil
}

// validateRequestedPaperVersion rejects snapshots and anything newer than
// SupportedPaperVersion before the installer contacts the build endpoint.
func validateRequestedPaperVersion(version string) error {
	if !isStablePaperVersion(version) {
		return fmt.Errorf("refusing Paper version %q: snapshots, previews, and release candidates are not installed", version)
	}
	if paperVersionNewer(version, SupportedPaperVersion) {
		return fmt.Errorf("refusing Paper version %q: newest supported release is %s", version, SupportedPaperVersion)
	}
	return nil
}

// validatePaperBuild enforces MinPaperBuild on the 26.3 line. Older Minecraft
// releases use a different build sequence and are not compared against it.
func validatePaperBuild(version string, buildID int) error {
	if !requiresMinPaperBuild(version) {
		return nil
	}
	if buildID < MinPaperBuild {
		return fmt.Errorf("refusing Paper %s build %d: need build %d or newer", version, buildID, MinPaperBuild)
	}
	return nil
}

func paperLatestVersion(ctx context.Context, apiBase string) (string, error) {
	url := fmt.Sprintf("%s/projects/paper/versions", apiBase)
	body, err := httpGet(ctx, url)
	if err != nil {
		return "", fmt.Errorf("fetching Paper versions: %w", err)
	}

	var resp paperVersionsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("parsing Paper versions: %w", err)
	}

	if len(resp.Versions) == 0 {
		return "", fmt.Errorf("no Paper versions found")
	}

	// Fill v3 returns versions newest-first, but "latest" is the newest stable
	// release that is still at or below SupportedPaperVersion. A numeric 26.4
	// id would match the stable pattern and must not win.
	var best string
	sawStable := false
	for _, version := range resp.Versions {
		id := version.Version.ID
		if !isStablePaperVersion(id) {
			continue
		}
		sawStable = true
		if paperVersionNewer(id, SupportedPaperVersion) {
			continue
		}
		if best == "" || paperVersionNewer(id, best) {
			best = id
		}
	}
	if best != "" {
		return best, nil
	}
	if !sawStable {
		return "", fmt.Errorf("no stable Paper versions found")
	}
	return "", fmt.Errorf("no Paper version at or below %s found", SupportedPaperVersion)
}

func requiresMinPaperBuild(version string) bool {
	parts, ok := parsePaperVersion(version)
	if !ok || len(parts) < 2 || parts[0] != 26 || parts[1] != 3 {
		return false
	}
	return len(parts) < 3 || parts[2] == 0
}

func paperVersionNewer(a, b string) bool {
	cmp, ok := comparePaperVersions(a, b)
	if !ok {
		return true
	}
	return cmp > 0
}

func comparePaperVersions(a, b string) (int, bool) {
	ap, aok := parsePaperVersion(a)
	bp, bok := parsePaperVersion(b)
	if !aok || !bok {
		return 0, false
	}
	n := max(len(ap), len(bp))
	for i := range n {
		av, bv := 0, 0
		if i < len(ap) {
			av = ap[i]
		}
		if i < len(bp) {
			bv = bp[i]
		}
		if av != bv {
			if av > bv {
				return 1, true
			}
			return -1, true
		}
	}
	return 0, true
}

func parsePaperVersion(version string) ([]int, bool) {
	if !isStablePaperVersion(version) {
		return nil, false
	}
	parts := strings.Split(version, ".")
	nums := make([]int, len(parts))
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil {
			return nil, false
		}
		nums[i] = n
	}
	return nums, true
}

func isStablePaperVersion(version string) bool {
	return stablePaperVersionPattern.MatchString(version)
}

func httpGet(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", paperUserAgent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	return io.ReadAll(resp.Body)
}
