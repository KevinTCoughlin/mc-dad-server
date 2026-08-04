package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
	"github.com/KevinTCoughlin/mc-dad-server/internal/verify"
)

// httpClient is used for all plugin API calls and downloads. http.DefaultClient
// has no timeout, so an unresponsive upstream would hang the installer
// indefinitely.
var httpClient = &http.Client{Timeout: 5 * time.Minute}

// source is a resolved plugin download: where to fetch it and what integrity
// metadata the upstream API published for it.
type source struct {
	url      string
	expected verify.Expected
}

// resolver locates the download for one plugin. Resolvers hit the upstream
// API, so they can fail independently of the download itself.
type resolver struct {
	name     string
	filename string
	resolve  func(context.Context) (source, error)

	// manualHint is shown when resolution fails, so the user knows where to
	// get the plugin by hand.
	manualHint string
}

// defaultResolvers returns the plugin set installed on a Paper server. The
// integrity metadata each one carries mirrors what the Containerfile verifies.
func defaultResolvers() []resolver {
	return []resolver{
		{
			name:     "Geyser",
			filename: "Geyser-Spigot.jar",
			resolve:  func(ctx context.Context) (source, error) { return geyserSource(ctx, "geyser") },
		},
		{
			name:     "Floodgate",
			filename: "Floodgate-Spigot.jar",
			resolve:  func(ctx context.Context) (source, error) { return geyserSource(ctx, "floodgate") },
		},
		{
			name:       "Parkour",
			filename:   "Parkour.jar",
			resolve:    func(ctx context.Context) (source, error) { return githubSource(ctx, "A5H73Y", "Parkour") },
			manualHint: "install manually from https://github.com/A5H73Y/Parkour/releases",
		},
		{
			name:     "Multiverse-Core",
			filename: "Multiverse-Core.jar",
			resolve:  func(ctx context.Context) (source, error) { return hangarSource(ctx, "Multiverse-Core") },
		},
		{
			name:     "WorldEdit",
			filename: "WorldEdit.jar",
			resolve:  func(ctx context.Context) (source, error) { return hangarSource(ctx, "WorldEdit") },
		},
	}
}

// InstallAll downloads all default plugins for a Paper server.
func InstallAll(ctx context.Context, serverDir string, output *ui.UI) error {
	pluginsDir := filepath.Join(serverDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		return fmt.Errorf("creating plugins dir: %w", err)
	}

	for _, r := range defaultResolvers() {
		installPlugin(ctx, r, pluginsDir, output)
	}

	output.Success("Plugin installation complete")
	return nil
}

// installPlugin resolves and downloads one plugin. Failures are reported but
// never fatal: a missing optional plugin should not abort an install.
func installPlugin(ctx context.Context, r resolver, pluginsDir string, output *ui.UI) {
	dest := filepath.Join(pluginsDir, r.filename)
	if _, err := os.Stat(dest); err == nil {
		output.Success("%s already downloaded", r.name)
		return
	}

	src, err := r.resolve(ctx)
	if err != nil {
		hint := r.manualHint
		if hint == "" {
			hint = "install it manually"
		}
		output.Warn("Could not resolve %s download: %v — %s", r.name, err, hint)
		return
	}

	// Verification is best-effort in the sense that a missing digest does not
	// block the install — upstream APIs change shape, and refusing to install
	// would be a worse failure than today's behaviour. A digest that is
	// present and wrong is always fatal for that plugin.
	if src.expected.Empty() {
		output.Warn("%s publishes no checksum — downloading unverified", r.name)
	}

	output.Info("Downloading %s...", r.name)
	if err := downloadVerified(ctx, src, dest); err != nil {
		output.Warn("Failed to download %s: %v — you can install it manually", r.name, err)
		return
	}

	if src.expected.Empty() {
		output.Success("%s downloaded", r.name)
		return
	}
	output.Success("%s downloaded (%s verified)", r.name, src.expected.Describe())
}

// downloadVerified fetches src into dest. The body is written to a temporary
// file in the destination directory and verified before being renamed into
// place, so a corrupt or tampered download never lands at dest — installPlugin
// treats any existing file as a completed download, so a bad one would
// otherwise be permanently mistaken for a good plugin.
func downloadVerified(ctx context.Context, src source, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src.url, http.NoBody)
	if err != nil {
		return err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, src.url)
	}

	f, err := os.CreateTemp(filepath.Dir(dest), "."+filepath.Base(dest)+".*.part")
	if err != nil {
		return err
	}
	tmpPath := f.Name()
	defer func() {
		_ = f.Close()
		_ = os.Remove(tmpPath) // no-op once the rename below succeeds
	}()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := verify.File(tmpPath, src.expected); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		return err
	}

	return os.Rename(tmpPath, dest)
}

// getJSON fetches url and decodes the JSON body into v.
func getJSON(ctx context.Context, url string, v any) error {
	body, err := get(ctx, url)
	if err != nil {
		return err
	}
	return decodeJSON(body, v)
}

// get fetches url and returns the response body.
func get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	return io.ReadAll(resp.Body)
}

// decodeJSON unmarshals body into v with a clearer error than the raw one.
func decodeJSON(body []byte, v any) error {
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}
	return nil
}
