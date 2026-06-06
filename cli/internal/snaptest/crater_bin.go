package snaptest

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

var craterBin struct {
	path string
	once sync.Once
	err  error
}

// CraterExecutable builds the crater CLI once per test process and returns the binary path.
func CraterExecutable(t *testing.T) string {
	t.Helper()
	craterBin.once.Do(func() {
		root, err := ModuleRoot()
		if err != nil {
			craterBin.err = err
			return
		}
		dir, err := os.MkdirTemp("", "crater-snapshot-*")
		if err != nil {
			craterBin.err = fmt.Errorf("snaptest: create temp dir: %w", err)
			return
		}
		bin := filepath.Join(dir, "crater")
		if runtime.GOOS == "windows" {
			bin += ".exe"
		}
		cmd := exec.Command("go", "build", "-C", root, "-o", bin, ".")
		out, err := cmd.CombinedOutput()
		if err != nil {
			craterBin.err = fmt.Errorf("snaptest: go build -C %q: %w\n%s", root, err, out)
			return
		}
		craterBin.path = bin
	})
	if craterBin.err != nil {
		t.Fatal(craterBin.err)
	}
	return craterBin.path
}
