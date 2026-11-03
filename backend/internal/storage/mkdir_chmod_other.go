//go:build !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd

package storage

import (
	"errors"
	"os"
)

func chmodCreatedStorageDirectory(parent *os.Root, name string, mode os.FileMode) error {
	created, err := parent.Open(name)
	if err != nil {
		return err
	}
	defer created.Close()
	info, err := created.Stat()
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("created storage entry is no longer a directory")
	}
	return created.Chmod(mode)
}
