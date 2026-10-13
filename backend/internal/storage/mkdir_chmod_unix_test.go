//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestChmodCreatedStorageDirectoryDoesNotFollowSymlink(t *testing.T) {
	storage := t.TempDir()
	target := filepath.Join(storage, "target")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("target", filepath.Join(storage, "created")); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(storage)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	if err := chmodCreatedStorageDirectory(root, "created", 0o777); err == nil {
		t.Fatal("chmod through a symlink unexpectedly succeeded")
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("symlink target mode = %v, want 0700", info.Mode().Perm())
	}
}
