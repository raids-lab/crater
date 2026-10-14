//go:build !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd

package storage

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveStorageEntryFailsClosedOnUnsupportedPlatform(t *testing.T) {
	storage := t.TempDir()
	path := filepath.Join(storage, "file")
	if err := os.WriteFile(path, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(storage)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	if err := removeStorageEntry(root, "file", false); !errors.Is(err, errRemoveOperationUnsupported) {
		t.Fatalf("error = %v, want errRemoveOperationUnsupported", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file changed on unsupported platform: %v", err)
	}
}
