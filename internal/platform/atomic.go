package platform

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// WriteAtomic replaces path with data, leaving any existing file untouched
// until the replacement is complete. The temporary file is created in the
// same directory so the final rename stays on one filesystem, then synced
// and renamed over path. mode is applied before the rename, so a crash
// cannot publish a world-writable file.
func WriteAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	f, err := os.CreateTemp(dir, ".tmp-"+filepath.Base(path)+"-*")
	if err != nil {
		return fmt.Errorf("creating temporary file in %s: %w", dir, err)
	}
	tmp := f.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = f.Close()
			_ = os.Remove(tmp)
		}
	}()

	if err := f.Chmod(mode); err != nil {
		return fmt.Errorf("setting mode on %s: %w", tmp, err)
	}
	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("writing %s: %w", tmp, err)
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("syncing %s: %w", tmp, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("closing %s: %w", tmp, err)
	}
	if err := renameOverwrite(tmp, path); err != nil {
		return fmt.Errorf("replacing %s: %w", path, err)
	}
	cleanup = false
	return nil
}

// renameOverwrite renames oldpath to newpath, replacing newpath when it
// already exists. os.Rename replaces on POSIX and fails on Windows when the
// destination exists, so the destination is moved aside first and restored
// if the new name does not land.
func renameOverwrite(oldpath, newpath string) error {
	backupPath := newpath + ".bak-rename"
	if err := os.Remove(backupPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing stale backup %s: %w", backupPath, err)
	}

	hasBackup := false
	if err := os.Rename(newpath, backupPath); err == nil {
		hasBackup = true
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("backing up existing %s: %w", newpath, err)
	}

	if err := os.Rename(oldpath, newpath); err != nil {
		if hasBackup {
			if restoreErr := os.Rename(backupPath, newpath); restoreErr != nil {
				return errors.Join(err, fmt.Errorf("restoring %s from backup %s: %w", newpath, backupPath, restoreErr))
			}
		}
		return err
	}
	if hasBackup {
		_ = os.Remove(backupPath)
	}
	return nil
}
