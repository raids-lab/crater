//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package storage

import (
	"errors"
	"os"
	"sync"

	"golang.org/x/sys/unix"
)

// This lock serializes compatibility moves in this process only. Other storage
// replicas, WebDAV writes, and jobs writing directly to the PVC do not take it.
var compatibilityMoveMutex sync.Mutex

func renameStorageEntryWithFallback(
	sourceParent *os.Root,
	sourceName string,
	destinationParent *os.Root,
	destinationName string,
	noReplace func(int, string, int, string) error,
) error {
	sourceDirectory, destinationDirectory, err := openUploadDirectoryHandles(sourceParent, destinationParent)
	if err != nil {
		return err
	}
	defer sourceDirectory.Close()
	defer destinationDirectory.Close()
	sourceFD := int(sourceDirectory.Fd())
	destinationFD := int(destinationDirectory.Fd())
	err = noReplace(sourceFD, sourceName, destinationFD, destinationName)
	if !errors.Is(err, unix.EINVAL) &&
		!errors.Is(err, unix.ENOSYS) &&
		!errors.Is(err, unix.EOPNOTSUPP) {
		return err
	}

	// NFS and older kernel CephFS reject no-replace flags. Preserve their legacy
	// rename support, but do not promise atomic no-clobber for this fallback:
	// another writer can create the destination after this final check.
	compatibilityMoveMutex.Lock()
	defer compatibilityMoveMutex.Unlock()
	var destination unix.Stat_t
	if err := fstatatRetry(destinationFD, destinationName, &destination); err == nil {
		return errMoveTargetExists
	} else if !errors.Is(err, unix.ENOENT) {
		return err
	}
	return unix.Renameat(sourceFD, sourceName, destinationFD, destinationName)
}
