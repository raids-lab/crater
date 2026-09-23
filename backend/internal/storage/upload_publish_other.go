//go:build !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd

package storage

import (
	"errors"
	"os"
)

var errUploadPublishUnsupported = errors.New("atomic upload publishing is unsupported on this platform")

func publishUploadNoClobber(*os.Root, *os.Root, string) error {
	return errUploadPublishUnsupported
}

func publishUploadOverwrite(*os.Root, *os.Root, string) error {
	return errUploadPublishUnsupported
}
