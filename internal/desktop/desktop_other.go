//go:build !windows

package desktop

import (
	"fmt"
	"os"
)

// Supported reports whether this build can show a native window.
func Supported() bool { return false }

// Detached is always false: without a native window the terminal is kept.
func Detached() bool { return false }

func HideConsole()       {}
func AllowForeground()   {}
func OpenBrowser(string) {}
func Show()              {}
func Quit()              {}

// Run is not supported here; it returns immediately.
func Run(Options) {}

func MessageBox(title, text string, isError bool) { fmt.Fprintln(os.Stderr, title+":", text) }
