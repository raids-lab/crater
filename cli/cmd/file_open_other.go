//go:build !aix && !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !solaris

package cmd

import "os"

func openUploadFileNoBlock(path string) (*os.File, error) {
	return os.Open(path)
}
