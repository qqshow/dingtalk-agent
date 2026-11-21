// dingtalk-agent: decrypts the local DingTalk (V2/V3) chat database and exposes
// it through an embedded web UI, a REST API and an MCP server for agents.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/qqshow/dingtalk-agent/internal/ai"
	"github.com/qqshow/dingtalk-agent/internal/analysis"
	"github.com/qqshow/dingtalk-agent/internal/app"
	"github.com/qqshow/dingtalk-agent/internal/chat"
	"github.com/qqshow/dingtalk-agent/internal/config"
	"github.com/qqshow/dingtalk-agent/internal/decrypt"
	"github.com/qqshow/dingtalk-agent/internal/desktop"
	"github.com/qqshow/dingtalk-agent/internal/mcp"
	"github.com/qqshow/dingtalk-agent/internal/update"
	"github.com/qqshow/dingtalk-agent/internal/web"
)

// version is stamped by the release build (-X main.version=1.2.3); plain
// builds report "dev" and are never prompted to update.
var version = "dev"

// guiMode: started by double-clicking (Windows) — the web UI is shown in a
// native window with a tray icon and there is no console for messages.
var guiMode bool

const appTitle = "钉钉助手"

const usage = `dingtalk-agent %s — 钉钉聊天记录（V2/V3）解密 / 浏览 / Agent 接口

用法:
  dingtalk-agent [serve] [flags]     启动网页 + REST API + MCP(HTTP /mcp)，默认命令
                                     （Windows 双击启动时以窗口 + 托盘图标运行）
  dingtalk-agent mcp [flags]         以 stdio 方式运行 MCP server（给 Claude Code / Desktop 用）
  dingtalk-agent sync [flags]        立即解密一次最新数据
  dingtalk-agent report [-date D]    输出某天的结构化统计（JSON）
  dingtalk-agent digest [-date D]    输出某天的消息摘要（Markdown）
  dingtalk-agent prompt [-date D]    输出推荐的每日总结提示词
  dingtalk-agent version

通用参数:
`

func main() {
	detached := desktop.Prepare() // before anything is printed
	args := os.Args[1:]
	cmd := "serve"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cmd, args = args[0], args[1:]
	}

	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	var o config.Overrides
	var syncMin int
	fs.StringVar(&o.DataDir, "data", "", "数据目录（默认: 程序所在目录/data）")
	fs.StringVar(&o.DingTalkDir, "dingtalk-dir", "", "钉钉账号目录，以 <hash>_v3 或 <uid>_v2 结尾（默认自动查找）")
	fs.StringVar(&o.UID, "uid", "", "真实数字 UID（默认从钉钉日志自动识别）")
	fs.StringVar(&o.Host, "host", "", "监听地址（默认 127.0.0.1）")
	fs.IntVar(&o.Port, "port", 0, "监听端口（默认 8090）")
	fs.IntVar(&syncMin, "sync-interval", -1, "自动同步间隔（分钟，0=关闭，默认 30）")
	date := fs.String("date", "", "日期 YYYY-MM-DD / today / yesterday（report/digest/prompt）")
	from := fs.String("from", "", "区间开始日期（report/digest）")
	to := fs.String("to", "", "区间结束日期（report/digest）")
	scope := fs.String("scope", "all", "digest 范围: all | involved")
	verbose := fs.Bool("v", false, "输出调试日志")
	noWindow := fs.Bool("no-window", false, "双击启动时不打开窗口和托盘图标，只在后台运行（Windows）")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), usage, version)
		fs.PrintDefaults()
	}
	fs.Parse(args)
	if syncMin >= 0 {
		o.SyncInterval, o.SyncSet = time.Duration(syncMin)*time.Minute, true
	}

	// Logs go to stderr (stdout is the MCP channel in `mcp` mode) and a file.
	level := slog.LevelInfo
	if *verbose {
		level = slog.LevelDebug
	}

	switch cmd {
	case "version", "-version", "--version":
		fmt.Println(version)
		return
	case "help", "-h", "--help":
		fs.Usage()
		return
	case "prompt":
		day, err := analysis.ParseDay(*date)
		if err != nil {
			fatal(err)
		}
		fmt.Println(strings.ReplaceAll(mcp.DailySummaryInstructions, "{{date}}", day.Format("2006-01-02")))
		return
	}

	if detached && cmd == "serve" && !*noWindow && os.Getenv("DINGTALK_AGENT_NO_WINDOW") == "" {
		guiMode = true
	}

	cfg, err := config.Load(o)
	if err != nil {
		fatal(err)
	}
	setupLogging(cfg, level, cmd)

	if err := cfg.ResolveAccount(decrypt.CheckKey); err != nil {
		fatal(err)
	}
	a, err := app.New(cfg, version)
	if err != nil {
		fatal(err)
	}
	defer a.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	switch cmd {
	case "serve":
		serve(ctx, stop, a)
	case "mcp":
		runMCP(ctx, a)
	case "sync":
		st, err := a.Syncer.SyncNow()
		if err != nil {
			fatal(err)
		}
		printJSON(st)
	case "report":
		r, err := a.Report(*date, *from, *to, true)
		if err != nil {
			fatal(err)
		}
		printJSON(r)
	case "digest":
		d, err := a.Digest(*date, *from, *to, analysis.DigestOptions{Scope: *scope, SkipBots: true}, true)
		if err != nil {
			fatal(err)
		}
		fmt.Print(d.Markdown)
	default:
		fs.Usage()
		os.Exit(2)
	}
}

