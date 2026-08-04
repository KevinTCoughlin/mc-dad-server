package server

import (
	"crypto/sha1" //nolint:gosec // SHA-1 is the only digest Mojang publishes for vanilla JARs.
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"strings"
)

// Artifact is a downloadable server JAR together with whatever checksum the
// upstream API publishes for it. The Containerfile verifies every artifact it
// pulls; the bare-metal installer verifies the same way so both paths reject
// a JAR that does not match the digest the API advertised.
type Artifact struct {
	URL string

	// SHA256 is published by the PaperMC Fill v3 API.
	SHA256 string

	// SHA1 is the only digest Mojang publishes for vanilla server JARs.
	SHA1 string
}

// Verify checks the file at path against the artifact's published checksum.
// An artifact with no checksum verifies trivially — the upstream API did not
// provide one, so there is nothing to compare against.
func (a Artifact) Verify(path string) error {
	var (
		h        hash.Hash
		expected string
		alg      string
	)

	switch {
	case a.SHA256 != "":
		h, expected, alg = sha256.New(), a.SHA256, "sha256"
	case a.SHA1 != "":
		h, expected, alg = sha1.New(), a.SHA1, "sha1" //nolint:gosec // upstream-published digest
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
