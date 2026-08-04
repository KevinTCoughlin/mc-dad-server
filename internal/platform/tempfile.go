package platform

import (
	"fmt"
	"os"
)

// writePrivateTemp writes data to a freshly created temp file and returns its
// path. Unlike a fixed name under os.TempDir(), os.CreateTemp fails rather
// than following a pre-existing symlink, so a local user cannot redirect the
// write or swap the contents before the file is consumed. Callers are
// responsible for removing the file.
//
// pattern follows the os.CreateTemp convention ("name-*.ext").
func writePrivateTemp(pattern string, data []byte, mode os.FileMode) (string, error) {
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", err
	}
	path := f.Name()

	cleanup := func(cause error) (string, error) {
		_ = f.Close()
		_ = os.Remove(path)
		return "", cause
	}

	if err := f.Chmod(mode); err != nil {
		return cleanup(fmt.Errorf("setting temp file mode: %w", err))
	}
	if _, err := f.Write(data); err != nil {
		return cleanup(fmt.Errorf("writing temp file: %w", err))
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("closing temp file: %w", err)
	}

	return path, nil
}
