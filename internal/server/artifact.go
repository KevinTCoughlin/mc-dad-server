package server

import "github.com/KevinTCoughlin/mc-dad-server/internal/verify"

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

// Expected returns the integrity metadata to check a download against.
func (a Artifact) Expected() verify.Expected {
	return verify.Expected{SHA256: a.SHA256, SHA1: a.SHA1}
}

// Verify checks the file at path against the artifact's published checksum.
// An artifact with no checksum verifies trivially — the upstream API did not
// provide one, so there is nothing to compare against.
func (a Artifact) Verify(path string) error {
	return verify.File(path, a.Expected())
}
