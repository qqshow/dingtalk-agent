// Package update checks GitHub releases for a newer version and replaces the
// running executable with it.
//
// Flow: Check → (user confirms) → Apply: download the platform archive and
// SHA256SUMS.txt, verify the checksum, extract the program next to the
// current one, make sure it runs, swap the files (the old one is kept as
// <exe>.old until the next start), then the caller restarts the process.
package update

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Repo is the GitHub repository releases are published to.
const Repo = "qqshow/dingtalk-agent"

// RestartEnv is set for the process started by Restart, so it knows to wait
// for the previous instance to release the port.
const RestartEnv = "DINGTALK_AGENT_RESTARTED"

// Schedule checks for updates shortly after start and then every 12 hours
// while enabled() returns true. Failures are recorded in the status only.
func (u *Updater) Schedule(ctx context.Context, enabled func() bool) {
	if !IsRelease(u.Current) {
		return // development builds are never prompted
	}
	go func() {
		t := time.NewTimer(30 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
			}
			if enabled() {
				u.Check(ctx)
			}
			t.Reset(12 * time.Hour)
		}
	}()
}

// Release is the subset of the GitHub release we use.
type Release struct {
	Tag         string  `json:"tag_name"`
	Name        string  `json:"name"`
	Notes       string  `json:"body"`
	URL         string  `json:"html_url"`
	PublishedAt string  `json:"published_at"`
	Draft       bool    `json:"draft"`
	Prerelease  bool    `json:"prerelease"`
	Assets      []Asset `json:"assets"`
}

type Asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
	Size int64  `json:"size"`
}

// Version returns the release version without the leading "v".
func (r *Release) Version() string { return strings.TrimPrefix(r.Tag, "v") }

func (r *Release) asset(name string) *Asset {
	for i := range r.Assets {
		if r.Assets[i].Name == name {
			return &r.Assets[i]
		}
	}
	return nil
}

// ---------------------------------------------------------------- versions

type semver struct {
	major, minor, patch int
	pre                 string // "" for a final release
}

func parseVersion(v string) (semver, bool) {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	var s semver
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		if v[i] == '-' {
			s.pre = v[i+1:]
			if j := strings.IndexByte(s.pre, '+'); j >= 0 {
				s.pre = s.pre[:j]
			}
		}
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return s, false
	}
	nums := make([]int, 3)
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return s, false
		}
		nums[i] = n
	}
	s.major, s.minor, s.patch = nums[0], nums[1], nums[2]
	return s, true
}

// Newer reports whether version latest is newer than current. A current
// version that is not semver (e.g. "dev-abc1234") is never considered
// outdated, so development builds are not prompted.
func Newer(latest, current string) bool {
	l, ok1 := parseVersion(latest)
	c, ok2 := parseVersion(current)
	if !ok1 || !ok2 {
		return false
	}
	if l.major != c.major {
		return l.major > c.major
	}
	if l.minor != c.minor {
		return l.minor > c.minor
	}
	if l.patch != c.patch {
		return l.patch > c.patch
	}
	// 1.2.0 > 1.2.0-dev; prerelease ordering beyond that does not matter here.
	return l.pre == "" && c.pre != ""
}

// IsRelease reports whether v is a release version (x.y.z, optionally with a
// prerelease suffix) rather than a development build.
func IsRelease(v string) bool { _, ok := parseVersion(v); return ok }

// AssetName is the release archive for a platform, matching the CI workflow.
func AssetName(goos, goarch string) string {
	ext := ".tar.gz"
	if goos == "windows" {
		ext = ".zip"
	}
	return fmt.Sprintf("dingtalk-agent-%s-%s%s", goos, goarch, ext)
}

func binName(goos string) string {
	if goos == "windows" {
		return "dingtalk-agent.exe"
	}
	return "dingtalk-agent"
}

// ------------------------------------------------------------------ status

// Status is what the web UI shows.
type Status struct {
	Current    string   `json:"current"`
	Latest     *Release `json:"latest,omitempty"`
	Available  bool     `json:"available"`
	Supported  bool     `json:"supported"` // this platform/build can self-update
	Reason     string   `json:"reason,omitempty"`
	CheckedAt  string   `json:"checked_at,omitempty"`
	CheckError string   `json:"check_error,omitempty"`
	Applying   bool     `json:"applying"`
	Stage      string   `json:"stage,omitempty"`
	Downloaded int64    `json:"downloaded"`
	Total      int64    `json:"total"`
	ApplyError string   `json:"apply_error,omitempty"`
	Restarting bool     `json:"restarting"`
}

