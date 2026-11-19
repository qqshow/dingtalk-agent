// Package decrypt implements native decryption of DingTalk desktop V3
// databases: SQLite pages encrypted with AES-128-ECB, key derived from the
// real numeric UID plus a per-account salt stored in <account>/user_config.
package decrypt

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/pbkdf2"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"dingtalk-agent/internal/sqlitex"
)

const (
	pageSize        = 4096
	walHeaderSize   = 32
	walFrameHdrSize = 24
	pbkdf2Iter      = 1000
	pbkdf2KeyLen    = 32
)

var (
	pbkdf2Salt  = []byte("666DingTalk888")[:8]
	sqliteMagic = []byte("SQLite format 3\x00")
)

// LoadSalt reads the per-account salt from <accountDir>/user_config
// (base64-encoded JSON with a "salt" field).
func LoadSalt(accountDir string) (string, error) {
	p := filepath.Join(accountDir, "user_config")
	b, err := os.ReadFile(p)
	if err != nil {
		return "", fmt.Errorf("read V3 user_config: %w", err)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(b)))
	if err != nil {
		return "", fmt.Errorf("decode V3 user_config: %w", err)
	}
	var v struct {
		Salt any `json:"salt"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return "", fmt.Errorf("parse V3 user_config: %w", err)
	}
	salt := strings.TrimSpace(fmt.Sprint(v.Salt))
	if v.Salt == nil || salt == "" {
		return "", errors.New("V3 user_config has no salt")
	}
	return salt, nil
}

// DeriveKey returns the 16-byte AES key for uid+salt.
func DeriveKey(uid, salt string) ([]byte, error) {
	dk, err := pbkdf2.Key(sha1.New, uid+salt, pbkdf2Salt, pbkdf2Iter, pbkdf2KeyLen)
	if err != nil {
		return nil, err
	}
	sum := md5.Sum(dk)
	return []byte(hex.EncodeToString(sum[:])[:16]), nil
}

// KeyFor derives the key for an account directory and uid.
func KeyFor(accountDir, uid string) ([]byte, error) {
	salt, err := LoadSalt(accountDir)
	if err != nil {
		return nil, err
	}
	return DeriveKey(uid, salt)
}

// CheckKey reports whether uid decrypts the first block of the account DB to
// the SQLite header.
func CheckKey(accountDir, uid string) bool {
	key, err := KeyFor(accountDir, uid)
	if err != nil {
		return false
	}
	f, err := os.Open(filepath.Join(accountDir, "DBFiles", "dingtalk.db"))
	if err != nil {
		return false
	}
	defer f.Close()
	buf := make([]byte, 16)
	if _, err := io.ReadFull(f, buf); err != nil {
		return false
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return false
	}
	block.Decrypt(buf, buf)
	return bytes.Equal(buf, sqliteMagic)
}

func ecbDecrypt(b cipher.Block, data []byte) {
	for i := 0; i+aes.BlockSize <= len(data); i += aes.BlockSize {
		b.Decrypt(data[i:i+aes.BlockSize], data[i:i+aes.BlockSize])
	}
}

func decryptPages(src, dst string, block cipher.Block) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	w := bufio.NewWriterSize(out, 1<<20)
	r := bufio.NewReaderSize(in, 1<<20)
	page := make([]byte, pageSize)
	for {
		n, err := io.ReadFull(r, page)
		if n == pageSize {
			ecbDecrypt(block, page)
		}
		if n > 0 {
			if _, werr := w.Write(page[:n]); werr != nil {
				out.Close()
				return werr
			}
		}
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			out.Close()
			return err
		}
	}
	if err := w.Flush(); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func walChecksum(data []byte, order binary.ByteOrder, s0, s1 uint32) (uint32, uint32) {
	for i := 0; i+8 <= len(data); i += 8 {
		s0 += order.Uint32(data[i:]) + s1
		s1 += order.Uint32(data[i+4:]) + s0
	}
	return s0, s1
}

// decryptWAL decrypts each frame's page and recomputes the frame checksum
// chain so SQLite accepts the plaintext WAL. Salts are kept, so frames from
// older WAL generations stay invalid exactly as in the original.
func decryptWAL(src, dst string, block cipher.Block) (int, error) {
	in, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer in.Close()
	r := bufio.NewReaderSize(in, 1<<20)
	hdr := make([]byte, walHeaderSize)
	if _, err := io.ReadFull(r, hdr); err != nil {
		return 0, fmt.Errorf("WAL header: %w", err)
	}
	var order binary.ByteOrder
	switch binary.BigEndian.Uint32(hdr) {
	case 0x377F0682:
		order = binary.LittleEndian
	case 0x377F0683:
		order = binary.BigEndian
	default:
		return 0, fmt.Errorf("unsupported WAL magic 0x%08x", binary.BigEndian.Uint32(hdr))
	}
	ps := int(binary.BigEndian.Uint32(hdr[8:12]))
	if ps == 1 {
		ps = 65536
	}
	if ps <= 0 || ps%aes.BlockSize != 0 {
		return 0, fmt.Errorf("invalid WAL page size %d", ps)
	}
	s0, s1 := walChecksum(hdr[:24], order, 0, 0)

	out, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	w := bufio.NewWriterSize(out, 1<<20)
	w.Write(hdr)
	fh := make([]byte, walFrameHdrSize)
	page := make([]byte, ps)
	frames := 0
	for {
		if _, err := io.ReadFull(r, fh); err != nil {
			break // EOF or truncated frame header (copied mid-write)
		}
		if _, err := io.ReadFull(r, page); err != nil {
			slog.Warn("ignoring incomplete WAL frame copied during an active write")
			break
		}
		ecbDecrypt(block, page)
		s0, s1 = walChecksum(fh[:8], order, s0, s1)
		s0, s1 = walChecksum(page, order, s0, s1)
		binary.BigEndian.PutUint32(fh[16:], s0)
		binary.BigEndian.PutUint32(fh[20:], s1)
		w.Write(fh)
		w.Write(page)
		frames++
	}
	if err := w.Flush(); err != nil {
		out.Close()
		return frames, err
	}
	return frames, out.Close()
}

// Result describes a freshly built plaintext snapshot.
type Result struct {
	Path         string // dingtalk_<ts>.db (messages)
	CalendarPath string // calendar_<ts>.db, "" when the account has no calendar DB
	WALFrames    int
	Size         int64
	Duration     time.Duration
}

// CalendarPathFor maps a message snapshot path to its calendar sibling.
func CalendarPathFor(msgPath string) string {
	dir, base := filepath.Split(msgPath)
	return filepath.Join(dir, "calendar_"+strings.TrimPrefix(base, "dingtalk_"))
}

// BuildSnapshot decrypts the account's message database and, when present,
// its calendar database (plus WALs) into new files in outDir, folding the WAL
// into each and validating the schema. The files are never modified after.
func BuildSnapshot(accountDir, uid, outDir string) (*Result, error) {
	start := time.Now()
	key, err := KeyFor(accountDir, uid)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	src := filepath.Join(accountDir, "DBFiles", "dingtalk.db")
	if _, err := os.Stat(src); err != nil {
		return nil, fmt.Errorf("DingTalk database not found: %w", err)
	}
	out := filepath.Join(outDir, fmt.Sprintf("dingtalk_%d.db", time.Now().UnixMilli()))
	frames, err := buildOne(src, out, block, "tbconversation")
	if err != nil {
		return nil, err
	}
	st, _ := os.Stat(out)
	res := &Result{Path: out, WALFrames: frames, Size: st.Size()}

	// The calendar is optional: a failure there must not block message sync.
	calSrc := filepath.Join(accountDir, "DBFiles", "calendar_v2.db")
	if _, err := os.Stat(calSrc); err == nil {
		calOut := CalendarPathFor(out)
		if _, err := buildOne(calSrc, calOut, block, "calendar_event"); err != nil {
			slog.Warn("calendar decrypt failed (messages are fine)", "err", err)
		} else {
			res.CalendarPath = calOut
		}
	}
	res.Duration = time.Since(start)
	return res, nil
}

func buildOne(src, out string, block cipher.Block, requiredTable string) (int, error) {
	cleanup := func() {
		for _, s := range []string{"", "-wal", "-shm", "-journal"} {
			os.Remove(out + s)
		}
	}
	var frames int
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		lastErr = decryptPages(src, out, block)
		if lastErr == nil {
			frames = 0
			if st, err := os.Stat(src + "-wal"); err == nil && st.Size() > walHeaderSize {
				frames, lastErr = decryptWAL(src+"-wal", out+"-wal", block)
			}
		}
		if lastErr == nil {
			lastErr = finalize(out, requiredTable)
		}
		if lastErr == nil {
			return frames, nil
		}
		cleanup()
		slog.Warn("decrypt attempt failed", "file", filepath.Base(src), "attempt", attempt, "err", lastErr)
		time.Sleep(time.Duration(attempt) * time.Second)
	}
	return 0, lastErr
}

// finalize checkpoints the WAL into the main file, switches to rollback
// journal mode (so read-only/immutable opens need no -shm) and validates.
func finalize(path, requiredTable string) error {
	db, err := sqlitex.Open(path, "")
	if err != nil {
		return err
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	var mode string
	if err := db.QueryRow("PRAGMA wal_checkpoint(TRUNCATE)").Scan(new(int), new(int), new(int)); err != nil {
		return fmt.Errorf("decrypted file is not a valid database (wrong key?): %w", err)
	}
	if err := db.QueryRow("PRAGMA journal_mode=DELETE").Scan(&mode); err != nil {
		return err
	}
	var n int
	if err := db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", requiredTable).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("decrypted database has no %s table (unsupported schema)", requiredTable)
	}
	if err := db.QueryRow(`SELECT count(*) FROM "` + requiredTable + `"`).Scan(&n); err != nil {
		return fmt.Errorf("decrypted database is corrupt: %w", err)
	}
	return nil
}
