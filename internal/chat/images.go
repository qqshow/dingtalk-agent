package chat

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const (
	MaxImages     = 4             // images per message
	MaxImageBytes = 5 << 20       // per image, decoded
	MaxBodyBytes  = 32 << 20      // request body for a message with images
	imagesDirName = "chat-images" // under the data dir, one folder per session
	imageNote     = "（附图片 %d 张）"  // stands in for images in earlier turns
)

// imageTypes maps the accepted image types to file extensions.
var imageTypes = map[string]string{
	"image/png":  ".png",
	"image/jpeg": ".jpg",
	"image/webp": ".webp",
	"image/gif":  ".gif",
}

var imageNameRe = regexp.MustCompile(`^[0-9a-f]{32}\.(png|jpg|webp|gif)$`)

// image is a validated upload.
type image struct {
	mime string
	data []byte
}

// dataURL is the form sent to the model.
func (im image) dataURL() string {
	return "data:" + im.mime + ";base64," + base64.StdEncoding.EncodeToString(im.data)
}

// parseImages validates data URLs sent by the page. The declared type must
// match the decoded bytes, so only real PNG / JPEG / WebP / GIF files pass.
func parseImages(urls []string) ([]image, error) {
	if len(urls) > MaxImages {
		return nil, fmt.Errorf("一次最多发送 %d 张图片", MaxImages)
	}
	out := make([]image, 0, len(urls))
	for i, u := range urls {
		head, payload, ok := strings.Cut(u, ",")
		mime, isB64 := strings.CutSuffix(strings.TrimPrefix(head, "data:"), ";base64")
		if !ok || !strings.HasPrefix(head, "data:") || !isB64 {
			return nil, fmt.Errorf("第 %d 张图片格式无效", i+1)
		}
		if _, known := imageTypes[mime]; !known {
			return nil, fmt.Errorf("第 %d 张图片类型 %q 不支持，请使用 PNG、JPEG、WebP 或 GIF", i+1, mime)
		}
		if base64.StdEncoding.DecodedLen(len(payload)) > MaxImageBytes+3 {
			return nil, fmt.Errorf("第 %d 张图片超过 %d MB", i+1, MaxImageBytes>>20)
		}
		data, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			return nil, fmt.Errorf("第 %d 张图片解码失败: %w", i+1, err)
		}
		if len(data) == 0 || len(data) > MaxImageBytes {
			return nil, fmt.Errorf("第 %d 张图片为空或超过 %d MB", i+1, MaxImageBytes>>20)
		}
		if got := http.DetectContentType(data); got != mime {
			return nil, fmt.Errorf("第 %d 张图片内容与类型不符（%s）", i+1, got)
		}
		out = append(out, image{mime: mime, data: data})
	}
	return out, nil
}

// SessionImageDir is where a session's images are kept.
func SessionImageDir(dataDir string, session int64) string {
	return filepath.Join(dataDir, imagesDirName, strconv.FormatInt(session, 10))
}

// ImagePath returns the file for a stored image, or an error when the name
// is not one this package writes (guards against path traversal).
func ImagePath(dataDir string, session int64, name string) (string, error) {
	if session <= 0 || !imageNameRe.MatchString(name) {
		return "", fmt.Errorf("invalid image %q", name)
	}
	return filepath.Join(SessionImageDir(dataDir, session), name), nil
}

// saveImages writes the images into the session folder and returns their
// file names. Files already written are removed when a later one fails.
func saveImages(dataDir string, session int64, imgs []image) ([]string, error) {
	if len(imgs) == 0 {
		return nil, nil
	}
	dir := SessionImageDir(dataDir, session)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("创建图片目录失败: %w", err)
	}
	names := make([]string, 0, len(imgs))
	for _, im := range imgs {
		var b [16]byte
		if _, err := rand.Read(b[:]); err != nil {
			removeImages(dir, names)
			return nil, err
		}
		name := hex.EncodeToString(b[:]) + imageTypes[im.mime]
		if err := os.WriteFile(filepath.Join(dir, name), im.data, 0o600); err != nil {
			removeImages(dir, names)
			return nil, fmt.Errorf("保存图片失败: %w", err)
		}
		names = append(names, name)
	}
	return names, nil
}

func removeImages(dir string, names []string) {
	for _, n := range names {
		os.Remove(filepath.Join(dir, n))
	}
}
