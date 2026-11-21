//go:build windows

package update

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Security software keeps a just-written program open (read allowed, delete
// not) while scanning it. swap must still install it right away, and the
// leftover .new is removed once the lock goes away.
func TestSwapWithLockedNewFile(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "dingtalk-agent.exe")
	newPath := exe + ".new"
	os.WriteFile(exe, []byte("old"), 0o755)
	os.WriteFile(newPath, []byte("new"), 0o755)

	lock, err := os.Open(newPath) // Go opens without FILE_SHARE_DELETE
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(newPath, exe+".probe"); err == nil {
		lock.Close()
		t.Skip("this Windows build allows renaming open files; nothing to test")
	}
	start := time.Now()
	if err := swap(exe, newPath); err != nil {
		t.Fatalf("swap failed: %v", err)
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Fatalf("swap should not wait for the lock, took %v", d)
	}
	if b, _ := os.ReadFile(exe); string(b) != "new" {
		t.Fatalf("exe = %q", b)
	}
	if b, _ := os.ReadFile(exe + ".old"); string(b) != "old" {
		t.Fatalf("old = %q", b)
	}
	lock.Close()
	for i := 0; i < 40; i++ { // background removal of .new
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			return
		}
		time.Sleep(250 * time.Millisecond)
	}
	t.Fatal(".new was not removed after the lock went away")
}
