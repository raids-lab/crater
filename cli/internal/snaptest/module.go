package snaptest

import (
	"fmt"
	"os"
	"path/filepath"
)

// ModuleRoot returns the directory containing go.mod (CLI module root).
// When running under `go test`, the working directory is normally the module root.
func ModuleRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
		return wd, nil
	}
	for d := wd; d != filepath.Dir(d); d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			return d, nil
		}
	}
	return "", fmt.Errorf("snaptest: go.mod not found from %s", wd)
}
