//go:build darwin || linux

package state

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

func lockStateFile(file *os.File) error {
	for {
		err := unix.Flock(int(file.Fd()), unix.LOCK_EX)
		if !errors.Is(err, unix.EINTR) {
			return err
		}
	}
}
