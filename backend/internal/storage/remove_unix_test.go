//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package storage

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

func TestRemoveStorageEntryUnlinksFinalSymlinkWithoutFollowingIt(t *testing.T) {
	storage := t.TempDir()
	outside := t.TempDir()
	outsideFile := filepath.Join(outside, "keep.txt")
	if err := os.WriteFile(outsideFile, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(storage, "link")
	if err := os.Symlink(outsideFile, link); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(storage)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	if err := removeStorageEntry(root, "link", false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Fatalf("symlink still exists: %v", err)
	}
	content, err := os.ReadFile(outsideFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "keep" {
		t.Fatalf("outside target changed: %q", content)
	}
}

func TestRecursiveRemoveDoesNotFollowNestedSymlink(t *testing.T) {
	storage := t.TempDir()
	outside := t.TempDir()
	outsideFile := filepath.Join(outside, "keep.txt")
	if err := os.WriteFile(outsideFile, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	tree := filepath.Join(storage, "tree")
	if err := os.Mkdir(tree, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tree, "inside.txt"), []byte("inside"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(tree, "outside-link")); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(storage)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	if err := removeStorageEntry(root, "tree", true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(tree); !os.IsNotExist(err) {
		t.Fatalf("tree still exists: %v", err)
	}
	content, err := os.ReadFile(outsideFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "keep" {
		t.Fatalf("outside target changed: %q", content)
	}
}

func TestRecursiveRemoveDeviceBoundaryCheckFailsClosed(t *testing.T) {
	first := unix.Stat_t{Dev: 1}
	same := unix.Stat_t{Dev: 1}
	different := unix.Stat_t{Dev: 2}
	if err := requireRecursiveStorageDevice(&first, &same); err != nil {
		t.Fatalf("same device rejected: %v", err)
	}
	if err := requireRecursiveStorageDevice(&first, &different); !errors.Is(err, errRemoveCrossDevice) {
		t.Fatalf("cross-device error = %v, want errRemoveCrossDevice", err)
	}
	if err := requireRecursiveStorageDevice(nil, &same); !errors.Is(err, errRemoveCrossDevice) {
		t.Fatalf("missing device error = %v, want errRemoveCrossDevice", err)
	}
}

func TestRecursiveRemoveDetectsDirectoryNameSubstitution(t *testing.T) {
	storage := t.TempDir()
	original := filepath.Join(storage, "target")
	if err := os.Mkdir(original, 0o700); err != nil {
		t.Fatal(err)
	}
	parent, err := os.Open(storage)
	if err != nil {
		t.Fatal(err)
	}
	defer parent.Close()
	childFD, err := openDirectoryAt(int(parent.Fd()), "target")
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(childFD)
	var opened unix.Stat_t
	if err := fstatRetry(childFD, &opened); err != nil {
		t.Fatal(err)
	}

	if err := os.Rename(original, filepath.Join(storage, "renamed")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(original, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := verifyDirectoryIdentityAt(int(parent.Fd()), "target", &opened); !errors.Is(err, errRemoveTargetChanged) {
		t.Fatalf("substitution error = %v, want errRemoveTargetChanged", err)
	}
}
