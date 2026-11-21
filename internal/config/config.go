// Package config resolves runtime settings: where the DingTalk (V2/V3) account data
// lives, the real numeric UID used as decryption key, and local storage paths.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// Config holds all runtime settings. Fields are safe to read after Load returns;
// UID/DingTalkDir may be updated via SetAccount (guarded by mu).
type Config struct {
	mu sync.RWMutex

	DataDir      string        // local working dir (decrypted snapshots, agent.db, exports)
	Host         string        // web listen host
	Port         int           // web listen port
	SyncInterval time.Duration // automatic decrypt interval; 0 disables

	ai              AIConfig
	dingTalkDir     string // ...\DingTalk\<hash>_v3 or <uid>_v2
	uid             string // real numeric DingTalk UID
	intervalChanged bool   // SyncInterval was changed at runtime → persist it
	aiChanged       bool   // AI settings were changed at runtime → persist them
}

// persisted is the on-disk shape of data/config.json.
type persisted struct {
	UID                 string    `json:"uid,omitempty"`
	DingTalkDir         string    `json:"dingtalk_dir,omitempty"`
	Host                string    `json:"host,omitempty"`
	Port                int       `json:"port,omitempty"`
	SyncIntervalMinutes *int      `json:"sync_interval_minutes,omitempty"`
	AI                  *AIConfig `json:"ai,omitempty"`
}

// AIConfig configures the built-in AI assistant (any OpenAI-compatible API).
type AIConfig struct {
	Enabled bool   `json:"enabled"`
	BaseURL string `json:"base_url"` // e.g. https://api.deepseek.com/v1
	Model   string `json:"model"`
	APIKey  string `json:"api_key"`
	// Scheduled runs (local time, "HH:MM"). The weekly report runs on
	// WeeklyDay (0=Sunday … 6=Saturday) at the same time, after the daily one.
	DailyEnabled  bool   `json:"daily_enabled"`
	DailyTime     string `json:"daily_time"`
	WeeklyEnabled bool   `json:"weekly_enabled"`
	WeeklyDay     int    `json:"weekly_day"`
	// MaxInputChars caps the chat digest sent to the model (default 30000).
	MaxInputChars int `json:"max_input_chars,omitempty"`
}

// DefaultAIConfig is used until the user configures the assistant.
func DefaultAIConfig() AIConfig {
	return AIConfig{DailyEnabled: true, DailyTime: "18:30", WeeklyEnabled: true, WeeklyDay: 5}
}

// Ready reports whether the assistant is enabled and fully configured.
func (a AIConfig) Ready() bool {
	return a.Enabled && strings.TrimSpace(a.BaseURL) != "" && strings.TrimSpace(a.Model) != ""
}

// Overrides come from command-line flags; zero values mean "not set".
type Overrides struct {
	DataDir      string
	DingTalkDir  string
	UID          string
	Host         string
	Port         int
	SyncInterval time.Duration
	SyncSet      bool
}

func (c *Config) UID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.uid
}

func (c *Config) DingTalkDir() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.dingTalkDir
}

// Interval returns the automatic sync interval (0 = disabled).
func (c *Config) Interval() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.SyncInterval
}

// SetInterval changes the automatic sync interval and persists it.
func (c *Config) SetInterval(d time.Duration) error {
	c.mu.Lock()
	c.SyncInterval = d
	c.intervalChanged = true
	c.mu.Unlock()
	return c.save()
}

// AI returns the assistant settings. DINGTALK_AI_API_KEY, when set, overrides
// the stored key (and is never written to config.json).
func (c *Config) AI() AIConfig {
	a := c.StoredAI()
	if v := strings.TrimSpace(os.Getenv("DINGTALK_AI_API_KEY")); v != "" {
		a.APIKey = v
	}
	return a
}

// StoredAI returns the assistant settings as stored, without env overrides.
func (c *Config) StoredAI() AIConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ai
}

