package configs

import "io/fs"

// embeddedFS is set from the cmd package which has the go:embed directive.
var embeddedFS fs.FS

// SetEmbeddedFS sets the embedded filesystem. Must be called before Deploy.
func SetEmbeddedFS(fsys fs.FS) {
	embeddedFS = fsys
}
