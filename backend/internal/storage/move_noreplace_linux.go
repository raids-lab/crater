//go:build linux

package storage

import (
	"os"

	"golang.org/x/sys/unix"
)

func renameStorageEntry(
	sourceParent *os.Root,
	sourceName string,
	destinationParent *os.Root,
	destinationName string,
) error {
	return renameStorageEntryWithFallback(sourceParent, sourceName, destinationParent, destinationName,
		func(sourceFD int, source string, destinationFD int, destination string) error {
			return unix.Renameat2(sourceFD, source, destinationFD, destination, unix.RENAME_NOREPLACE)
		})
}
