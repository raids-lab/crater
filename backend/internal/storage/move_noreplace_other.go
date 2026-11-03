//go:build !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd

package storage

import (
	"errors"
	"os"
)

func renameStorageEntry(*os.Root, string, *os.Root, string) error {
	return errors.New("FD-relative move is unsupported on this platform")
}
