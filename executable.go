//go:build !windows

package main

import (
	"os"
	"syscall"
)

func makeExecutable(path string) error {
	umask := syscall.Umask(0)
	syscall.Umask(umask)
	return os.Chmod(path, os.FileMode(0o777 & ^umask))
}
