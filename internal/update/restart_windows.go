//go:build windows

package update

import (
	"os"
	"os/exec"
	"path/filepath"
)

// Restart starts the (updated) executable with the same arguments, attached
// to the same console. The caller must exit right after it returns nil; the
// new process waits for the port to be released (see RestartEnv).
func Restart() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Env = append(os.Environ(), RestartEnv+"=1")
	return cmd.Start()
}
