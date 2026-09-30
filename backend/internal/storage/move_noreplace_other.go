//go:build !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd

package storage

import (
	"os"
)

func renameStorageNoReplace(*os.Root, string, *os.Root, string) error {
	return errMoveNoReplaceUnsupported
}
