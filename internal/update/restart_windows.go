//go:build windows

package update

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

var procGetConsoleWindow = syscall.NewLazyDLL("kernel32.dll").NewProc("GetConsoleWindow")

// Restart starts the (updated) executable with the same arguments. The caller
// must exit right after it returns nil; the new process waits for the port to
// be released (see RestartEnv).
//
// Started from a terminal, the new process shares that console. Running as a
// desktop app (console already detached), it is started without a console so
// it comes back as a window + tray icon and no console window flashes up.
func Restart() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Env = append(os.Environ(), RestartEnv+"=1")
	if h, _, _ := procGetConsoleWindow.Call(); h == 0 {
		const detachedProcess = 0x00000008
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: detachedProcess}
	} else {
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	}
	return cmd.Start()
}
