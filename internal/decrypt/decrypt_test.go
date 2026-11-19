package decrypt

import (
	"crypto/aes"
	"encoding/base64"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// Reference value computed with the original Python implementation
// (hashlib.pbkdf2_hmac('sha1', uid+salt, b'666DingT', 1000, 32) -> md5 hex[:16]).
func TestDeriveKeyMatchesPython(t *testing.T) {
	key, err := DeriveKey("123456789", "abcsalt")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(key), "5d6b06a070e74f40"; got != want {
		t.Fatalf("key = %s, want %s", got, want)
	}
}

func TestCheckKeyRoundTrip(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "DBFiles"), 0o755)
	cfg := base64.StdEncoding.EncodeToString([]byte(`{"salt":"s4lt"}`))
	os.WriteFile(filepath.Join(dir, "user_config"), []byte(cfg), 0o644)

	key, err := KeyFor(dir, "42424242")
	if err != nil {
		t.Fatal(err)
	}
	block, _ := aes.NewCipher(key)
	page := make([]byte, pageSize)
	copy(page, sqliteMagic)
	enc := make([]byte, pageSize)
	for i := 0; i < pageSize; i += 16 {
		block.Encrypt(enc[i:i+16], page[i:i+16])
	}
	os.WriteFile(filepath.Join(dir, "DBFiles", "dingtalk.db"), enc, 0o644)

	if !CheckKey(dir, "42424242") {
		t.Fatal("correct uid rejected")
	}
	if CheckKey(dir, "11111111") {
		t.Fatal("wrong uid accepted")
	}
}

func TestWALChecksum(t *testing.T) {
	// s0/s1 recurrence from the SQLite WAL format spec.
	data := make([]byte, 16)
	binary.LittleEndian.PutUint32(data[0:], 1)
	binary.LittleEndian.PutUint32(data[4:], 2)
	binary.LittleEndian.PutUint32(data[8:], 3)
	binary.LittleEndian.PutUint32(data[12:], 4)
	s0, s1 := walChecksum(data, binary.LittleEndian, 0, 0)
	// step1: s0=0+1+0=1, s1=0+2+1=3; step2: s0=1+3+3=7, s1=3+4+7=14
	if s0 != 7 || s1 != 14 {
		t.Fatalf("got %d,%d", s0, s1)
	}
}
