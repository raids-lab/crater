//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package storage

import (
	"os"

	"golang.org/x/sys/unix"
)

// Publish from the already-open staging directory. Resolving the source
// relative to its directory descriptor prevents a writable-parent rename race
// from substituting another staging directory after the upload completes.
func publishUploadNoClobber(stage, parent *os.Root, targetName string) error {
	stageDirectory, parentDirectory, err := openUploadDirectoryHandles(stage, parent)
	if err != nil {
		return err
	}
	defer stageDirectory.Close()
	defer parentDirectory.Close()
	return unix.Linkat(
		int(stageDirectory.Fd()),
		uploadStagePayload,
		int(parentDirectory.Fd()),
		targetName,
		0,
	)
}

func publishUploadOverwrite(stage, parent *os.Root, targetName string) error {
	stageDirectory, parentDirectory, err := openUploadDirectoryHandles(stage, parent)
	if err != nil {
		return err
	}
	defer stageDirectory.Close()
	defer parentDirectory.Close()
	return unix.Renameat(
		int(stageDirectory.Fd()),
		uploadStagePayload,
		int(parentDirectory.Fd()),
		targetName,
	)
}

func openUploadDirectoryHandles(stage, parent *os.Root) (
	stageDirectory *os.File,
	parentDirectory *os.File,
	err error,
) {
	stageDirectory, err = stage.Open(".")
	if err != nil {
		return nil, nil, err
	}
	parentDirectory, err = parent.Open(".")
	if err != nil {
		_ = stageDirectory.Close()
		return nil, nil, err
	}
	return stageDirectory, parentDirectory, nil
}
