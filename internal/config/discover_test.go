package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverAccountsFindsMacOSContainer(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("APPDATA", "")
	t.Setenv("LOCALAPPDATA", "")

	base := filepath.Join(home, "Library", "Containers", "ABCDE12345.com.dingtalk.mac",
		"Data", "Library", "Application Support", "DingTalkMac")
	acct := filepath.Join(base, "0123456789abcdef_v3")
	if err := os.MkdirAll(filepath.Join(acct, "DBFiles"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(acct, "DBFiles", "dingtalk.db"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	// A legacy _v2 account must be ignored.
	if err := os.MkdirAll(filepath.Join(base, "123_v2", "DBFiles"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := DiscoverAccounts()

	if len(got) != 1 || got[0].Dir != acct {
		t.Fatalf("DiscoverAccounts() = %+v, want only %s", got, acct)
	}
}
