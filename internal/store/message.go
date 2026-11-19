package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/qqshow/dingtalk-agent/internal/sqlitex"
)

// Content types seen in DingTalk messages.
const (
	CTText     = 1
	CTImage    = 2
	CTVoice    = 300
	CTFile     = 501
	CTRichText = 1200
	CTCall     = 1101
	CTApproval = 1400
	CTQuote    = 3100
)

var ContentTypeNames = map[int64]string{
	1: "文本", 2: "图片", 4: "文件", 102: "系统消息", 104: "系统通知", 202: "链接",
	300: "语音", 400: "视频", 500: "位置", 501: "文件", 503: "文件", 1101: "通话",
	1200: "富文本", 1201: "交互卡片", 1202: "系统提示", 1400: "审批", 1500: "任务",
	1600: "日程", 2900: "互动卡片", 2950: "小程序卡片", 3100: "引用消息",
}

type Attachment struct {
	Type           int64  `json:"type"`
	URL            string `json:"url"`
	Size           int64  `json:"size"`
	Filename       string `json:"filename,omitempty"`
	FileSize       int64  `json:"file_size,omitempty"`
	FileType       string `json:"file_type,omitempty"`
	LocalAvailable *bool  `json:"local_available,omitempty"`
	LocalPath      string `json:"local_path,omitempty"`
	Markdown       string `json:"markdown,omitempty"`
	Description    string `json:"description,omitempty"`
	ExportPath     string `json:"export_path,omitempty"`
}

type Image struct {
	FileSize   int64  `json:"file_size"`
	URL        string `json:"url"`
	Cached     bool   `json:"cached"`
	LocalPath  string `json:"local_path"`
	Src        string `json:"src"`                 // best local copy (original when cached)
	ThumbSrc   string `json:"thumb_src,omitempty"` // 720px thumbnail, when cached
	IsOriginal bool   `json:"is_original"`
}

type ImageInfo struct {
	FileSize   int64   `json:"file_size"`
	URL        string  `json:"url"`
	Cached     bool    `json:"cached"`
	LocalPath  string  `json:"local_path"`
	Src        string  `json:"src,omitempty"`
	IsOriginal bool    `json:"is_original"`
	Images     []Image `json:"images"`
}

type Message struct {
	ID              int64             `json:"id"`
	CID             string            `json:"cid"`
	SenderID        int64             `json:"sender_id"`
	SenderName      string            `json:"sender_name"`
	ContentType     int64             `json:"content_type"`
	ContentTypeName string            `json:"content_type_name"`
	Text            string            `json:"text"`
	CreatedAt       int64             `json:"created_at"`
	CreatedAtStr    string            `json:"created_at_str"`
	RecallStatus    int64             `json:"recall_status"`
	AtIDs           map[string]string `json:"at_ids"`
	Attachments     []Attachment      `json:"attachments"`
	ImageInfo       *ImageInfo        `json:"image_info,omitempty"`
	IsMine          bool              `json:"is_mine"`
	MentionsMe      bool              `json:"mentions_me"`
	MentionsAll     bool              `json:"mentions_all"`
	CreatorType     int64             `json:"creator_type"`
	IsSystem        bool              `json:"is_system"`
}

const msgCols = `mid, cid, senderId, createdAt, contentType, content, recallStatus, atIds, attachments, creatorType`