// AIKeyFromEnv reports whether DINGTALK_AI_API_KEY provides the API key.
func AIKeyFromEnv() bool { return strings.TrimSpace(os.Getenv("DINGTALK_AI_API_KEY")) != "" }

// SetAI replaces the assistant settings and persists them.
func (c *Config) SetAI(a AIConfig) error {
	c.mu.Lock()
	c.ai = a
	c.aiChanged = true
	c.mu.Unlock()
	return c.save()
}

// SetAccount updates the active account and persists it to config.json.
func (c *Config) SetAccount(dir, uid string) error {
	c.mu.Lock()
	c.dingTalkDir, c.uid = dir, uid
	c.mu.Unlock()
	return c.save()
}

func (c *Config) DecryptedDir() string { return filepath.Join(c.DataDir, "decrypted") }
func (c *Config) ExportDir() string    { return filepath.Join(c.DataDir, "exports") }
func (c *Config) AgentDBPath() string  { return filepath.Join(c.DataDir, "agent.db") }
func (c *Config) StatePath() string    { return filepath.Join(c.DataDir, "sync_state.json") }
func (c *Config) configPath() string   { return filepath.Join(c.DataDir, "config.json") }

// EncryptedDB is the DingTalk client's (encrypted) message database.
func (c *Config) EncryptedDB() string {
	return filepath.Join(c.DingTalkDir(), "DBFiles", "dingtalk.db")
}

// DefaultDataDir returns <exe dir>/data, or ./data when running via `go run`.
func DefaultDataDir() string {
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(exe)
		tmp := os.TempDir()
		if !strings.HasPrefix(strings.ToLower(dir), strings.ToLower(tmp)) &&
			!strings.Contains(strings.ToLower(dir), "go-build") {
			return filepath.Join(dir, "data")
		}
	}
	wd, _ := os.Getwd()
	return filepath.Join(wd, "data")
}

// Load builds the config: defaults < data/config.json < env vars < flags.
// It does not resolve the account; call ResolveAccount for that.
func Load(o Overrides) (*Config, error) {
	c := &Config{
		DataDir:      o.DataDir,
		Host:         "127.0.0.1",
		Port:         8090,
		SyncInterval: 30 * time.Minute,
		ai:           DefaultAIConfig(),
	}
	if c.DataDir == "" {
		c.DataDir = DefaultDataDir()
	}
	for _, d := range []string{c.DataDir, c.DecryptedDir(), c.ExportDir()} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, err
		}
	}

	var p persisted
	if b, err := os.ReadFile(c.configPath()); err == nil {
		if err := json.Unmarshal(b, &p); err != nil {
			return nil, fmt.Errorf("parse %s: %w", c.configPath(), err)
		}
	}
	c.uid, c.dingTalkDir = p.UID, p.DingTalkDir
	if p.Host != "" {
		c.Host = p.Host
	}
	if p.Port != 0 {
		c.Port = p.Port
	}
	if p.SyncIntervalMinutes != nil {
		c.SyncInterval = time.Duration(*p.SyncIntervalMinutes) * time.Minute
	}
	if p.AI != nil {
		c.ai = *p.AI
	}

	if v := strings.TrimSpace(os.Getenv("DINGTALK_UID")); v != "" {
		c.uid = v
	}
	if v := strings.TrimSpace(os.Getenv("DINGTALK_DATA_DIR")); v != "" {
		c.dingTalkDir = v
	}

	if o.UID != "" {
		c.uid = o.UID
	}
	if o.DingTalkDir != "" {
		c.dingTalkDir = o.DingTalkDir
	}
	if o.Host != "" {
		c.Host = o.Host
	}
	if o.Port != 0 {
		c.Port = o.Port
	}
	if o.SyncSet {
		c.SyncInterval = o.SyncInterval
	}
	return c, nil
}

