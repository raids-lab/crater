//go:build dragonfly || freebsd || netbsd || openbsd

package storage

import "os"

func renameStorageNoReplace(
	sourceParent *os.Root,
	sourceName string,
	destinationParent *os.Root,
	destinationName string,
) error {
	return errMoveNoReplaceUnsupported
}
