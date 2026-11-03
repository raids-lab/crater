//go:build dragonfly || freebsd || netbsd || openbsd

package storage

import (
	"errors"
	"os"
)

func renameStorageEntry(
	sourceParent *os.Root,
	sourceName string,
	destinationParent *os.Root,
	destinationName string,
) error {
	return errors.New("FD-relative move is unsupported on this platform")
}
