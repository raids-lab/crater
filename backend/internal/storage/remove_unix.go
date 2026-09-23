//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package storage

import (
	"errors"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

func removeStorageNonDirectory(parent *os.Root, name string) error {
	parentDirectory, err := parent.Open(".")
	if err != nil {
		return err
	}
	defer parentDirectory.Close()

	return unlinkatRetry(int(parentDirectory.Fd()), name, 0)
}

func removeStorageDirectoryRecursive(parent *os.Root, name string) error {
	parentDirectory, err := parent.Open(".")
	if err != nil {
		return err
	}
	defer parentDirectory.Close()

	var parentStat unix.Stat_t
	if err := fstatRetry(int(parentDirectory.Fd()), &parentStat); err != nil {
		return err
	}

	targetDirectory, err := openDirectoryAt(int(parentDirectory.Fd()), name)
	if err != nil {
		return classifyRecursiveOpenError(err)
	}
	target := os.NewFile(uintptr(targetDirectory), name)
	if target == nil {
		_ = unix.Close(targetDirectory)
		return errors.New("failed to create directory handle")
	}

	var targetStat unix.Stat_t
	if err := fstatRetry(targetDirectory, &targetStat); err != nil {
		_ = target.Close()
		return err
	}
	if err := requireRecursiveStorageDevice(&parentStat, &targetStat); err != nil {
		_ = target.Close()
		return err
	}

	if err := removeDirectoryContents(target, &parentStat); err != nil {
		_ = target.Close()
		return err
	}
	if err := verifyDirectoryIdentityAt(int(parentDirectory.Fd()), name, &targetStat); err != nil {
		_ = target.Close()
		return err
	}
	if err := target.Close(); err != nil {
		return err
	}
	if err := unlinkatRetry(int(parentDirectory.Fd()), name, unix.AT_REMOVEDIR); err != nil {
		return classifyRecursiveUnlinkError(err)
	}
	return nil
}

func removeDirectoryContents(directory *os.File, rootDevice *unix.Stat_t) error {
	names, err := directory.Readdirnames(-1)
	if err != nil {
		return err
	}
	directoryFD := int(directory.Fd())
	for _, name := range names {
		if name == "" || name == "." || name == parentPathSegment {
			return errRemoveTargetChanged
		}

		var entryStat unix.Stat_t
		err := fstatatRetry(directoryFD, name, &entryStat, unix.AT_SYMLINK_NOFOLLOW)
		if errors.Is(err, unix.ENOENT) {
			continue
		}
		if err != nil {
			return err
		}
		if err := requireRecursiveStorageDevice(rootDevice, &entryStat); err != nil {
			return err
		}

		if entryStat.Mode&unix.S_IFMT == unix.S_IFDIR {
			if err := removeDirectoryEntry(directoryFD, name, rootDevice); errors.Is(err, errRemoveTargetNotFound) {
				continue
			} else if err != nil {
				return err
			}
			continue
		}
		if err := unlinkatRetry(directoryFD, name, 0); err != nil {
			switch {
			case errors.Is(err, unix.ENOENT):
				continue
			case errors.Is(err, unix.EISDIR), errors.Is(err, unix.EPERM):
				return errRemoveTargetChanged
			default:
				return err
			}
		}
	}
	return nil
}

func removeDirectoryEntry(parentFD int, name string, rootDevice *unix.Stat_t) error {
	childFD, err := openDirectoryAt(parentFD, name)
	if err != nil {
		return classifyRecursiveOpenError(err)
	}
	child := os.NewFile(uintptr(childFD), name)
	if child == nil {
		_ = unix.Close(childFD)
		return errors.New("failed to create directory handle")
	}

	var childStat unix.Stat_t
	if err := fstatRetry(childFD, &childStat); err != nil {
		_ = child.Close()
		return err
	}
	if err := requireRecursiveStorageDevice(rootDevice, &childStat); err != nil {
		_ = child.Close()
		return err
	}
	if err := removeDirectoryContents(child, rootDevice); err != nil {
		_ = child.Close()
		return err
	}
	if err := verifyDirectoryIdentityAt(parentFD, name, &childStat); err != nil {
		_ = child.Close()
		return err
	}
	if err := child.Close(); err != nil {
		return err
	}
	if err := unlinkatRetry(parentFD, name, unix.AT_REMOVEDIR); err != nil {
		return classifyRecursiveUnlinkError(err)
	}
	return nil
}

func openDirectoryAt(parentFD int, name string) (int, error) {
	for {
		directory, err := unix.Openat(
			parentFD,
			name,
			unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC,
			0,
		)
		if !errors.Is(err, unix.EINTR) {
			return directory, err
		}
	}
}

func fstatRetry(fd int, stat *unix.Stat_t) error {
	for {
		err := unix.Fstat(fd, stat)
		if !errors.Is(err, unix.EINTR) {
			return err
		}
	}
}

func fstatatRetry(fd int, name string, stat *unix.Stat_t, flags int) error {
	for {
		err := unix.Fstatat(fd, name, stat, flags)
		if !errors.Is(err, unix.EINTR) {
			return err
		}
	}
}

func unlinkatRetry(fd int, name string, flags int) error {
	for {
		err := unix.Unlinkat(fd, name, flags)
		if !errors.Is(err, unix.EINTR) {
			return err
		}
	}
}

func verifyDirectoryIdentityAt(parentFD int, name string, opened *unix.Stat_t) error {
	var current unix.Stat_t
	if err := fstatatRetry(parentFD, name, &current, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		if errors.Is(err, unix.ENOENT) {
			return errRemoveTargetNotFound
		}
		return err
	}
	if current.Mode&unix.S_IFMT != unix.S_IFDIR ||
		current.Dev != opened.Dev ||
		current.Ino != opened.Ino {
		return errRemoveTargetChanged
	}
	return nil
}

func requireRecursiveStorageDevice(rootDevice, entryDevice *unix.Stat_t) error {
	if rootDevice == nil || entryDevice == nil || entryDevice.Dev != rootDevice.Dev {
		return errRemoveCrossDevice
	}
	return nil
}

func classifyRecursiveOpenError(err error) error {
	switch {
	case errors.Is(err, unix.ENOENT):
		return errRemoveTargetNotFound
	case errors.Is(err, unix.ENOTDIR), errors.Is(err, unix.ELOOP):
		return errRemoveTargetChanged
	default:
		return err
	}
}

func classifyRecursiveUnlinkError(err error) error {
	switch {
	case errors.Is(err, unix.ENOENT):
		return errRemoveTargetNotFound
	case errors.Is(err, unix.ENOTEMPTY),
		errors.Is(err, unix.EEXIST),
		errors.Is(err, unix.ENOTDIR):
		return errRemoveTargetChanged
	case errors.Is(err, unix.EBUSY), errors.Is(err, syscall.EXDEV):
		return errRemoveCrossDevice
	default:
		return err
	}
}
