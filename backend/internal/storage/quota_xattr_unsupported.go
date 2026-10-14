//go:build !linux

package storage

import (
	"fmt"
	"path/filepath"

	"github.com/raids-lab/crater/pkg/storagequota"
)

//nolint:gocritic // The tuple mirrors the Linux implementation's validated root and target paths.
func secureStoragePaths(rootPath, targetPath string) (string, string, error) {
	root, err := filepath.Abs(rootPath)
	if err != nil {
		return "", "", err
	}
	target, err := filepath.Abs(targetPath)
	if err != nil {
		return "", "", err
	}
	return root, target, nil
}

func inspectXattrCapabilities(_ string) storagequota.Capabilities {
	return storagequota.Capabilities{
		Reasons: []string{"CephFS quota xattrs are only supported by storage-server on Linux"},
	}
}

func readXattrInt64(_, name string) (int64, error) {
	return 0, fmt.Errorf("reading xattr %s is not supported on this operating system", name)
}

func writeXattr(_, name string, _ []byte) error {
	return fmt.Errorf("writing xattr %s is not supported on this operating system", name)
}
