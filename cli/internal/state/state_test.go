package state

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNewManagerDoesNotCreateConfigDirWhenStateMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	m, err := NewManager()
	if err != nil {
		t.Fatal(err)
	}

	configHome, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	wantPath := filepath.Join(configHome, "crater", "state.json")
	if m.Path != wantPath {
		t.Fatalf("Path = %q, want %q", m.Path, wantPath)
	}
	if _, err := os.Stat(filepath.Dir(wantPath)); !os.IsNotExist(err) {
		t.Fatalf("config dir should not be created during NewManager, stat err = %v", err)
	}
}

func TestManagerSaveCreatesConfigDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	m, err := NewManager()
	if err != nil {
		t.Fatal(err)
	}
	m.State = State{
		AuthInfos: []AuthInfo{{Username: "alice", Token: "secret-token"}},
		Language:  "zh-CN",
	}

	if err := m.Save(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(m.Path); err != nil {
		t.Fatalf("state file should exist after Save: %v", err)
	}
}

func TestManagerSaveRestrictsExistingStateFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not expose POSIX file permission bits")
	}

	home := t.TempDir()
	t.Setenv("HOME", home)

	m, err := NewManager()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(m.Path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(m.Path, []byte(`{"language":"en"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(m.Path, 0644); err != nil {
		t.Fatal(err)
	}

	m.State.Language = "zh-CN"
	if err := m.Save(); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(m.Path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("state file permissions = %04o, want 0600", got)
	}
}
