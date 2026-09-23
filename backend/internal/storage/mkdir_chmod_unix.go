//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package storage

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

func chmodCreatedStorageDirectory(parent *os.Root, name string, mode os.FileMode) error {
	parentDirectory, err := parent.Open(".")
	if err != nil {
		return err
	}
	defer parentDirectory.Close()

	var directory int
	for {
		directory, err = unix.Openat(
			int(parentDirectory.Fd()),
			name,
			unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC,
			0,
		)
		if !errors.Is(err, unix.EINTR) {
			break
		}
	}
	if err != nil {
		return err
	}
	defer unix.Close(directory)

	for {
		err = unix.Fchmod(directory, uint32(mode.Perm()))
		if !errors.Is(err, unix.EINTR) {
			return err
		}
	}
}
