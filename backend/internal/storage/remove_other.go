//go:build !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd

package storage

import "os"

func removeStorageNonDirectory(*os.Root, string) error {
	return errRemoveOperationUnsupported
}

func removeStorageDirectoryRecursive(*os.Root, string) error {
	return errRemoveOperationUnsupported
}
