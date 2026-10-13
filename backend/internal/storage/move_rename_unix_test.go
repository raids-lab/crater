//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"golang.org/x/sys/unix"
)

func TestMoveCompatibilityFallbackMovesFilesAndDirectories(t *testing.T) {
	for _, unsupported := range []error{unix.EINVAL, unix.ENOSYS, unix.EOPNOTSUPP} {
		for _, directory := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/directory=%t", unsupported, directory), func(t *testing.T) {
				storage := t.TempDir()
				source := filepath.Join(storage, "source")
				if directory {
					makeRemoveIdentityTestDirectory(t, source, "payload")
				} else if err := os.WriteFile(source, []byte("payload"), 0o600); err != nil {
					t.Fatal(err)
				}
				root, err := os.OpenRoot(storage)
				if err != nil {
					t.Fatal(err)
				}
				defer root.Close()
				err = renameStorageEntryWithFallback(root, "source", root, "destination",
					func(int, string, int, string) error { return unsupported })
				if err != nil {
					t.Fatal(err)
				}
				if _, err := root.Lstat("source"); !os.IsNotExist(err) {
					t.Fatalf("source still exists: %v", err)
				}
				destination := filepath.Join(storage, "destination")
				if directory {
					destination = filepath.Join(destination, "keep.txt")
				}
				assertStoredFile(t, destination, []byte("payload"))
			})
		}
	}
}

func TestMoveCompatibilityFallbackRechecksDestination(t *testing.T) {
	for _, kind := range []string{"file", "directory", "dangling symlink"} {
		t.Run(kind, func(t *testing.T) {
			storage := t.TempDir()
			if err := os.WriteFile(filepath.Join(storage, "source"), []byte("source"), 0o600); err != nil {
				t.Fatal(err)
			}
			root, err := os.OpenRoot(storage)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			var expected os.FileInfo
			unsupported := func(int, string, int, string) error {
				destination := filepath.Join(storage, "destination")
				switch kind {
				case "file":
					err = os.WriteFile(destination, []byte("existing"), 0o600)
				case "directory":
					err = os.Mkdir(destination, 0o700)
				case "dangling symlink":
					err = os.Symlink("missing", destination)
				}
				if err != nil {
					t.Fatal(err)
				}
				expected, err = root.Lstat("destination")
				if err != nil {
					t.Fatal(err)
				}
				return unix.EOPNOTSUPP
			}
			if err := renameStorageEntryWithFallback(root, "source", root, "destination", unsupported); !errors.Is(err, errMoveTargetExists) {
				t.Fatalf("error = %v, want errMoveTargetExists", err)
			}
			current, err := root.Lstat("destination")
			if err != nil || !os.SameFile(expected, current) {
				t.Fatalf("destination was replaced: %v", err)
			}
			assertStoredFile(t, filepath.Join(storage, "source"), []byte("source"))
			if kind == "file" {
				assertStoredFile(t, filepath.Join(storage, "destination"), []byte("existing"))
			}
		})
	}
}

func TestMoveCompatibilityDoesNotFallbackForOtherErrors(t *testing.T) {
	for _, failure := range []error{unix.EACCES, unix.EXDEV, unix.EIO, unix.EEXIST, unix.ENOENT} {
		t.Run(failure.Error(), func(t *testing.T) {
			storage := t.TempDir()
			if err := os.WriteFile(filepath.Join(storage, "source"), []byte("source"), 0o600); err != nil {
				t.Fatal(err)
			}
			root, err := os.OpenRoot(storage)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			err = renameStorageEntryWithFallback(root, "source", root, "destination",
				func(int, string, int, string) error { return failure })
			if !errors.Is(err, failure) {
				t.Fatalf("error = %v, want %v", err, failure)
			}
			assertStoredFile(t, filepath.Join(storage, "source"), []byte("source"))
			if _, err := root.Lstat("destination"); !os.IsNotExist(err) {
				t.Fatalf("destination unexpectedly created: %v", err)
			}
		})
	}
}

func TestMoveCompatibilityRaceHasOneWinnerWithinProcess(t *testing.T) {
	storage := t.TempDir()
	const contenders = 8
	root, err := os.OpenRoot(storage)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	for index := range contenders {
		name := fmt.Sprintf("source-%d", index)
		if err := os.WriteFile(filepath.Join(storage, name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	var ready, done sync.WaitGroup
	ready.Add(contenders)
	done.Add(contenders)
	results := make(chan error, contenders)
	for index := range contenders {
		go func() {
			defer done.Done()
			results <- renameStorageEntryWithFallback(root, fmt.Sprintf("source-%d", index), root, "destination",
				func(int, string, int, string) error {
					ready.Done()
					ready.Wait()
					return unix.EOPNOTSUPP
				})
		}()
	}
	done.Wait()
	close(results)
	successes, conflicts := 0, 0
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, errMoveTargetExists):
			conflicts++
		default:
			t.Fatalf("unexpected move error: %v", err)
		}
	}
	if successes != 1 || conflicts != contenders-1 {
		t.Fatalf("successes=%d conflicts=%d", successes, conflicts)
	}
	winner, err := os.ReadFile(filepath.Join(storage, "destination"))
	if err != nil {
		t.Fatal(err)
	}
	for index := range contenders {
		name := fmt.Sprintf("source-%d", index)
		if name == string(winner) {
			if _, err := root.Lstat(name); !os.IsNotExist(err) {
				t.Fatalf("winning source still exists: %v", err)
			}
		} else {
			assertStoredFile(t, filepath.Join(storage, name), []byte(name))
		}
	}
}
