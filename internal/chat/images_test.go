package chat

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qqshow/dingtalk-agent/internal/agentdb"
)

var (
	pngBytes  = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")
	jpegBytes = []byte("\xff\xd8\xff\xe0\x00\x10JFIF\x00")
)

func dataURL(mime string, b []byte) string {
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(b)
}

func TestParseImages(t *testing.T) {
	ok, err := parseImages([]string{dataURL("image/png", pngBytes), dataURL("image/jpeg", jpegBytes)})
	if err != nil || len(ok) != 2 || ok[0].mime != "image/png" || ok[1].mime != "image/jpeg" {
		t.Fatalf("valid images: %+v %v", ok, err)
	}
	if got := ok[0].dataURL(); got != dataURL("image/png", pngBytes) {
		t.Errorf("dataURL round trip = %q", got)
	}

	bad := map[string][]string{
		"too many":        {dataURL("image/png", pngBytes), dataURL("image/png", pngBytes), dataURL("image/png", pngBytes), dataURL("image/png", pngBytes), dataURL("image/png", pngBytes)},
		"not a data url":  {"https://example.com/a.png"},
		"not base64 flag": {"data:image/png," + string(pngBytes)},
		"unsupported":     {dataURL("image/svg+xml", []byte("<svg/>"))},
		"type mismatch":   {dataURL("image/png", jpegBytes)},
		"html as png":     {dataURL("image/png", []byte("<html><script>x</script>"))},
		"bad base64":      {"data:image/png;base64,@@@"},
		"empty":           {"data:image/png;base64,"},
		"too large":       {dataURL("image/png", append(append([]byte{}, pngBytes...), make([]byte, MaxImageBytes)...))},
	}
	for name, in := range bad {
		if _, err := parseImages(in); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestImagePathRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	good := strings.Repeat("a", 32) + ".png"
	if p, err := ImagePath(dir, 3, good); err != nil || p != filepath.Join(dir, imagesDirName, "3", good) {
		t.Fatalf("ImagePath(good) = %q %v", p, err)
	}
	for _, name := range []string{"../config.json", "..%2fx.png", strings.Repeat("a", 32) + ".svg", "x.png", ""} {
		if _, err := ImagePath(dir, 3, name); err == nil {
			t.Errorf("ImagePath(%q) should fail", name)
		}
	}
	if _, err := ImagePath(dir, 0, good); err == nil {
		t.Error("session 0 should fail")
	}
}

func TestSaveImages(t *testing.T) {
	dir := t.TempDir()
	imgs, err := parseImages([]string{dataURL("image/png", pngBytes), dataURL("image/jpeg", jpegBytes)})
	if err != nil {
		t.Fatal(err)
	}
	names, err := saveImages(dir, 7, imgs)
	if err != nil || len(names) != 2 || !strings.HasSuffix(names[0], ".png") || !strings.HasSuffix(names[1], ".jpg") {
		t.Fatalf("saveImages = %v %v", names, err)
	}
	for i, n := range names {
		p, err := ImagePath(dir, 7, n)
		if err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(p)
		if err != nil || string(b) != string(imgs[i].data) {
			t.Errorf("stored %s = %q %v", n, b, err)
		}
	}
	if names, err := saveImages(dir, 8, nil); err != nil || names != nil {
		t.Errorf("no images: %v %v", names, err)
	}
}

func TestUserContentNotesEarlierImages(t *testing.T) {
	m := agentdb.ChatMessage{Content: "这个报错怎么办", Context: "资料", Meta: agentdb.ChatMeta{Images: []string{"a.png", "b.png"}}}
	if got := userContent(m, false); got != "这个报错怎么办\n（附图片 2 张）" {
		t.Errorf("without context = %q", got)
	}
	if got := userContent(m, true); !strings.HasPrefix(got, "【引用资料】\n资料") || !strings.HasSuffix(got, "（附图片 2 张）") {
		t.Errorf("with context = %q", got)
	}
}
