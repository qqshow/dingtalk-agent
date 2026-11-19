// Package app wires the components together and exposes the operations shared
// by the HTTP API and the MCP server.
package app

import (
	"fmt"
	"time"

	"dingtalk-agent/internal/agentdb"
	"dingtalk-agent/internal/analysis"
	"dingtalk-agent/internal/config"
	"dingtalk-agent/internal/export"
	"dingtalk-agent/internal/store"
	"dingtalk-agent/internal/syncer"
)

type App struct {
	Cfg      *config.Config
	Store    *store.Store
	Syncer   *syncer.Syncer
	Agent    *agentdb.DB
	Exporter *export.Exporter
	Version  string
}

func New(cfg *config.Config, version string) (*App, error) {
	st := store.New(cfg)
	store.AccountDir = cfg.DingTalkDir
	adb, err := agentdb.Open(cfg.AgentDBPath())
	if err != nil {
		return nil, err
	}
	return &App{
		Cfg: cfg, Store: st, Syncer: syncer.New(cfg, st), Agent: adb,
		Exporter: &export.Exporter{Dir: cfg.ExportDir()}, Version: version,
	}, nil
}

func (a *App) Close() { a.Agent.Close() }

// FreshnessForAgents: agent-facing reads sync first when data is older.
const FreshnessForAgents = 5 * time.Minute

// Snapshot returns the active snapshot; with refresh it first syncs when the
// data is older than FreshnessForAgents. A missing snapshot triggers a sync.
func (a *App) Snapshot(refresh bool) (*store.Snapshot, error) {
	if refresh || a.Store.CurrentPath() == "" {
		maxAge := FreshnessForAgents
		if !refresh {
			maxAge = 100 * 365 * 24 * time.Hour
		}
		if err := a.Syncer.EnsureFresh(maxAge); err != nil {
			if a.Store.CurrentPath() == "" {
				return nil, fmt.Errorf("sync failed: %w", err)
			}
		}
	}
	return a.Store.Snapshot()
}

func (a *App) load(date, from, to string, refresh bool) (*analysis.Input, error) {
	p, err := analysis.ParsePeriod(date, from, to)
	if err != nil {
		return nil, err
	}
	if p.To.Sub(p.From) > 93*24*time.Hour {
		return nil, fmt.Errorf("period too long (max 93 days)")
	}
	s, err := a.Snapshot(refresh)
	if err != nil {
		return nil, err
	}
	return analysis.Load(s, p)
}

// ReportWithExtras is the report plus the agent-written summary and todos.
type ReportWithExtras struct {
	*analysis.Report
	Summary *agentdb.Summary `json:"saved_summary"`
	Todos   []agentdb.Todo   `json:"open_todos"`
}

func (a *App) Report(date, from, to string, refresh bool) (*ReportWithExtras, error) {
	in, err := a.load(date, from, to, refresh)
	if err != nil {
		return nil, err
	}
	r := analysis.BuildReport(in)
	sum, _ := a.Agent.GetSummary(r.Period.DateKey)
	todos, _ := a.Agent.ListTodos(agentdb.TodoFilter{Status: "open"})
	return &ReportWithExtras{Report: r, Summary: sum, Todos: todos}, nil
}

// CalendarResult lists calendar entries of a period.
type CalendarResult struct {
	Period    string        `json:"period"`
	Available bool          `json:"available"`
	Events    []store.Event `json:"events"`
}

func (a *App) Calendar(date, from, to string, includeClosed, refresh bool) (*CalendarResult, error) {
	p, err := analysis.ParsePeriod(date, from, to)
	if err != nil {
		return nil, err
	}
	s, err := a.Snapshot(refresh)
	if err != nil {
		return nil, err
	}
	evs, err := s.Events(store.EventQuery{From: p.FromMS(), To: p.ToMS(), IncludeClosed: includeClosed})
	if err != nil {
		return nil, err
	}
	return &CalendarResult{Period: p.Label, Available: s.Cal != nil, Events: evs}, nil
}

func (a *App) Digest(date, from, to string, o analysis.DigestOptions, refresh bool) (*analysis.Digest, error) {
	in, err := a.load(date, from, to, refresh)
	if err != nil {
		return nil, err
	}
	return analysis.BuildDigest(in, analysis.BuildReport(in), o), nil
}
