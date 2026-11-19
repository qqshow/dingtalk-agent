// Package syncer runs decryption (on demand and on a schedule) and keeps a
// small status record the web UI and agents can poll.
package syncer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/qqshow/dingtalk-agent/internal/config"
	"github.com/qqshow/dingtalk-agent/internal/decrypt"
	"github.com/qqshow/dingtalk-agent/internal/store"
)

// State is persisted to data/sync_state.json. Field names match the
// original web UI.
type State struct {
	IsSyncing        bool   `json:"is_syncing"`
	Operation        string `json:"operation,omitempty"`
	LastSyncTime     int64  `json:"last_sync_time"` // latest message time (ms) in the snapshot
	LastSyncTimeStr  string `json:"last_sync_time_str"`
	LastError        string `json:"last_error"`
	SyncCount        int    `json:"sync_count"`
	LastDurationMS   int64  `json:"last_duration_ms"`
	LastExportPath   string `json:"last_export_path"`
	NextSyncTime     string `json:"next_sync_time"`
	SnapshotFile     string `json:"snapshot_file"`
	LatestMessageStr string `json:"latest_message_str"`
}

var ErrBusy = errors.New("another sync/export is in progress")

type Syncer struct {
	cfg   *config.Config
	store *store.Store

	mu    sync.Mutex
	state State
	reset chan struct{}
}

func New(cfg *config.Config, st *store.Store) *Syncer {
	s := &Syncer{cfg: cfg, store: st, reset: make(chan struct{}, 1)}
	if b, err := os.ReadFile(cfg.StatePath()); err == nil {
		json.Unmarshal(b, &s.state)
	}
	s.state.IsSyncing, s.state.Operation = false, ""
	s.state.NextSyncTime = "" // set by the scheduler once it is armed
	return s
}

func (s *Syncer) State() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Another process (stdio MCP) may have synced; refresh from disk when idle.
	if !s.state.IsSyncing {
		if b, err := os.ReadFile(s.cfg.StatePath()); err == nil {
			var disk State
			if json.Unmarshal(b, &disk) == nil && disk.SyncCount > s.state.SyncCount {
				next := s.state.NextSyncTime
				s.state = disk
				s.state.NextSyncTime = next
				s.state.IsSyncing = false
			}
		}
	}
	return s.state
}

func (s *Syncer) save() {
	b, _ := json.MarshalIndent(s.state, "", "  ")
	tmp := s.cfg.StatePath() + ".tmp"
	if os.WriteFile(tmp, b, 0o644) == nil {
		os.Rename(tmp, s.cfg.StatePath())
	}
}

// Begin reserves the syncer for an operation (sync / export).
func (s *Syncer) Begin(op string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state.IsSyncing {
		return false
	}
	s.state.IsSyncing, s.state.Operation, s.state.LastError = true, op, ""
	s.save()
	return true
}

// Finish releases the reservation, recording err if any.
func (s *Syncer) Finish(err error, exportPath string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.IsSyncing, s.state.Operation = false, ""
	if err != nil {
		s.state.LastError = err.Error()
	}
	if exportPath != "" {
		s.state.LastExportPath = exportPath
	}
	s.save()
}

// lockFile provides cross-process mutual exclusion for decryption.
func (s *Syncer) lockFile() (func(), error) {
	p := filepath.Join(s.cfg.DataDir, "sync.lock")
	for i := 0; i < 2; i++ {
		f, err := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			fmt.Fprintf(f, "%d", os.Getpid())
			f.Close()
			return func() { os.Remove(p) }, nil
		}
		if st, serr := os.Stat(p); serr == nil && time.Since(st.ModTime()) > 10*time.Minute {
			os.Remove(p) // stale lock
			continue
		}
		return nil, ErrBusy
	}
	return nil, ErrBusy
}

// SyncNow decrypts synchronously (reserving the syncer itself).
func (s *Syncer) SyncNow() (*State, error) {
	if !s.Begin("manual_sync") {
		return nil, ErrBusy
	}
	err := s.run()
	s.Finish(err, "")
	st := s.State()
	return &st, err
}