// scanMessages parses rows selected with msgCols.
func (s *Snapshot) scanMessages(rows rowsScanner, withImages bool) ([]*Message, error) {
	var out []*Message
	for rows.Next() {
		var mid, cid, sender, created, ct, content, recall, atIDs, atts, creator any
		if err := rows.Scan(&mid, &cid, &sender, &created, &ct, &content, &recall, &atIDs, &atts, &creator); err != nil {
			return nil, err
		}
		m := s.parseMessage(sqlitex.Int(mid), sqlitex.Str(cid), sqlitex.Int(sender), sqlitex.Int(created),
			sqlitex.Int(ct), sqlitex.Str(content), sqlitex.Int(recall), sqlitex.Str(atIDs), sqlitex.Str(atts), withImages)
		m.CreatorType = sqlitex.Int(creator)
		if m.CreatorType == 2 || m.ContentType == 102 || m.ContentType == 104 || m.ContentType == 1202 {
			// System tips (e.g. "xx 发起了群聊") carry the actor as sender; they are not chat.
			m.IsSystem, m.IsMine, m.MentionsMe = true, false, false
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

type rowsScanner interface {
	Next() bool
	Scan(...any) error
	Err() error
}

func (s *Snapshot) parseMessage(mid int64, cid string, sender, created, ct int64, contentRaw string,
	recall int64, atRaw, attRaw string, withImages bool) *Message {

	content := map[string]any{}
	if contentRaw != "" {
		if json.Unmarshal([]byte(contentRaw), &content) != nil {
			content = map[string]any{"text": contentRaw}
		}
	}

	var text string
	switch ct {
	case CTText:
		text = sqlitex.Str(content["text"])
	case CTImage:
		text = "[图片]"
	case CTVoice:
		text = "[语音]"
	case CTFile:
		text = "[文件]"
	case CTRichText, 1201, 1202:
		text = richText(content)
	case CTQuote:
		text = quoteText(content)
	case 2900, 2950:
		text = cardText(content)
	case CTApproval:
		text = approvalText(content)
	case CTCall:
		text = "[通话记录]"
	default:
		text = sqlitex.Str(content["text"])
	}

	m := &Message{
		ID: mid, CID: cid, SenderID: sender, ContentType: ct,
		Text: cleanText(text), CreatedAt: created, CreatedAtStr: FormatTS(created),
		RecallStatus: recall, AtIDs: map[string]string{},
	}
	if n, ok := ContentTypeNames[ct]; ok {
		m.ContentTypeName = n
	} else {
		m.ContentTypeName = "未知(" + strconv.FormatInt(ct, 10) + ")"
	}
	m.SenderName = cleanText(s.UserName(cid, sender))
	m.IsMine = s.MyUID != 0 && sender == s.MyUID

	if atRaw != "" {
		var at map[string]any
		if json.Unmarshal([]byte(atRaw), &at) == nil {
			me := strconv.FormatInt(s.MyUID, 10)
			for k, v := range at {
				name := sqlitex.Str(v)
				m.AtIDs[k] = name
				if k == me {
					m.MentionsMe = true
				}
				if name == "所有人" || name == "all" || k == "-1" {
					m.MentionsAll = true
				}
			}
		}
	}
	if strings.Contains(m.Text, "@所有人") {
		m.MentionsAll = true
	}

	m.Attachments = parseAttachments(attRaw, content)
	if withImages && (ct == CTImage || strings.Contains(m.Text, "[图片]")) {
		m.ImageInfo = s.imageInfo(cid, mid, content)
	}
	return m
}

// AgentText renders the message text with @uid replaced by names; used for
// digests consumed by LLM agents.
func (m *Message) AgentText() string {
	t := m.Text
	for uid, name := range m.AtIDs {
		if name != "" {
			t = strings.ReplaceAll(t, "@"+uid, "@"+name)
		}
	}
	var files []string
	for _, a := range m.Attachments {
		if a.Filename != "" {
			files = append(files, a.Filename)
		}
	}
	if len(files) > 0 {
		t = strings.Replace(t, "[文件]", "", 1)
		t = strings.TrimSpace(t + " [文件: " + strings.Join(files, ", ") + "]")
	}
	return strings.TrimSpace(t)
}

func asMap(v any) map[string]any {
	switch x := v.(type) {
	case map[string]any:
		return x
	case string:
		var m map[string]any
		if json.Unmarshal([]byte(x), &m) == nil {
			return m
		}
	}
	return map[string]any{}
}

func firstAttachmentExt(content map[string]any) map[string]any {
	atts, _ := content["attachments"].([]any)
	if len(atts) == 0 {
		return map[string]any{}
	}
	return asMap(asMap(atts[0])["extension"])
}

func richText(content map[string]any) string {
	ext := firstAttachmentExt(content)
	if md := sqlitex.Str(ext["markdown"]); md != "" {
		var keep []string
		for _, line := range strings.Split(md, "\n") {
			if strings.HasPrefix(line, "> ###### ") || strings.HasPrefix(line, "---") {
				continue
			}
			keep = append(keep, line)
		}
		return strings.TrimSpace(strings.Join(keep, "\n"))
	}
	if t := sqlitex.Str(ext["title"]); t != "" {
		return t
	}
	return sqlitex.Str(ext["desc"])
}

func quoteText(content map[string]any) string {
	ext := firstAttachmentExt(content)
	if d := sqlitex.Str(ext["desc"]); d != "" {
		return d
	}
	if t := sqlitex.Str(ext["title"]); t != "" {
		return t
	}
	return sqlitex.Str(content["text"])
}

func cardText(content map[string]any) string {
	ext := firstAttachmentExt(content)
	if d := sqlitex.Str(ext["searchDesc"]); d != "" {
		return d
	}
	if i18n := asMap(ext["LastMessageI18n"]); sqlitex.Str(i18n["zh_CN"]) != "" {
		return sqlitex.Str(i18n["zh_CN"])
	}
	if d := sqlitex.Str(ext["interactiveCardLastMessage"]); d != "" {
		return d
	}
	return sqlitex.Str(ext["title"])
}

func approvalText(content map[string]any) string {
	ext := firstAttachmentExt(content)
	if md := sqlitex.Str(ext["markdown"]); md != "" {
		return md
	}
	return sqlitex.Str(ext["title"])
}

func boolPtr(b bool) *bool { return &b }

func fileExists(p string) bool {
	if p == "" || strings.HasPrefix(p, `\\`) {
		return false
	}
	_, err := os.Stat(p)
	return err == nil
}

// resolveLocal picks the first existing candidate path and sets availability.
func resolveLocal(a *Attachment, candidates []string) {
	fpath := ""
	for _, p := range candidates {
		if fileExists(p) {
			fpath = p
			break
		}
	}
	if fpath == "" && len(candidates) > 0 {
		fpath = candidates[0]
	}
	if fpath != "" && !strings.HasPrefix(fpath, `\\`) {
		ok := fileExists(fpath)
		a.LocalAvailable = boolPtr(ok)
		if ok {
			a.LocalPath = fpath
		}
	} else {
		a.LocalAvailable = boolPtr(false)
	}
}

func parseAttachments(raw string, content map[string]any) []Attachment {
	out := []Attachment{}
	contentAtts, _ := content["attachments"].([]any)
	byName := map[string]string{}
	for _, ca := range contentAtts {
		cam := asMap(ca)
		ext := asMap(cam["extension"])
		name := sqlitex.Str(ext["f_name"])
		if name == "" {
			name = sqlitex.Str(cam["filename"])
		}
		if fp := sqlitex.Str(cam["filepath"]); name != "" && fp != "" {
			byName[name] = fp
		}
	}

	if raw != "" {
		var list []any
		if json.Unmarshal([]byte(raw), &list) == nil {
			for _, item := range list {
				att := asMap(item)
				ext := asMap(att["extension"])
				a := Attachment{Type: sqlitex.Int(att["type"]), URL: sqlitex.Str(att["url"]), Size: sqlitex.Int(att["size"])}
				if a.Type == CTFile || sqlitex.Str(ext["f_name"]) != "" {
					a.Filename = sqlitex.Str(ext["f_name"])
					a.FileSize = sqlitex.Int(ext["f_size"])
					a.FileType = sqlitex.Str(ext["f_type"])
					var cands []string
					for _, p := range []string{sqlitex.Str(ext["path"]), sqlitex.Str(att["filepath"]), byName[a.Filename]} {
						if p != "" {
							cands = append(cands, p)
						}
					}
					resolveLocal(&a, cands)
				}
				out = append(out, a)
			}
		}
	}

	if len(out) == 0 {
		for _, item := range contentAtts {
			att := asMap(item)
			ext := asMap(att["extension"])
			a := Attachment{Type: sqlitex.Int(att["type"]), URL: sqlitex.Str(att["url"]), Size: sqlitex.Int(att["size"])}
			if n := sqlitex.Str(ext["f_name"]); n != "" {
				a.Filename = n
				a.FileSize = sqlitex.Int(ext["f_size"])
				a.FileType = sqlitex.Str(ext["f_type"])
				var cands []string
				for _, p := range []string{sqlitex.Str(ext["path"]), sqlitex.Str(att["filepath"])} {
					if p != "" {
						cands = append(cands, p)
					}
				}
				resolveLocal(&a, cands)
			}
			if md := sqlitex.Str(ext["markdown"]); md != "" {
				a.Markdown = truncRunes(md, 500)
			}
			if d := sqlitex.Str(ext["desc"]); d != "" {
				a.Description = truncRunes(d, 500)
			}
			out = append(out, a)
		}
	}
	return out
}

// thumbSuffix matches DingTalk's resized-thumbnail URL suffix, e.g.
// "<original>.png_720x720q90.jpg_.webp".
var thumbSuffix = regexp.MustCompile(`_\d+x\d+q\d+\.[a-z]+_(\.[a-z]+)?$`)

// imageInfo resolves the locally cached images of a message. im_image_info
// can hold both a 720px thumbnail (image_type 1) and the original (type 2)
// for the same picture; originals are preferred, thumbnails are the fallback.
func (s *Snapshot) imageInfo(cid string, mid int64, content map[string]any) *ImageInfo {
	info := &ImageInfo{Images: []Image{}}
	type pair struct{ orig, thumb *Image }
	var order []string
	byBase := map[string]*pair{}
	rows, err := s.DB.Query(`SELECT url, local_path, image_type FROM im_image_info WHERE cid = ? AND mid = ?
		ORDER BY primary_key`, cid, mid)
	if err == nil {
		for rows.Next() {
			var url, lp, typ any
			rows.Scan(&url, &lp, &typ)
			p := sqlitex.Str(lp)
			st, err := os.Stat(p)
			if p == "" || err != nil || st.Size() <= 500 {
				continue
			}
			u := sqlitex.Str(url)
			isThumb := sqlitex.Int(typ) == 1 || thumbSuffix.MatchString(u)
			base := thumbSuffix.ReplaceAllString(u, "")
			if base == "" {
				base = p
			}
			pr := byBase[base]
			if pr == nil {
				pr = &pair{}
				byBase[base] = pr
				order = append(order, base)
			}
			img := &Image{FileSize: st.Size(), URL: u, Cached: true, LocalPath: p, Src: s.localURL(p)}
			if isThumb {
				pr.thumb = img
			} else {
				pr.orig = img
			}
		}
		rows.Close()
	}
	for _, b := range order {
		pr := byBase[b]
		img := pr.orig
		if img == nil {
			img = pr.thumb
		} else {
			img.IsOriginal = true
		}
		if pr.thumb != nil {
			img.ThumbSrc = pr.thumb.Src
		}
		info.Images = append(info.Images, *img)
	}
	if len(info.Images) == 0 {
		if p := sqlitex.Str(content["blurredPath"]); p != "" {
			if st, err := os.Stat(p); err == nil && st.Size() > 500 {
				info.Images = append(info.Images, Image{FileSize: st.Size(), Cached: true, LocalPath: p, Src: s.localURL(p)})
			}
		}
	}
	if len(info.Images) > 0 {
		p := info.Images[0]
		info.FileSize, info.URL, info.Cached, info.LocalPath, info.Src = p.FileSize, p.URL, true, p.LocalPath, p.Src
		info.IsOriginal = p.IsOriginal
	}
	return info
}

// AccountDir is set by the server so image URLs can be made relative to it.
var AccountDir func() string

func (s *Snapshot) localURL(p string) string {
	if AccountDir == nil {
		return ""
	}
	base := filepath.Clean(AccountDir()) + string(filepath.Separator)
	if strings.HasPrefix(strings.ToLower(p), strings.ToLower(base)) {
		return "/api/attachments/" + filepath.ToSlash(p[len(base):])
	}
	return ""
}

// FormatTS formats a millisecond timestamp in local time.
func FormatTS(ms int64) string {
	if ms == 0 {
		return ""
	}
	return time.UnixMilli(ms).Format("2006-01-02 15:04:05")
}

// cleanText drops invalid UTF-8 / lone surrogates.
func cleanText(s string) string {
	return strings.ToValidUTF8(s, "")
}

func truncRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
