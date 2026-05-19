package state

import (
	"os"
	"path/filepath"
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
	m.State.Language = "zh-CN"

	if err := m.Save(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(m.Path); err != nil {
		t.Fatalf("state file should exist after Save: %v", err)
	}
}
