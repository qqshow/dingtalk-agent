// Package export writes self-contained JSON exports (with copied attachments)
// and serves them as zip archives.
package export

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/qqshow/dingtalk-agent/internal/store"
)

type Exporter struct {
	Dir string // base export directory
}

type exportMsg struct {
	MessageID       int64              `json:"message_id"`
	SenderID        int64              `json:"sender_id"`
	SenderName      string             `json:"sender_name"`
	ContentType     int64              `json:"content_type"`
	ContentTypeName string             `json:"content_type_name"`
	Text            string             `json:"text"`
	Content         string             `json:"content"`
	CreatedAt       int64              `json:"created_at"`
	CreatedAtStr    string             `json:"created_at_str"`
	Attachments     []store.Attachment `json:"attachments"`
	AtIDs           map[string]string  `json:"at_ids"`
	ImagePaths      []string           `json:"image_export_paths,omitempty"`
}

type exportConv struct {
	ConversationID string      `json:"conversation_id"`
	Title          string      `json:"title"`
	Type           string      `json:"type"`
	MemberCount    int64       `json:"member_count"`
	MessageCount   int         `json:"message_count"`
	Messages       []exportMsg `json:"messages"`
}

type exportFile struct {
	ExportTime         string       `json:"export_time"`
	ExportType         string       `json:"export_type"`
	AccountUID         string       `json:"account_uid"`
	SinceTime          int64        `json:"since_time,omitempty"`
	TotalConversations int          `json:"total_conversations"`
	Conversations      []exportConv `json:"conversations"`
}

type copier struct {
	root   string
	copied map[string]string // source -> relative path
	stats  map[string]int
}

