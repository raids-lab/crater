package state

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/raids-lab/crater/cli/internal/testutil"
)

func TestNewManagerDoesNotCreateConfigDirWhenStateMissing(t *testing.T) {
	configHome := testutil.IsolateUserConfigDir(t)

	m, err := NewManager()
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
	testutil.IsolateUserConfigDir(t)

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

	testutil.IsolateUserConfigDir(t)

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