// Updater checks for and applies updates. It is safe for concurrent use.
type Updater struct {
	Current string
	API     string // releases API base, overridable in tests
	HTTP    *http.Client
	Exe     string // path of the running executable ("" = os.Executable)
	GOOS    string
	GOARCH  string
	// Verify checks the extracted program before it is swapped in
	// (default: run "<bin> version" and compare). Tests override it.
	Verify func(ctx context.Context, bin, want string) error

	mu sync.Mutex
	st Status
}

func New(current string) *Updater {
	u := &Updater{Current: current, API: "https://api.github.com/repos/" + Repo,
		HTTP: &http.Client{Timeout: 15 * time.Minute}, GOOS: runtime.GOOS, GOARCH: runtime.GOARCH}
	if v := strings.TrimRight(os.Getenv("DINGTALK_UPDATE_API"), "/"); v != "" {
		u.API = v // for testing against a local release server
	}
	u.st.Current = current
	return u
}

func (u *Updater) Status() Status {
	u.mu.Lock()
	defer u.mu.Unlock()
	s := u.st
	s.Supported, s.Reason = u.supported()
	return s
}

func (u *Updater) set(f func(*Status)) {
	u.mu.Lock()
	f(&u.st)
	u.mu.Unlock()
}

// supported reports whether this build can replace itself.
func (u *Updater) supported() (bool, string) {
	if !IsRelease(u.Current) {
		return false, "开发版本不支持自动升级"
	}
	if u.GOOS != "windows" && u.GOOS != "darwin" {
		return false, "该系统没有发布安装包，请从源码编译"
	}
	exe, err := u.exePath()
	if err != nil {
		return false, err.Error()
	}
	lower := strings.ToLower(exe)
	if strings.Contains(lower, "go-build") {
		return false, "通过 go run 运行时不支持自动升级"
	}
	return true, ""
}

func (u *Updater) exePath() (string, error) {
	exe := u.Exe
	if exe == "" {
		var err error
		if exe, err = os.Executable(); err != nil {
			return "", fmt.Errorf("找不到程序路径: %w", err)
		}
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	return exe, nil
}

func (u *Updater) get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "dingtalk-agent/"+u.Current)
	if strings.HasPrefix(url, u.API) {
		req.Header.Set("Accept", "application/vnd.github+json")
	}
	resp, err := u.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		resp.Body.Close()
		return nil, fmt.Errorf("%s 返回 %d: %s", url, resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return resp, nil
}

// Check fetches the latest release and records whether it is newer.
func (u *Updater) Check(ctx context.Context) (Status, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	rel, err := u.latest(ctx)
	now := time.Now().Format(time.DateTime)
	u.set(func(s *Status) {
		s.CheckedAt = now
		if err != nil {
			s.CheckError = "检查更新失败: " + err.Error()
			return
		}
		s.CheckError = ""
		s.Latest = rel
		s.Available = Newer(rel.Version(), u.Current)
	})
	return u.Status(), err
}

func (u *Updater) latest(ctx context.Context) (*Release, error) {
	resp, err := u.get(ctx, u.API+"/releases/latest")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var rel Release
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&rel); err != nil {
		return nil, fmt.Errorf("解析版本信息失败: %w", err)
	}
	if rel.Tag == "" {
		return nil, errors.New("没有找到已发布的版本")
	}
	rel.Notes = trimNotes(rel.Notes)
	return &rel, nil
}

func trimNotes(s string) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) > 4000 {
		return string(r[:4000]) + "…"
	}
	return string(r)
}

// ------------------------------------------------------------------- apply

// Apply downloads, verifies and installs the latest release found by Check.
// On success the new program is in place and the caller should restart.
func (u *Updater) Apply(ctx context.Context) (err error) {
	u.mu.Lock()
	if u.st.Applying || u.st.Restarting {
		u.mu.Unlock()
		return errors.New("正在升级中")
	}
	rel := u.st.Latest
	if rel == nil || !u.st.Available {
		u.mu.Unlock()
		return errors.New("没有可用的新版本，请先检查更新")
	}
	u.st.Applying, u.st.ApplyError, u.st.Downloaded, u.st.Total = true, "", 0, 0
	u.mu.Unlock()
	defer func() {
		u.set(func(s *Status) {
			s.Applying = false
			s.Stage = ""
			if err != nil {
				s.ApplyError = err.Error()
			}
		})
	}()
	if ok, reason := u.supported(); !ok {
		return errors.New(reason)
	}
	return u.install(ctx, rel)
}

