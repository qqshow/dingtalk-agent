//go:build !windows

package update

import (
	"os"
	"path/filepath"
	"syscall"
)

// Restart replaces the current process with the (updated) executable, keeping
// the PID, arguments and environment. Listening sockets are close-on-exec, so
// the new program can bind the same port. It only returns on failure.
func Restart() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	return syscall.Exec(exe, os.Args, append(os.Environ(), RestartEnv+"=1"))
}