func (c *Config) save() error {
	c.mu.RLock()
	p := persisted{UID: c.uid, DingTalkDir: c.dingTalkDir}
	changed, minutes := c.intervalChanged, int(c.SyncInterval/time.Minute)
	aiChanged, ai := c.aiChanged, c.ai
	c.mu.RUnlock()
	// Preserve fields we do not manage here.
	if b, err := os.ReadFile(c.configPath()); err == nil {
		var old persisted
		if json.Unmarshal(b, &old) == nil {
			p.Host, p.Port, p.SyncIntervalMinutes, p.AI = old.Host, old.Port, old.SyncIntervalMinutes, old.AI
		}
	}
	if changed {
		p.SyncIntervalMinutes = &minutes
	}
	if aiChanged {
		p.AI = &ai
	}
	b, _ := json.MarshalIndent(p, "", "  ")
	tmp := c.configPath() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, c.configPath())
}

// ---------------------------------------------------------------------------
// Account discovery
// ---------------------------------------------------------------------------

// Account is a local DingTalk account directory (<hash>_v3 or <uid>_v2).
type Account struct {
	Dir      string    `json:"data_dir"`
	DirName  string    `json:"dir_name"`
	Version  string    `json:"version"` // "v2" or "v3"
	Modified time.Time `json:"last_modified"`
}

// SearchBases lists candidate DingTalk data roots for this OS.
func SearchBases() []string {
	var bases []string
	add := func(parts ...string) {
		if parts[0] != "" {
			bases = append(bases, filepath.Join(parts...))
		}
	}
	add(os.Getenv("APPDATA"), "DingTalk")
	add(os.Getenv("LOCALAPPDATA"), "DingTalk")
	if home, err := os.UserHomeDir(); err == nil {
		add(home, "AppData", "Roaming", "DingTalk")
		add(home, "Library", "Application Support", "DingTalk")
		// macOS: the client stores data under "DingTalkMac", and the sandboxed
		// build keeps it inside its container (<TeamID>.com.dingtalk.mac).
		add(home, "Library", "Application Support", "DingTalkMac")
		containers, _ := filepath.Glob(filepath.Join(home, "Library", "Containers", "*com.dingtalk.mac"))
		for _, c := range containers {
			add(c, "Data", "Library", "Application Support", "DingTalkMac")
		}
		add(home, ".config", "DingTalk")
	}
	seen := map[string]bool{}
	var out []string
	for _, b := range bases {
		k := strings.ToLower(filepath.Clean(b))
		if !seen[k] {
			seen[k] = true
			out = append(out, b)
		}
	}
	return out
}

// DiscoverAccounts finds *_v3 / *_v2 account dirs that contain DBFiles/dingtalk.db,
// most recently modified first.
func DiscoverAccounts() []Account {
	var out []Account
	for _, base := range SearchBases() {
		entries, err := os.ReadDir(base)
		if err != nil {
			continue
		}
		for _, e := range entries {
			ver := AccountVersion(e.Name())
			if !e.IsDir() || ver == "" {
				continue
			}
			dir := filepath.Join(base, e.Name())
			db := filepath.Join(dir, "DBFiles", "dingtalk.db")
			st, err := os.Stat(db)
			if err != nil {
				continue
			}
			mod := st.ModTime()
			if w, err := os.Stat(db + "-wal"); err == nil && w.ModTime().After(mod) {
				mod = w.ModTime()
			}
			out = append(out, Account{Dir: dir, DirName: e.Name(), Version: ver, Modified: mod})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Modified.After(out[j].Modified) })
	return out
}

// AccountVersion returns "v3" / "v2" for an account directory name, or "".
func AccountVersion(dirName string) string {
	n := strings.ToLower(filepath.Base(filepath.Clean(dirName)))
	switch {
	case strings.HasSuffix(n, "_v3"):
		return "v3"
	case strings.HasSuffix(n, "_v2"):
		return "v2"
	}
	return ""
}