// RunReserved performs a sync when the caller already called Begin.
func (s *Syncer) RunReserved() error {
	err := s.run()
	s.Finish(err, "")
	return err
}

func (s *Syncer) run() error {
	unlock, err := s.lockFile()
	if err != nil {
		return err
	}
	defer unlock()
	start := time.Now()
	if err := s.cfg.ResolveAccount(decrypt.CheckKey); err != nil {
		return err
	}
	res, err := decrypt.BuildSnapshot(s.cfg.DingTalkDir(), s.cfg.UID(), s.cfg.DecryptedDir())
	if err != nil {
		slog.Error("sync failed", "err", err)
		return err
	}
	if err := s.store.Publish(res.Path); err != nil {
		return err
	}
	snap, err := s.store.Snapshot()
	if err != nil {
		return err
	}
	stats := snap.Stats()
	s.mu.Lock()
	s.state.LastSyncTime = stats.LatestMessageTime
	s.state.LastSyncTimeStr = time.Now().Format(time.DateTime)
	s.state.LatestMessageStr = stats.LatestMessageStr
	s.state.SyncCount++
	s.state.LastDurationMS = time.Since(start).Milliseconds()
	s.state.SnapshotFile = filepath.Base(res.Path)
	s.mu.Unlock()
	slog.Info("sync complete", "duration", time.Since(start).Round(time.Millisecond),
		"wal_frames", res.WALFrames, "latest_message", stats.LatestMessageStr)
	return nil
}

// EnsureFresh syncs if there is no snapshot or it is older than maxAge.
func (s *Syncer) EnsureFresh(maxAge time.Duration) error {
	p := s.store.CurrentPath()
	if p != "" {
		if st, err := os.Stat(p); err == nil && time.Since(st.ModTime()) < maxAge {
			return nil
		}
	}
	_, err := s.SyncNow()
	if errors.Is(err, ErrBusy) {
		// Someone else is syncing; wait briefly for them.
		for i := 0; i < 30 && s.State().IsSyncing; i++ {
			time.Sleep(time.Second)
		}
		return nil
	}
	return err
}

// Schedule runs syncs every cfg.Interval() until ctx is done. The first run
// happens immediately when the snapshot is missing or stale. SetInterval
// re-arms the timer without a restart.
func (s *Syncer) Schedule(ctx context.Context) {
	go func() {
		iv := s.cfg.Interval()
		maxAge := iv
		if iv <= 0 {
			maxAge = 100 * 365 * 24 * time.Hour // disabled: only sync when there is no data
		}
		if err := s.EnsureFresh(maxAge); err != nil {
			slog.Error("initial sync failed", "err", err)
		}
		for {
			iv := s.cfg.Interval()
			var tick <-chan time.Time
			var timer *time.Timer
			next := ""
			if iv > 0 {
				timer = time.NewTimer(iv)
				tick = timer.C
				next = time.Now().Add(iv).Format(time.DateTime)
			}
			s.mu.Lock()
			s.state.NextSyncTime = next
			s.mu.Unlock()
			select {
			case <-ctx.Done():
				if timer != nil {
					timer.Stop()
				}
				return
			case <-s.reset:
				if timer != nil {
					timer.Stop()
				}
			case <-tick:
				if _, err := s.SyncNow(); err != nil && !errors.Is(err, ErrBusy) {
					slog.Error("scheduled sync failed", "err", err)
				}
			}
		}
	}()
}

// SetInterval changes the automatic sync interval at runtime and persists it.
func (s *Syncer) SetInterval(d time.Duration) error {
	if err := s.cfg.SetInterval(d); err != nil {
		return err
	}
	slog.Info("sync interval changed", "interval", d)
	select {
	case s.reset <- struct{}{}:
	default: // scheduler not running (e.g. stdio mode) or a reset is already pending
	}
	return nil
}
