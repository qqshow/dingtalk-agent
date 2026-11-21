package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewer(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"1.2.0", "1.1.1", true},
		{"v1.2.0", "1.1.9", true},
		{"1.10.0", "1.9.0", true},
		{"2.0.0", "1.99.99", true},
		{"1.1.1", "1.1.1", false},
		{"1.1.0", "1.1.1", false},
		{"1.2.0", "1.2.0-dev", true},
		{"1.2.0-rc1", "1.2.0", false},
		{"1.2.0", "dev", false},
		{"1.2.0", "dev-abc1234", false},
		{"garbage", "1.0.0", false},
	}
	for _, c := range cases {
		if got := Newer(c.latest, c.current); got != c.want {
			t.Errorf("Newer(%q, %q) = %v, want %v", c.latest, c.current, got, c.want)
		}
	}
}

func TestAssetName(t *testing.T) {
	if got := AssetName("windows", "amd64"); got != "dingtalk-agent-windows-amd64.zip" {
		t.Error(got)
	}
	if got := AssetName("darwin", "arm64"); got != "dingtalk-agent-darwin-arm64.tar.gz" {
		t.Error(got)
	}
}

func makeZip(t *testing.T, files map[string][]byte) []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, data := range files {
		w, _ := zw.Create(name)
		w.Write(data)
	}
	zw.Close()
	return buf.Bytes()
}

func makeTarGz(t *testing.T, files map[string][]byte) []byte {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, data := range files {
		tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(data)), Typeflag: tar.TypeReg})
		tw.Write(data)
	}
	tw.Close()
	gz.Close()
	return buf.Bytes()
}

// fakeGitHub serves a release with one platform archive and SHA256SUMS.txt.
func fakeGitHub(t *testing.T, tag, assetName string, archive []byte, badSum bool) *httptest.Server {
	sum := sha256.Sum256(archive)
	hexSum := hex.EncodeToString(sum[:])
	if badSum {
		hexSum = strings.Repeat("0", 64)
	}
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("User-Agent"), "dingtalk-agent/") {
			t.Errorf("user agent = %q", r.Header.Get("User-Agent"))
		}
		switch r.URL.Path {
		case "/repos/x/releases/latest":
			json.NewEncoder(w).Encode(Release{Tag: tag, URL: "https://example/release", Notes: "notes",
				Assets: []Asset{
					{Name: assetName, URL: srv.URL + "/dl/" + assetName, Size: int64(len(archive))},
					{Name: "SHA256SUMS.txt", URL: srv.URL + "/dl/SHA256SUMS.txt"},
				}})
		case "/dl/" + assetName:
			w.Write(archive)
		case "/dl/SHA256SUMS.txt":
			fmt.Fprintf(w, "%s  other-file.zip\n%s  %s\n", strings.Repeat("1", 64), hexSum, assetName)
		default:
			http.NotFound(w, r)
		}
	}))
	return srv
}

func newTestUpdater(t *testing.T, srvURL, goos string) (*Updater, string) {
	dir := t.TempDir()
	exe := filepath.Join(dir, binName(goos))
	os.WriteFile(exe, []byte("old program"), 0o755)
	u := New("1.1.1")
	u.API = srvURL + "/repos/x"
	u.Exe, u.GOOS, u.GOARCH = exe, goos, "amd64"
	u.Verify = func(ctx context.Context, bin, want string) error {
		b, _ := os.ReadFile(bin)
		if !bytes.HasPrefix(b, []byte("new program")) || want != "1.2.0" {
			return errors.New("verify failed")
		}
		return nil
	}
	return u, exe
}

