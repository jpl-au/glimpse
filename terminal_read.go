//go:build unix

package main

import (
	"syscall"
	"time"
	"unsafe"
)

// readWithTimeout reads from fd with the given timeout using poll(2).
// Returns the number of bytes read and any error.
func readWithTimeout(fd int, buf []byte, timeout time.Duration) (int, error) {
	ms := int(timeout.Milliseconds())
	if ms <= 0 {
		ms = 1
	}

	pfd := struct {
		fd      int32
		events  int16
		revents int16
	}{
		fd:     int32(fd),
		events: 1, // POLLIN
	}

	n, _, errno := syscall.Syscall(
		syscall.SYS_POLL,
		uintptr(unsafe.Pointer(&pfd)),
		1,
		uintptr(ms),
	)
	if n == 0 || errno != 0 {
		if errno != 0 {
			return 0, errno
		}
		return 0, nil // timeout
	}

	return syscall.Read(fd, buf)
}
