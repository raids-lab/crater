package storage

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveStorageDirectory(t *testing.T) {
	root := t.TempDir()
	originalRoot := storageRootDir
	storageRootDir = root
	t.Cleanup(func() { storageRootDir = originalRoot })

	validDirectory := filepath.Join(root, "users", "alice")
	if err := os.MkdirAll(validDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "users", "file.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path    string
		wantRel string
		wantErr error
	}{
		{name: "valid", path: "users/alice", wantRel: "users/alice"},
		{name: "absolute", path: "/users/alice", wantErr: errInvalidStoragePath},
		{name: "traversal", path: "../outside", wantErr: errInvalidStoragePath},
		{name: "file", path: "users/file.txt", wantErr: errStoragePathNotDirectory},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, relative, err := resolveStorageDirectory(tt.path)
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if relative != tt.wantRel {
				t.Fatalf("relative path = %q, want %q", relative, tt.wantRel)
			}
		})
	}
}

func TestResolveStorageUsagePathAllowsRoot(t *testing.T) {
	root := t.TempDir()
	originalRoot := storageRootDir
	storageRootDir = root
	t.Cleanup(func() { storageRootDir = originalRoot })

	target, relative, err := resolveStorageUsagePath(".")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantTarget, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	if target != wantTarget {
		t.Fatalf("target path = %q, want %q", target, wantTarget)
	}
	if relative != "." {
		t.Fatalf("relative path = %q, want %q", relative, ".")
	}
}