func TestCheckAndApplyZip(t *testing.T) {
	archive := makeZip(t, map[string][]byte{
		"dingtalk-agent-windows-amd64/README.md":          []byte("readme"),
		"dingtalk-agent-windows-amd64/dingtalk-agent.exe": []byte("new program v1.2.0"),
	})
	srv := fakeGitHub(t, "v1.2.0", "dingtalk-agent-windows-amd64.zip", archive, false)
	defer srv.Close()
	u, exe := newTestUpdater(t, srv.URL, "windows")

	st, err := u.Check(context.Background())
	if err != nil || !st.Available || !st.Supported || st.Latest.Version() != "1.2.0" {
		t.Fatalf("check: %+v %v", st, err)
	}
	if err := u.Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(exe); string(b) != "new program v1.2.0" {
		t.Fatalf("exe = %q", b)
	}
	if b, _ := os.ReadFile(exe + ".old"); string(b) != "old program" {
		t.Fatalf("old = %q", b)
	}
	st = u.Status()
	if !st.Restarting || st.Applying || st.Downloaded != int64(len(archive)) {
		t.Fatalf("status after apply: %+v", st)
	}
	if err := u.Apply(context.Background()); err == nil {
		t.Fatal("second apply while restarting should fail")
	}
}

func TestApplyTarGz(t *testing.T) {
	archive := makeTarGz(t, map[string][]byte{"dingtalk-agent-darwin-amd64/dingtalk-agent": []byte("new program")})
	srv := fakeGitHub(t, "v1.2.0", "dingtalk-agent-darwin-amd64.tar.gz", archive, false)
	defer srv.Close()
	u, exe := newTestUpdater(t, srv.URL, "darwin")
	if _, err := u.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := u.Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(exe); string(b) != "new program" {
		t.Fatalf("exe = %q", b)
	}
}

func TestApplyRejectsBadChecksum(t *testing.T) {
	archive := makeZip(t, map[string][]byte{"x/dingtalk-agent.exe": []byte("new program")})
	srv := fakeGitHub(t, "v1.2.0", "dingtalk-agent-windows-amd64.zip", archive, true)
	defer srv.Close()
	u, exe := newTestUpdater(t, srv.URL, "windows")
	u.Check(context.Background())
	err := u.Apply(context.Background())
	if err == nil || !strings.Contains(err.Error(), "校验失败") {
		t.Fatalf("err = %v", err)
	}
	if b, _ := os.ReadFile(exe); string(b) != "old program" {
		t.Fatal("program must be untouched after a failed update")
	}
	if st := u.Status(); st.ApplyError == "" || st.Restarting {
		t.Fatalf("status = %+v", st)
	}
	if _, err := os.Stat(exe + ".new"); !os.IsNotExist(err) {
		t.Fatal("leftover .new file")
	}
}

func TestApplyRejectsBrokenBinary(t *testing.T) {
	archive := makeZip(t, map[string][]byte{"x/dingtalk-agent.exe": []byte("")}) // emptied by antivirus
	srv := fakeGitHub(t, "v1.2.0", "dingtalk-agent-windows-amd64.zip", archive, false)
	defer srv.Close()
	u, exe := newTestUpdater(t, srv.URL, "windows")
	u.Check(context.Background())
	if err := u.Apply(context.Background()); err == nil {
		t.Fatal("broken binary accepted")
	}
	if b, _ := os.ReadFile(exe); string(b) != "old program" {
		t.Fatal("program must be untouched")
	}
}

func TestNoUpdateForCurrentOrDev(t *testing.T) {
	archive := makeZip(t, map[string][]byte{"x/dingtalk-agent.exe": []byte("new program")})
	srv := fakeGitHub(t, "v1.1.1", "dingtalk-agent-windows-amd64.zip", archive, false)
	defer srv.Close()
	u, _ := newTestUpdater(t, srv.URL, "windows")
	if st, _ := u.Check(context.Background()); st.Available {
		t.Fatal("same version reported as update")
	}
	if err := u.Apply(context.Background()); err == nil {
		t.Fatal("apply without update should fail")
	}
	u.Current = "dev"
	if ok, _ := u.supported(); ok {
		t.Fatal("dev build should not self-update")
	}
}