// dirUID returns the numeric UID a V2 directory is named after (<uid>_v2).
func dirUID(dir string) string {
	n := filepath.Base(filepath.Clean(dir))
	if AccountVersion(n) != "v2" {
		return ""
	}
	id := n[:len(n)-3]
	for _, r := range id {
		if r < '0' || r > '9' {
			return ""
		}
	}
	return id
}

var uidPatterns = []struct {
	re     *regexp.Regexp
	weight int
}{
	{regexp.MustCompile(`real_uid=(\d{6,14})`), 10000},
	{regexp.MustCompile(`myOpenId=(\d{6,14})`), 10000},
	{regexp.MustCompile(`[?&]uid=(\d{6,14})`), 10},
	{regexp.MustCompile(`&cid=(\d{6,14}):`), 1},
}

// UIDCandidates scans recent DingTalk client logs for likely real UIDs,
// best candidates first.
func UIDCandidates(dingTalkBase string) []string {
	logDir := filepath.Join(dingTalkBase, "log")
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return nil
	}
	var cef, gaea []string
	for _, e := range entries {
		n := e.Name()
		switch {
		case strings.HasPrefix(n, "cef_debug.log"):
			cef = append(cef, n)
		case strings.HasPrefix(n, "gaea.log"):
			gaea = append(gaea, n)
		}
	}
	sort.Strings(cef)
	sort.Strings(gaea)
	if len(cef) > 12 {
		cef = cef[len(cef)-12:]
	}
	if len(gaea) > 3 {
		gaea = gaea[len(gaea)-3:]
	}
	scores := map[string]int{}
	for _, n := range append(cef, gaea...) {
		b, err := os.ReadFile(filepath.Join(logDir, n))
		if err != nil {
			continue
		}
		s := string(b)
		for _, p := range uidPatterns {
			for _, m := range p.re.FindAllStringSubmatch(s, -1) {
				scores[m[1]] += p.weight
			}
		}
	}
	type kv struct {
		k string
		v int
	}
	var list []kv
	for k, v := range scores {
		list = append(list, kv{k, v})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].v != list[j].v {
			return list[i].v > list[j].v
		}
		return list[i].k < list[j].k
	})
	var out []string
	for i, x := range list {
		if i >= 30 {
			break
		}
		out = append(out, x.k)
	}
	return out
}

// KeyChecker reports whether uid decrypts the database in accountDir.
type KeyChecker func(accountDir, uid string) bool

// ResolveAccount makes sure DingTalkDir and UID are set and valid. When the
// configured UID does not decrypt the database it searches the client logs
// for the real one and persists it.
func (c *Config) ResolveAccount(check KeyChecker) error {
	dir := c.DingTalkDir()
	if dir == "" {
		accts := DiscoverAccounts()
		if len(accts) == 0 {
			return errors.New("no DingTalk account found (looked for *_v3 or *_v2/DBFiles/dingtalk.db); " +
				"open the DingTalk desktop client once, or set dingtalk_dir in data/config.json")
		}
		dir = accts[0].Dir
		if len(accts) > 1 {
			slog.Warn("multiple DingTalk accounts found, using most recent", "dir", dir)
		}
	}
	uid := c.UID()
	if uid != "" && check(dir, uid) {
		if dir != c.DingTalkDir() {
			return c.SetAccount(dir, uid)
		}
		return nil
	}
	if uid != "" {
		slog.Warn("configured UID does not decrypt the database, searching logs", "uid", uid)
	}
	cands := UIDCandidates(filepath.Dir(dir))
	if id := dirUID(dir); id != "" {
		cands = append([]string{id}, cands...) // V2 dirs are named after the UID
	}
	for _, cand := range cands {
		if check(dir, cand) {
			slog.Info("resolved real DingTalk UID from client logs", "uid", cand)
			return c.SetAccount(dir, cand)
		}
	}
	return fmt.Errorf("cannot determine the real DingTalk UID for %s (tried %d candidates from logs); "+
		"set it with -uid <number> or \"uid\" in data/config.json", dir, len(cands))
}