func (u *Updater) stage(s string) { u.set(func(st *Status) { st.Stage = s }) }

func (u *Updater) install(ctx context.Context, rel *Release) error {
	exe, err := u.exePath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(exe)
	name := AssetName(u.GOOS, u.GOARCH)
	asset, sums := rel.asset(name), rel.asset("SHA256SUMS.txt")
	if asset == nil {
		return fmt.Errorf("新版本 %s 中没有适用于 %s/%s 的安装包", rel.Tag, u.GOOS, u.GOARCH)
	}
	if sums == nil {
		return fmt.Errorf("新版本 %s 中没有校验文件 SHA256SUMS.txt", rel.Tag)
	}

	u.stage("获取校验信息")
	want, err := u.checksum(ctx, sums.URL, name)
	if err != nil {
		return err
	}

	u.stage("下载新版本")
	archive, err := os.CreateTemp(dir, ".dingtalk-agent-download-*")
	if err != nil {
		return fmt.Errorf("程序目录不可写，无法自动升级（%v）。请手动下载新版本", err)
	}
	defer os.Remove(archive.Name())
	defer archive.Close()
	got, err := u.download(ctx, asset, archive)
	if err != nil {
		return err
	}
	if !strings.EqualFold(got, want) {
		return fmt.Errorf("安装包校验失败（SHA256 不一致），已取消升级")
	}

	u.stage("解压")
	newPath := exe + ".new"
	os.Remove(newPath)
	if err := extract(archive.Name(), name, binName(u.GOOS), newPath); err != nil {
		removeRetry(newPath)
		return err
	}

	u.stage("检查新版本")
	verify := u.Verify
	if verify == nil {
		verify = verifyBinary
	}
	if err := verify(ctx, newPath, rel.Version()); err != nil {
		removeRetry(newPath)
		return err
	}

	u.stage("替换程序")
	if err := swap(exe, newPath); err != nil {
		removeRetry(newPath)
		return err
	}
	u.set(func(s *Status) { s.Restarting = true; s.Stage = "正在重启" })
	return nil
}

// checksum returns the expected SHA256 of file from a SHA256SUMS.txt.
func (u *Updater) checksum(ctx context.Context, url, file string) (string, error) {
	resp, err := u.get(ctx, url)
	if err != nil {
		return "", fmt.Errorf("下载校验文件失败: %w", err)
	}
	defer resp.Body.Close()
	sc := bufio.NewScanner(io.LimitReader(resp.Body, 1<<20))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) == 2 && strings.TrimPrefix(f[1], "*") == file && len(f[0]) == 64 {
			return f[0], nil
		}
	}
	return "", fmt.Errorf("校验文件中没有 %s", file)
}

type progressWriter struct {
	u *Updater
	n int64
}

func (p *progressWriter) Write(b []byte) (int, error) {
	p.n += int64(len(b))
	n := p.n
	p.u.set(func(s *Status) { s.Downloaded = n })
	return len(b), nil
}

func (u *Updater) download(ctx context.Context, a *Asset, dst *os.File) (string, error) {
	resp, err := u.get(ctx, a.URL)
	if err != nil {
		return "", fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()
	total := resp.ContentLength
	if total <= 0 {
		total = a.Size
	}
	u.set(func(s *Status) { s.Total = total })
	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(dst, h, &progressWriter{u: u}), io.LimitReader(resp.Body, 512<<20)); err != nil {
		return "", fmt.Errorf("下载中断: %w", err)
	}
	if err := dst.Sync(); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// extract writes the program file from the archive to dst.
func extract(archivePath, archiveName, bin, dst string) error {
	var src io.Reader
	var closeFn func()
	if strings.HasSuffix(archiveName, ".zip") {
		zr, err := zip.OpenReader(archivePath)
		if err != nil {
			return fmt.Errorf("安装包损坏: %w", err)
		}
		defer zr.Close()
		for _, f := range zr.File {
			if path.Base(f.Name) == bin && !f.FileInfo().IsDir() {
				rc, err := f.Open()
				if err != nil {
					return err
				}
				src, closeFn = rc, func() { rc.Close() }
				break
			}
		}
	} else {
		f, err := os.Open(archivePath)
		if err != nil {
			return err
		}
		defer f.Close()
		gz, err := gzip.NewReader(f)
		if err != nil {
			return fmt.Errorf("安装包损坏: %w", err)
		}
		defer gz.Close()
		tr := tar.NewReader(gz)
		for {
			h, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("安装包损坏: %w", err)
			}
			if h.Typeflag == tar.TypeReg && path.Base(h.Name) == bin {
				src, closeFn = tr, func() {}
				break
			}
		}
	}
	if src == nil {
		return fmt.Errorf("安装包中没有 %s", bin)
	}
	defer closeFn()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, io.LimitReader(src, 512<<20)); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// verifyBinary runs "<bin> version" and checks it reports the new version,
