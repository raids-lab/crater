//go:build darwin

package storage

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

func renameStorageNoReplace(
	sourceParent *os.Root,
	sourceName string,
	destinationParent *os.Root,
	destinationName string,
) error {
	sourceDirectory, destinationDirectory, err := openUploadDirectoryHandles(sourceParent, destinationParent)
	if err != nil {
		return err
	}
	defer sourceDirectory.Close()
	defer destinationDirectory.Close()
	err = unix.RenameatxNp(
		int(sourceDirectory.Fd()),
		sourceName,
		int(destinationDirectory.Fd()),
		destinationName,
		unix.RENAME_EXCL,
	)
	if errors.Is(err, unix.EINVAL) ||
		errors.Is(err, unix.ENOSYS) ||
		errors.Is(err, unix.EOPNOTSUPP) {
		return errMoveNoReplaceUnsupported
	}
	return err
}
