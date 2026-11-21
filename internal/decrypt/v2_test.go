package decrypt

import (
	"crypto/aes"
	"os"
	"path/filepath"
	"testing"

	"github.com/qqshow/dingtalk-agent/internal/sqlitex"
)

// Reference value: dingwave's GenerateKey (md5(uid) hex[:16]).
func TestDeriveKeyV2(t *testing.T) {
	// md5("123456789") = 25f9e794323b453885f5181f1b624d0b
	if got, want := string(DeriveKeyV2("123456789")), "25f9e794323b4538"; got != want {
		t.Fatalf("key = %s, want %s", got, want)
	}
}

func TestIsV2(t *testing.T) {
	// Native separators: a backslash is an ordinary filename character on Unix.
	base := filepath.Join("home", "x", "DingTalk")
	for dir, want := range map[string]bool{
		filepath.Join(base, "69465484_v2"):                              true,
		filepath.Join(base, "69465484_V2") + string(filepath.Separator): true,
		filepath.Join(base, "0123456789abcdef_v3"):                      false,
	} {
		if IsV2(dir) != want {
			t.Errorf("IsV2(%q) = %v", dir, !want)
		}
	}
}

// End to end: a real SQLite DB encrypted the V2 way decrypts into a usable snapshot.
func TestBuildSnapshotV2(t *testing.T) {
	tmp := t.TempDir()
	plain := filepath.Join(tmp, "plain.db")
	db, err := sqlitex.Open(plain, "mode=rwc")
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`PRAGMA page_size=4096`,
		`PRAGMA journal_mode=DELETE`,
		`CREATE TABLE tbconversation (cid TEXT PRIMARY KEY, title TEXT)`,
		`INSERT INTO tbconversation VALUES ('1:2', 'hello v2')`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(q, err)
		}
	}
	db.Close()

	const uid = "42424242"
	acct := filepath.Join(tmp, uid+"_v2")
	os.MkdirAll(filepath.Join(acct, "DBFiles"), 0o755)
	raw, err := os.ReadFile(plain)
	if err != nil || len(raw)%pageSize != 0 {
		t.Fatalf("plain db: %v, size %d", err, len(raw))
	}
	block, _ := aes.NewCipher(DeriveKeyV2(uid))
	for i := 0; i < len(raw); i += 16 {
		block.Encrypt(raw[i:i+16], raw[i:i+16])
	}
	os.WriteFile(filepath.Join(acct, "DBFiles", "dingtalk.db"), raw, 0o644)

	if !CheckKey(acct, uid) || CheckKey(acct, "11111111") {
		t.Fatal("CheckKey does not distinguish the V2 key")
	}
	out := filepath.Join(tmp, "out")
	os.MkdirAll(out, 0o755)
	res, err := BuildSnapshot(acct, uid, out)
	if err != nil {
		t.Fatal(err)
	}
	snap, err := sqlitex.Open(res.Path, "mode=ro")
	if err != nil {
		t.Fatal(err)
	}
	defer snap.Close()
	var title string
	if err := snap.QueryRow(`SELECT title FROM tbconversation`).Scan(&title); err != nil || title != "hello v2" {
		t.Fatalf("title = %q, %v", title, err)
	}
}