// which catches corrupt files, wrong architectures and files emptied by
// security software.
func verifyBinary(ctx context.Context, bin, want string) error {
	st, err := os.Stat(bin)
	if err != nil || st.Size() < 1<<20 {
		return errors.New("新版本程序文件不完整（可能被杀毒软件拦截），已取消升级")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, bin, "version").Output()
	if err != nil {
		return fmt.Errorf("新版本程序无法运行（%v），可能被杀毒软件拦截，已取消升级", err)
	}
	if got := strings.TrimSpace(string(out)); got != want {
		return fmt.Errorf("新版本程序报告的版本是 %q，预期 %q，已取消升级", got, want)
	}
	return nil
}

// swap installs newPath as exe: exe is moved to exe.old (renaming a running
// program is allowed), then newPath's content is COPIED to exe.
//
// Copying instead of renaming matters on Windows: security software
// (Defender, 360, Huorong, ...) keeps a just-written or just-executed program
// open while scanning it. Its handle allows reading but not deleting, so a
// rename of newPath fails with "being used by another process" for as long
// as the scan lasts, while reading it works. newPath is removed afterwards,
// in the background if it is still locked.
func swap(exe, newPath string) error {
	old := exe + ".old"
	removeRetry(old)
	if err := renameRetry(exe, old); err != nil {
		return fmt.Errorf("无法替换程序文件: %w", err)
	}
	if err := copyFile(newPath, exe); err != nil {
		os.Remove(exe)
		if rerr := renameRetry(old, exe); rerr != nil {
			return fmt.Errorf("无法写入新版本（%v），恢复旧版本也失败（%v）：请把 %s 改名为 %s", err, rerr,
				filepath.Base(old), filepath.Base(exe))
		}
		return fmt.Errorf("无法写入新版本: %w", err)
	}
	if err := os.Remove(newPath); err != nil && !os.IsNotExist(err) {
		go removeRetry(newPath) // still being scanned; CleanupOld also retries at next start
	}
	return nil
}

// copyFile copies src to a new file dst (mode 0755), retrying the open of
// src briefly in case it is locked exclusively.
func copyFile(src, dst string) error {
	var in *os.File
	var err error
	for i := 0; i < 40; i++ {
		if in, err = os.Open(src); err == nil {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Sync(); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// renameRetry renames, retrying for up to ~20 seconds. On Windows a file that
// was just written or executed is often held open briefly by antivirus
// scanning (Defender), which makes the rename fail with a sharing violation.
func renameRetry(from, to string) error {
	var err error
	for i := 0; i < 80; i++ {
		if err = os.Rename(from, to); err == nil {
			return nil
		}
		if os.IsNotExist(err) {
			return err
		}
		time.Sleep(250 * time.Millisecond)
	}
	return err
}

// removeRetry deletes path, retrying briefly: on Windows a program that was
// just executed (or is being scanned by antivirus) stays locked for a moment.
func removeRetry(path string) {
	for i := 0; i < 25; i++ {
		if err := os.Remove(path); err == nil || os.IsNotExist(err) {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// CleanupOld removes files left by a previous update: the old version
// (<exe>.old, whose process may still be exiting) and an interrupted
// download (<exe>.new).
func CleanupOld() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	for _, p := range []string{exe + ".old", exe + ".new"} {
		if _, err := os.Stat(p); err != nil {
			continue
		}
		go func(p string) {
			for i := 0; i < 30; i++ {
				if err := os.Remove(p); err == nil || os.IsNotExist(err) {
					return
				}
				time.Sleep(2 * time.Second)
			}
		}(p)
	}
}
