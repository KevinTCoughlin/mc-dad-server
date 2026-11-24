package platform

import "fmt"

// MinInstallDisk is the free space required before downloading Paper, plugins,
// and generating a new world.
const MinInstallDisk = 2 * 1024 * 1024 * 1024

// CheckDiskSpace reports a clear error when path has less than minBytes free.
func CheckDiskSpace(path string, minBytes uint64) error {
	free, err := freeBytes(path)
	if err != nil {
		return fmt.Errorf("could not check free disk space in %s: %w. Choose another directory with --dir", path, err)
	}
	if free < minBytes {
		return fmt.Errorf("not enough free disk space in %s: %s free, need at least %s for the server JAR, plugins, and a new world. Free space or pass --dir pointing at a larger filesystem", path, FormatBytes(free), FormatBytes(minBytes))
	}
	return nil
}
