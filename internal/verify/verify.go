// Package verify checks downloaded files against the integrity metadata that
// upstream APIs publish for them.
//
// Different upstreams publish different things: PaperMC gives a SHA-256,
// Mojang a SHA-1, and the GitHub releases API no digest at all — only a file
// size. Expected carries whatever is available and File checks every field
// that was populated.
package verify

import (
	"crypto/sha1" //nolint:gosec // SHA-1 is the only digest Mojang publishes.
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"strings"
)

// Expected is the integrity metadata published for a file. A zero value
// carries nothing to check.
type Expected struct {
	// SHA256 is published by PaperMC Fill v3, the GeyserMC builds API, and
	// Hangar.
	SHA256 string

	// SHA1 is the only digest Mojang publishes for vanilla server JARs.
	SHA1 string

	// Size is the file size in bytes. Zero means unknown. It is a weak check
	// used only where no digest is published (GitHub releases), matching what
	// the Containerfile does for the Parkour plugin.
	Size int64
}

// Empty reports whether there is nothing to verify against.
func (e Expected) Empty() bool {
	return e.SHA256 == "" && e.SHA1 == "" && e.Size == 0
}

// Describe returns a short human-readable summary of what will be checked.
func (e Expected) Describe() string {
	switch {
	case e.SHA256 != "":
		return "sha256"
	case e.SHA1 != "":
		return "sha1"
	case e.Size > 0:
		return "size"
	default:
		return "nothing"
	}
}

// File checks path against every populated field of e. An empty Expected
// verifies trivially — the upstream published nothing to compare against.
func File(path string, e Expected) error {
	if e.Empty() {
		return nil
	}

	if e.Size > 0 {
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if info.Size() != e.Size {
			return fmt.Errorf("size mismatch for %s: got %d bytes, want %d", path, info.Size(), e.Size)
		}
	}

	var (
		h        hash.Hash
		expected string
		alg      string
	)
	switch {
	case e.SHA256 != "":
		h, expected, alg = sha256.New(), e.SHA256, "sha256"
	case e.SHA1 != "":
		h, expected, alg = sha1.New(), e.SHA1, "sha1" //nolint:gosec // upstream-published digest
	default:
		return nil
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("hashing %s: %w", path, err)
	}

	actual := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("%s checksum mismatch for %s: got %s, want %s", alg, path, actual, expected)
	}

	return nil
}