func sanitize(name string) string {
	r := strings.NewReplacer("<", "_", ">", "_", ":", "_", `"`, "_", "/", "_", `\`, "_", "|", "_", "?", "_", "*", "_")
	name = r.Replace(name)
	if rs := []rune(name); len(rs) > 200 {
		ext := filepath.Ext(name)
		name = string(rs[:200-len([]rune(ext))]) + ext
	}
	return name
}

func (c *copier) copy(src, subdir, name string) string {
	key := strings.ToLower(filepath.Clean(src))
	if rel, ok := c.copied[key]; ok {
		c.stats["deduplicated"]++
		return rel
	}
	dir := filepath.Join(c.root, subdir)
	os.MkdirAll(dir, 0o755)
	dst := filepath.Join(dir, sanitize(name))
	stem, ext := strings.TrimSuffix(dst, filepath.Ext(dst)), filepath.Ext(dst)
	for i := 1; ; i++ {
		if _, err := os.Stat(dst); os.IsNotExist(err) {
			break
		}
		dst = fmt.Sprintf("%s_%d%s", stem, i, ext)
	}
	in, err := os.Open(src)
	if err != nil {
		c.stats["errors"]++
		return ""
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		c.stats["errors"]++
		return ""
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		c.stats["errors"]++
		return ""
	}
	out.Close()
	rel, _ := filepath.Rel(c.root, dst)
	rel = filepath.ToSlash(rel)
	c.copied[key] = rel
	return rel
}

func agentContent(m *store.Message, imgs []string) string {
	t := m.AgentText()
	if m.ContentType == store.CTImage {
		if len(imgs) > 0 && imgs[0] != "" {
			return "[图片: " + imgs[0] + "]"
		}
		return "[图片未下载到本地]"
	}
	for _, p := range imgs {
		if p != "" {
			t = strings.Replace(t, "[图片]", "[图片: "+p+"]", 1)
		}
	}
	t = strings.ReplaceAll(t, "[图片]", "[图片未下载到本地]")
	var lines []string
	for _, a := range m.Attachments {
		if a.Filename == "" {
			continue
		}
		if a.ExportPath != "" {
			lines = append(lines, fmt.Sprintf("[附件: %s | %s]", a.Filename, a.ExportPath))
		} else if a.LocalAvailable != nil && !*a.LocalAvailable {
			lines = append(lines, "[附件未下载: "+a.Filename+"]")
		}
	}
	if len(lines) > 0 {
		t = strings.TrimSpace(t + "\n" + strings.Join(lines, "\n"))
	}
	return t
}

// Export writes the given conversations (all when cids is empty). Messages
// newer than since (ms, 0 = all) are included.
func (e *Exporter) Export(s *store.Snapshot, uid string, cids []string, since int64) (string, error) {
	typ := "selected"
	if len(cids) == 0 {
		typ = "full"
		for _, c := range s.Convs {
			cids = append(cids, c.CID)
		}
	}
	name := fmt.Sprintf("%s_export_%s_%s", typ, uid, time.Now().Format("20060102_150405"))
	root := filepath.Join(e.Dir, name)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", err
	}
	cp := &copier{root: root, copied: map[string]string{}, stats: map[string]int{}}
	out := exportFile{ExportTime: time.Now().Format(time.RFC3339), ExportType: typ, AccountUID: uid, SinceTime: since}

	for _, cid := range cids {
		c := s.ConvByID[cid]
		if c == nil {
			continue
		}
		msgs, err := s.Range(store.RangeQuery{From: since + 1, CIDs: []string{cid}, WithImages: true})
		if err != nil {
			return "", err
		}
		ec := exportConv{ConversationID: cid, Title: c.Title, Type: c.Type, MemberCount: c.MemberCount,
			Messages: []exportMsg{}}
		for _, m := range msgs {
			var imgs []string
			if m.ImageInfo != nil {
				for i, img := range m.ImageInfo.Images {
					ext := filepath.Ext(img.LocalPath)
					if ext == "" {
						ext = ".jpg"
					}
					n := fmt.Sprintf("%s_%d", strings.Split(cid, ":")[0], m.ID)
					if len(m.ImageInfo.Images) > 1 {
						n += fmt.Sprintf("_%d", i)
					}
					rel := cp.copy(img.LocalPath, "images", n+ext)
					if rel != "" {
						cp.stats["images"]++
					}
					imgs = append(imgs, rel)
				}
			}
			for i := range m.Attachments {
				a := &m.Attachments[i]
				if a.LocalPath == "" {
					if a.Filename != "" {
						cp.stats["skipped"]++
					}
					continue
				}
				if rel := cp.copy(a.LocalPath, "files", a.Filename); rel != "" {
					a.ExportPath = rel
					cp.stats["files"]++
				}
			}
			ec.Messages = append(ec.Messages, exportMsg{
				MessageID: m.ID, SenderID: m.SenderID, SenderName: m.SenderName, ContentType: m.ContentType,
				ContentTypeName: m.ContentTypeName, Text: m.Text, Content: agentContent(m, imgs),
				CreatedAt: m.CreatedAt, CreatedAtStr: m.CreatedAtStr, Attachments: m.Attachments,
				AtIDs: m.AtIDs, ImagePaths: imgs,
			})
		}
		ec.MessageCount = len(ec.Messages)
		if typ == "full" && ec.MessageCount == 0 {
			continue
		}
		out.Conversations = append(out.Conversations, ec)
	}
	out.TotalConversations = len(out.Conversations)
	f, err := os.Create(filepath.Join(root, "export.json"))
	if err != nil {
		return "", err
	}
	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		f.Close()
		return "", err
	}
	f.Close()
	slog.Info("export complete", "dir", name, "conversations", out.TotalConversations, "attachments", cp.stats)
	return root, nil
}

type Entry struct {
	Filename    string  `json:"filename"`
	Type        string  `json:"type"`
	Size        int64   `json:"size"`
	Modified    float64 `json:"modified"`
	DownloadURL string  `json:"download_url"`
}

func (e *Exporter) List() []Entry {
	ents, _ := os.ReadDir(e.Dir)
	out := []Entry{}
	for _, d := range ents {
		if !d.IsDir() {
			continue
		}
		js := filepath.Join(e.Dir, d.Name(), "export.json")
		st, err := os.Stat(js)
		if err != nil {
			continue
		}
		out = append(out, Entry{Filename: d.Name(), Type: "directory", Size: st.Size(),
			Modified: float64(st.ModTime().UnixMilli()) / 1000, DownloadURL: "/api/exports/" + d.Name() + "/download"})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Filename > out[j].Filename })
	return out
}

// Resolve returns the export directory for name, rejecting path traversal.
func (e *Exporter) Resolve(name string) (string, bool) {
	if name == "" || strings.ContainsAny(name, `/\`) || name == "." || name == ".." {
		return "", false
	}
	p := filepath.Join(e.Dir, name)
	st, err := os.Stat(p)
	return p, err == nil && st.IsDir()
}

var storedExt = map[string]bool{".7z": true, ".avi": true, ".bmp": true, ".docx": true, ".gif": true,
	".gz": true, ".jpeg": true, ".jpg": true, ".m4a": true, ".mp3": true, ".mp4": true, ".pdf": true,
	".png": true, ".rar": true, ".wav": true, ".webp": true, ".xls": true, ".xlsx": true, ".zip": true, ".pptx": true}

// WriteZip streams the export directory as a zip archive.
func WriteZip(w io.Writer, dir string) error {
	zw := zip.NewWriter(w)
	base := filepath.Dir(dir)
	err := filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(base, p)
		h, _ := zip.FileInfoHeader(info)
		h.Name = filepath.ToSlash(rel)
		h.Method = zip.Deflate
		if storedExt[strings.ToLower(filepath.Ext(p))] {
			h.Method = zip.Store
		}
		fw, err := zw.CreateHeader(h)
		if err != nil {
			return err
		}
		f, err := os.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(fw, f)
		return err
	})
	if err != nil {
		return err
	}
	return zw.Close()
}