func setupLogging(cfg *config.Config, level slog.Level, cmd string) {
	logDir := filepath.Join(cfg.DataDir, "logs")
	os.MkdirAll(logDir, 0o755)
	var w io.Writer = os.Stderr
	if f, err := os.OpenFile(filepath.Join(logDir, "dingtalk-agent.log"),
		os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644); err == nil {
		w = io.MultiWriter(os.Stderr, f)
		if guiMode {
			w = f // no console: writes to stderr would fail and stop MultiWriter
		}
	}
	if cmd == "report" || cmd == "digest" || cmd == "sync" {
		if level == slog.LevelInfo {
			level = slog.LevelWarn // keep CLI output clean
		}
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: level})).With("cmd", cmd))
}

func serve(ctx context.Context, stop context.CancelFunc, a *app.App) {
	cfg := a.Cfg
	m := mcp.New(a)
	runner := ai.New(a)
	upd := update.New(version)
	restartCh := make(chan struct{}, 1)
	restart := func() {
		select {
		case restartCh <- struct{}{}:
		default:
		}
	}
	handler := withDesktop(web.New(a, m, runner, upd, chat.New(a, m.Tools()), restart))
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	srv := &http.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: 10 * time.Second}

	a.Syncer.Schedule(ctx)
	runner.Schedule(ctx)
	upd.Schedule(ctx, cfg.AutoUpdateCheck)
	update.CleanupOld()

	ln, err := listen(addr)
	if err != nil {
		if guiMode && showRunningInstance(cfg.Port) {
			os.Exit(0) // already running: its window was brought to the front
		}
		fatal(fmt.Errorf("listen %s: %w (is another instance running?)", addr, err))
	}
	host := cfg.Host
	if host == "0.0.0.0" || host == "" {
		host = "localhost"
	}
	base := "http://" + net.JoinHostPort(host, strconv.Itoa(cfg.Port))
	slog.Info("server started", "web", base, "mcp", base+"/mcp", "account", cfg.DingTalkDir(), "uid", cfg.UID(),
		"sync_interval", cfg.Interval())
	fmt.Fprintf(os.Stderr, "\n  网页:      %s\n  Agent API: %s/api/agent\n  MCP:       %s/mcp\n\n", base, base, base)

	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()
	shutdown := func() {
		sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(sctx)
	}
	wait := func() {
		select {
		case <-ctx.Done():
			shutdown()
		case err := <-errc:
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				fatal(err)
			}
		case <-restartCh:
			// An update was installed: release the port and start the new version.
			shutdown()
			a.Close()
			desktop.Quit() // remove the tray icon before the new process adds its own
			fmt.Fprintln(os.Stderr, "\n  已升级到新版本，正在重启…")
			if err := update.Restart(); err != nil {
				fatal(fmt.Errorf("新版本已安装，但自动重启失败: %w。请手动重新启动程序", err))
			}
			os.Exit(0) // Windows: the new process has been started
		}
	}
	if !guiMode {
		wait()
		return
	}
	done := make(chan struct{})
	go func() { defer close(done); wait(); desktop.Quit() }()
	desktop.Run(desktop.Options{
		URL:      "http://" + net.JoinHostPort("127.0.0.1", strconv.Itoa(cfg.Port)) + "/",
		Title:    appTitle,
		StateDir: cfg.DataDir,
		OnQuit:   stop,
	})
	stop() // window and tray are gone: stop the server too
	select {
	case <-done:
	case <-time.After(8 * time.Second):
	}
}

// withDesktop adds POST /api/desktop/show, which a second instance started by
// double-clicking uses to bring this instance's window to the front.
func withDesktop(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/desktop/show" && r.Method == http.MethodPost {
			if !guiMode {
				http.Error(w, "not running as a desktop app", http.StatusConflict)
				return
			}
			desktop.Show()
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// showRunningInstance asks an instance already listening on port to show its
// window (or, if it runs in a terminal, opens its page in the browser). It
// reports whether that worked.
func showRunningInstance(port int) bool {
	base := "http://" + net.JoinHostPort("127.0.0.1", strconv.Itoa(port)) + "/"
	desktop.AllowForeground()
	c := &http.Client{Timeout: 3 * time.Second}
	resp, err := c.Post(base+"api/desktop/show", "text/plain", nil)
	if err != nil {
		return false
	}
	resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusNoContent:
		return true
	case http.StatusConflict:
		desktop.OpenBrowser(base)
		return true
	}
	return false
}

// listen binds addr. A process started by an update retries for a while,
// because the previous instance may still be releasing the port.
func listen(addr string) (net.Listener, error) {
	ln, err := net.Listen("tcp", addr)
	if err == nil || os.Getenv(update.RestartEnv) == "" {
		return ln, err
	}
	for i := 0; i < 40; i++ {
		time.Sleep(500 * time.Millisecond)
		if ln, err = net.Listen("tcp", addr); err == nil {
			return ln, nil
		}
	}
	return nil, err
}

func runMCP(ctx context.Context, a *app.App) {
	// Make sure some data exists; later calls refresh on demand.
	if a.Store.CurrentPath() == "" {
		if _, err := a.Syncer.SyncNow(); err != nil {
			slog.Error("initial sync failed", "err", err)
		}
	}
	if err := mcp.New(a).ServeStdio(ctx, os.Stdin, os.Stdout); err != nil {
		slog.Error("stdio closed", "err", err)
	}
}

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

func fatal(err error) {
	if guiMode {
		slog.Error("fatal", "err", err)
		desktop.MessageBox(appTitle+" - 无法启动", err.Error(), true)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "错误:", err)
	os.Exit(1)
}
