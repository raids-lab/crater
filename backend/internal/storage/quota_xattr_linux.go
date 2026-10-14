//go:build linux

package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"

	"github.com/raids-lab/crater/pkg/storagequota"
)

func inspectXattrCapabilities(rootPath string) storagequota.Capabilities {
	capabilities := storagequota.Capabilities{}
	if _, err := readXattrInt64(rootPath, cephDirectoryBytesXattr); err != nil {
		capabilities.Reasons = append(capabilities.Reasons, err.Error())
		return capabilities
	}

	capabilities.UsageReadable = true
	_, err := readXattr(rootPath, cephQuotaMaxBytesXattr)
	if err != nil {
		if !errors.Is(err, unix.ENODATA) {
			capabilities.Reasons = append(capabilities.Reasons, err.Error())
			return capabilities
		}
	}

	capabilities.QuotaReadable = true
	testDir, err := os.MkdirTemp(rootPath, ".cephfs-quota-capability-")
	if err != nil {
		capabilities.Reasons = append(capabilities.Reasons, fmt.Sprintf("create quota capability test directory: %v", err))
		return capabilities
	}
	defer func() {
		_ = os.Remove(testDir)
	}()

	// A zero quota is unlimited, so the probe verifies the CephX MDS "p"
	// capability without restricting the temporary directory.
	if err := writeXattr(testDir, cephQuotaMaxBytesXattr, []byte("0")); err != nil {
		capabilities.Reasons = append(capabilities.Reasons, err.Error())
		return capabilities
	}
	capabilities.QuotaWritable = true
	return capabilities
}

//nolint:gocritic // The tuple returns the validated root and target paths used by the caller.
func secureStoragePaths(rootPath, targetPath string) (string, string, error) {
	root, err := filepath.EvalSymlinks(rootPath)
	if err != nil {
		return "", "", err
	}
	target, err := filepath.EvalSymlinks(targetPath)
	if err != nil {
		return "", "", err
	}
	return root, target, nil
}

func readXattrInt64(targetPath, name string) (int64, error) {
	value, err := readXattr(targetPath, name)
	if err != nil {
		if errors.Is(err, unix.ENODATA) && name == cephQuotaMaxBytesXattr {
			return 0, nil
		}
		return 0, err
	}
	parsed, err := strconv.ParseInt(strings.TrimSpace(strings.Trim(string(value), "\x00\"")), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse xattr %s on %s: %w", name, targetPath, err)
	}
	return parsed, nil
}

func readXattr(targetPath, name string) ([]byte, error) {
	size, err := unix.Getxattr(targetPath, name, nil)
	if err != nil {
		return nil, fmt.Errorf("read xattr %s on %s: %w", name, targetPath, err)
	}
	buffer := make([]byte, size)
	read, err := unix.Getxattr(targetPath, name, buffer)
	if err != nil {
		return nil, fmt.Errorf("read xattr %s on %s: %w", name, targetPath, err)
	}
	return buffer[:read], nil
}

func writeXattr(targetPath, name string, value []byte) error {
	if err := unix.Setxattr(targetPath, name, value, 0); err != nil {
		return fmt.Errorf("write xattr %s on %s: %w", name, targetPath, err)
	}
	return nil
}
