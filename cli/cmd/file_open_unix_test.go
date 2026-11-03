//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package cmd

import (
	"syscall"
	"testing"
	"time"
)

func TestOpenUploadSourceRejectsFIFOWithoutBlocking(t *testing.T) {
	path := t.TempDir() + "/source.pipe"
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		source, err := openUploadSource(path)
		if source != nil {
			_ = source.Close()
		}
		done <- err
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("FIFO source should fail")
		}
	case <-time.After(time.Second):
		t.Fatal("opening a FIFO blocked instead of rejecting it")
	}
}
