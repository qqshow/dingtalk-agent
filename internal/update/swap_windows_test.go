//go:build windows

package update

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// A file held open (like Defender scanning a just-executed program) cannot
// be renamed on Windows; swap must wait for the lock to go away.
func TestSwapWaitsForLockedFile(t *testing.T) {
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
	go func() {
		time.Sleep(1500 * time.Millisecond)
		lock.Close()
	}()
	start := time.Now()
	if err := swap(exe, newPath); err != nil {
		t.Fatalf("swap failed: %v", err)
	}
	if time.Since(start) < time.Second {
		t.Fatal("swap should have waited for the lock")
	}
	if b, _ := os.ReadFile(exe); string(b) != "new" {
		t.Fatalf("exe = %q", b)
	}
	if b, _ := os.ReadFile(exe + ".old"); string(b) != "old" {
		t.Fatalf("old = %q", b)
	}
}
