package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
	"github.com/KevinTCoughlin/mc-dad-server/internal/verify"
)

type fabricInstaller struct {
	URL     string `json:"url"`
	Version string `json:"version"`
}

var fabricInstallerVersionsURL = "https://meta.fabricmc.net/v2/versions/installer"

// fabricChecksums is the strongest-first list of sidecar files Maven
// publishes next to the installer jar. meta.fabricmc.net does not include a
// hash. maven.fabricmc.net does: .sha512, .sha256, .sha1, and .md5 (checked
// for fabric-installer 1.1.2). MD5 is ignored. The first sidecar that exists
// is the one that is checked. If none of them is published, the download
// continues with a warning rather than a hash pinned in this repo, because
// the installer version comes from the live meta API.
var fabricChecksums = []struct {
	suffix string
	alg    string
	hexLen int
}{
	{suffix: ".sha512", alg: "sha512", hexLen: 128},
	{suffix: ".sha256", alg: "sha256", hexLen: 64},
	{suffix: ".sha1", alg: "sha1", hexLen: 40},
}

// FabricDownload downloads and runs the Fabric installer.
func FabricDownload(ctx context.Context, version, destDir string, runner platform.CommandRunner, output *ui.UI) error {
	body, err := httpGet(ctx, fabricInstallerVersionsURL)
	if err != nil {
		return fmt.Errorf("fetching Fabric installer versions: %w", err)
	}

	var installers []fabricInstaller
	if err := json.Unmarshal(body, &installers); err != nil {
		return fmt.Errorf("parsing Fabric installer versions: %w", err)
	}

	if len(installers) == 0 {
		return fmt.Errorf("no Fabric installers found")
	}

	installer := installers[0]
	installerPath := filepath.Join(destDir, "fabric-installer.jar")

	if err := downloadFile(ctx, installer.URL, installerPath); err != nil {
		return fmt.Errorf("downloading Fabric installer: %w", err)
	}

	if err := verifyFabricInstaller(ctx, installer, installerPath, output); err != nil {
		_ = os.Remove(installerPath)
		return err
	}

	mcVersion := version
	if mcVersion == "latest" {
		mcVersion = "" // Fabric installer uses latest by default
	}

	args := []string{"-jar", installerPath, "server", "-downloadMinecraft"}
	if mcVersion != "" {
		args = append(args, "-mcversion", mcVersion)
	}

	if err := runner.Run(ctx, "java", args...); err != nil {
		return fmt.Errorf("running Fabric installer: %w", err)
	}

	// Rename the output
	src := filepath.Join(destDir, "fabric-server-launch.jar")
	dst := filepath.Join(destDir, "server.jar")
	if err := runner.Run(ctx, "mv", src, dst); err != nil {
		return fmt.Errorf("renaming Fabric server launcher: %w", err)
	}

	// Cleanup installer
	_ = runner.Run(ctx, "rm", "-f", installerPath)

	output.Success("Fabric server downloaded")
	return nil
}

func verifyFabricInstaller(ctx context.Context, installer fabricInstaller, path string, output *ui.UI) error {
	alg, sum, published, err := fetchFabricChecksum(ctx, installer.URL)
	if err != nil {
		return err
	}
	if !published {
		label := installer.Version
		if label == "" {
			label = installer.URL
		}
		output.Warn("Fabric installer %s publishes no .sha512, .sha256, or .sha1 next to the jar — downloading unverified", label)
		return nil
	}

	expected := verify.Expected{}
	switch alg {
	case "sha512":
		expected.SHA512 = sum
	case "sha256":
		expected.SHA256 = sum
	case "sha1":
		expected.SHA1 = sum
	default:
		return fmt.Errorf("unknown Fabric checksum algorithm %q", alg)
	}
	if err := verify.File(path, expected); err != nil {
		return fmt.Errorf("verifying Fabric installer: %w", err)
	}
	output.Success("Fabric installer checksum verified (%s)", alg)
	return nil
}

// fetchFabricChecksum returns the strongest checksum published beside
// artifactURL. published is false when every sidecar responds 404.
func fetchFabricChecksum(ctx context.Context, artifactURL string) (alg, sum string, published bool, err error) {
	for _, candidate := range fabricChecksums {
		body, status, getErr := httpGetStatus(ctx, artifactURL+candidate.suffix)
		if getErr != nil {
			return "", "", false, fmt.Errorf("fetching Fabric installer %s checksum: %w", candidate.alg, getErr)
		}
		if status == http.StatusNotFound {
			continue
		}
		if status != http.StatusOK {
			return "", "", false, fmt.Errorf("fetching Fabric installer %s checksum: HTTP %d", candidate.alg, status)
		}
		parsed, ok := parseMavenChecksum(body, candidate.hexLen)
		if !ok {
			return "", "", false, fmt.Errorf("fabric installer %s checksum is malformed", candidate.alg)
		}
		return candidate.alg, parsed, true, nil
	}
	return "", "", false, nil
}

func parseMavenChecksum(body []byte, hexLen int) (string, bool) {
	for _, field := range strings.Fields(string(body)) {
		if len(field) == hexLen && isHex(field) {
			return strings.ToLower(field), true
		}
	}
	return "", false
}

func isHex(s string) bool {
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		case r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return s != ""
}

func httpGetStatus(ctx context.Context, url string) (body []byte, status int, err error) {
	ctx, cancel := platform.WithTimeout(ctx, paperAPITimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", paperUserAgent)

	resp, err := paperAPIClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err = io.ReadAll(io.LimitReader(resp.Body, 1024))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}
