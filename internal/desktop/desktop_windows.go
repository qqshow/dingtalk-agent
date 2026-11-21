//go:build windows

package desktop

import (
	_ "embed"
	"encoding/binary"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"fyne.io/systray"
	webview2 "github.com/jchv/go-webview2"
	"golang.org/x/sys/windows"
)

//go:embed icon.ico
var iconICO []byte

var (
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")
	user32   = windows.NewLazySystemDLL("user32.dll")

	pGetConsoleWindow      = kernel32.NewProc("GetConsoleWindow")
	pGetConsoleProcessList = kernel32.NewProc("GetConsoleProcessList")
	pFreeConsole           = kernel32.NewProc("FreeConsole")

	pMessageBoxW                   = user32.NewProc("MessageBoxW")
	pShowWindow                    = user32.NewProc("ShowWindow")
	pIsIconic                      = user32.NewProc("IsIconic")
	pIsWindowVisible               = user32.NewProc("IsWindowVisible")
	pSetForegroundWindow           = user32.NewProc("SetForegroundWindow")
	pAllowSetForegroundWindow      = user32.NewProc("AllowSetForegroundWindow")
	pSetWindowLongPtrW             = user32.NewProc("SetWindowLongPtrW")
	pCallWindowProcW               = user32.NewProc("CallWindowProcW")
	pPostMessageW                  = user32.NewProc("PostMessageW")
	pSendMessageW                  = user32.NewProc("SendMessageW")
	pGetWindowPlacement            = user32.NewProc("GetWindowPlacement")
	pSetWindowPlacement            = user32.NewProc("SetWindowPlacement")
	pCreateIconFromResourceEx      = user32.NewProc("CreateIconFromResourceEx")
	pGetSystemMetrics              = user32.NewProc("GetSystemMetrics")
	pSetProcessDpiAwarenessContext = user32.NewProc("SetProcessDpiAwarenessContext")
	pSetProcessDPIAware            = user32.NewProc("SetProcessDPIAware")
	pGetDpiForSystem               = user32.NewProc("GetDpiForSystem")
)

const (
	wmClose     = 0x0010
	wmSetIcon   = 0x0080
	swHide      = 0
	swShowNorm  = 1
	swShowMax   = 3
	swShow      = 5
	swRestore   = 9
	gwlpWndProc = ^uintptr(3) // -4
	smCxScreen  = 0
	smCyScreen  = 1
	smCxIcon    = 11
	smCxSmIcon  = 49
	mbIconError = 0x10
	mbIconInfo  = 0x40
	mbTopmost   = 0x40000
)

// Supported reports whether this build can show a native window.
func Supported() bool { return true }

// Detached reports whether the program was started without a terminal of its
// own, e.g. by double-clicking the exe in Explorer (the console window was
// created just for this process) or by a restart without any console.
// Started from cmd/PowerShell/Windows Terminal, the console is shared with
// the shell and this returns false.
func Detached() bool {
	if h, _, _ := pGetConsoleWindow.Call(); h == 0 {
		return true
	}
	var pids [4]uint32
	n, _, _ := pGetConsoleProcessList.Call(uintptr(unsafe.Pointer(&pids[0])), uintptr(len(pids)))
	return n == 1
}

// HideConsole detaches from the console so its window goes away. Writes to
// stdout/stderr fail silently afterwards.
func HideConsole() { pFreeConsole.Call() }

// MessageBox shows a modal message (used for errors when there is no console).
func MessageBox(title, text string, isError bool) {
	t, _ := windows.UTF16PtrFromString(title)
	m, _ := windows.UTF16PtrFromString(text)
	flags := uintptr(mbIconInfo | mbTopmost)
	if isError {
		flags = mbIconError | mbTopmost
	}
	pMessageBoxW.Call(0, uintptr(unsafe.Pointer(m)), uintptr(unsafe.Pointer(t)), flags)
}

// OpenBrowser opens url with the default browser.
func OpenBrowser(url string) {
	verb, _ := windows.UTF16PtrFromString("open")
	u, _ := windows.UTF16PtrFromString(url)
	if err := windows.ShellExecute(0, verb, u, nil, nil, swShowNorm); err != nil {
		slog.Warn("open browser failed", "err", err)
	}
}

// AllowForeground lets any process (the running instance) bring its window
// to the front; called by a second instance just before asking it to.
func AllowForeground() { pAllowSetForegroundWindow.Call(^uintptr(0)) } // ASFW_ANY

// ---- window + tray ---------------------------------------------------------

type state struct {
	Placement     *windowPlacement `json:"placement,omitempty"`
	TrayHintShown bool             `json:"tray_hint_shown,omitempty"`
}

type windowPlacement struct {
	Length, Flags, ShowCmd   uint32
	MinX, MinY, MaxX, MaxY   int32
	Left, Top, Right, Bottom int32
}

var (
	opts      Options
	hwnd      atomic.Uintptr // 0 until the window exists (or if WebView2 is missing)
	quitting  atomic.Bool
	origProc  uintptr
	stMu      sync.Mutex
	st        state
	winDone   = make(chan struct{})
	runOnce   sync.Once
	quitOnce  sync.Once
	noWebView atomic.Bool
)

// Run shows the window and the tray icon and blocks until Quit is called or
// the user picks 退出 in the tray menu. It must be called from the main
// goroutine. If WebView2 is not available the page opens in the browser and
// only the tray icon is shown.
func Run(o Options) {
	runOnce.Do(func() {
		opts = o
		setDPIAware()
		loadState()
		go windowLoop()
		systray.Run(trayReady, nil)
		quitWindow()
		select {
		case <-winDone:
		case <-time.After(3 * time.Second):
		}
	})
}

// Show brings the window to the front (or opens the browser without WebView2).
func Show() {
	h := hwnd.Load()
	if h == 0 {
		if noWebView.Load() {
			OpenBrowser(opts.URL)
		}
		return
	}
	if r, _, _ := pIsIconic.Call(h); r != 0 {
		pShowWindow.Call(h, swRestore)
	} else {
		pShowWindow.Call(h, swShow)
	}
	pSetForegroundWindow.Call(h)
}

// Quit closes the window and removes the tray icon; Run returns afterwards.
func Quit() {
	quitOnce.Do(func() {
		quitWindow()
		systray.Quit()
	})
}

func trayReady() {
	systray.SetIcon(iconICO)
	systray.SetTooltip(opts.Title)
	systray.SetOnTapped(Show)
	open := systray.AddMenuItem("打开主界面", "")
	browser := systray.AddMenuItem("在浏览器中打开", "")
	systray.AddSeparator()
	quit := systray.AddMenuItem("退出", "退出后不再定时同步和生成 AI 日报")
	go func() {
		for {
			select {
			case <-open.ClickedCh:
				Show()
			case <-browser.ClickedCh:
				OpenBrowser(opts.URL)
			case <-quit.ClickedCh:
				if opts.OnQuit != nil {
					opts.OnQuit()
				}
				Quit()
				return
			}
		}
	}()
}

func windowLoop() {
	defer close(winDone)
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	w, h := defaultSize()
	wv := webview2.NewWithOptions(webview2.WebViewOptions{
		AutoFocus: true,
		DataPath:  filepath.Join(opts.StateDir, "webview"),
		WindowOptions: webview2.WindowOptions{
			Title: opts.Title, Width: uint(w), Height: uint(h), Center: true,
		},
	})
	if wv == nil {
		slog.Warn("WebView2 is not available, opening the browser instead")
		noWebView.Store(true)
		OpenBrowser(opts.URL)
		return
	}
	handle := uintptr(wv.Window())
	setIcons(handle)
	restorePlacement(handle)
	origProc, _, _ = pSetWindowLongPtrW.Call(handle, gwlpWndProc, windows.NewCallback(wndProc))
	wv.Bind("__openExternal", func(u string) { OpenBrowser(u) })
	wv.Init(openExternalJS)
	wv.Navigate(opts.URL)
	hwnd.Store(handle)
	if quitting.Load() { // Quit raced with window creation
		pPostMessageW.Call(handle, wmClose, 0, 0)
	}
	wv.Run()
	hwnd.Store(0)
}

// wndProc intercepts closing the window: it hides to the tray instead, unless
// the program is quitting.
func wndProc(h, msg, wp, lp uintptr) uintptr {
	if msg == wmClose && !quitting.Load() {
		savePlacement(h)
		pShowWindow.Call(h, swHide)
		stMu.Lock()
		first := !st.TrayHintShown
		st.TrayHintShown = true
		stMu.Unlock()
		if first {
			saveState()
			go MessageBox(opts.Title, "窗口已关闭，程序会在任务栏右下角的托盘里继续运行（定时同步、AI 日报照常进行）。\n\n"+
				"单击托盘图标重新打开；右键托盘图标选择「退出」可完全退出。\n"+
				"（如果看不到图标，点任务栏右下角的 ^ 展开。）", false)
		}
		return 0
	}
	r, _, _ := pCallWindowProcW.Call(origProc, h, msg, wp, lp)
	return r
}

func quitWindow() {
	quitting.Store(true)
	if h := hwnd.Load(); h != 0 {
		if v, _, _ := pIsWindowVisible.Call(h); v != 0 {
			savePlacement(h)
		}
		pPostMessageW.Call(h, wmClose, 0, 0)
	}
}

// ---- helpers ---------------------------------------------------------------

func setDPIAware() {
	// PER_MONITOR_AWARE_V2 (Win10 1703+); fall back to system-aware.
	if pSetProcessDpiAwarenessContext.Find() == nil {
		if r, _, _ := pSetProcessDpiAwarenessContext.Call(^uintptr(3)); r != 0 { // -4
			return
		}
	}
	pSetProcessDPIAware.Call()
}

func defaultSize() (int, int) {
	dpi := uintptr(96)
	if pGetDpiForSystem.Find() == nil {
		if d, _, _ := pGetDpiForSystem.Call(); d != 0 {
			dpi = d
		}
	}
	w, h := int(1280*dpi/96), int(840*dpi/96)
	sw, _, _ := pGetSystemMetrics.Call(smCxScreen)
	sh, _, _ := pGetSystemMetrics.Call(smCyScreen)
	if sw > 0 && w > int(sw)*9/10 {
		w = int(sw) * 9 / 10
	}
	if sh > 0 && h > int(sh)*9/10 {
		h = int(sh) * 9 / 10
	}
	return w, h
}

// setIcons gives the window (and so its taskbar button) the app icon.
func setIcons(h uintptr) {
	big, _, _ := pGetSystemMetrics.Call(smCxIcon)
	small, _, _ := pGetSystemMetrics.Call(smCxSmIcon)
	if ic := iconFromICO(iconICO, int(big)); ic != 0 {
		pSendMessageW.Call(h, wmSetIcon, 1, ic) // ICON_BIG
	}
	if ic := iconFromICO(iconICO, int(small)); ic != 0 {
		pSendMessageW.Call(h, wmSetIcon, 0, ic) // ICON_SMALL
	}
}

// iconFromICO creates an HICON from the .ico entry closest to size px.
func iconFromICO(ico []byte, size int) uintptr {
	if len(ico) < 6 || size <= 0 {
		return 0
	}
	n := int(binary.LittleEndian.Uint16(ico[4:]))
	best, bestDim := -1, 0
	for i := 0; i < n && 6+16*(i+1) <= len(ico); i++ {
		e := ico[6+16*i:]
		dim := int(e[0])
		if dim == 0 {
			dim = 256
		}
		// smallest entry that is at least size; otherwise the largest
		if best < 0 || (dim >= size && (bestDim < size || dim < bestDim)) || (bestDim < size && dim > bestDim) {
			best, bestDim = i, dim
		}
	}
	if best < 0 {
		return 0
	}
	e := ico[6+16*best:]
	sz := int(binary.LittleEndian.Uint32(e[8:]))
	off := int(binary.LittleEndian.Uint32(e[12:]))
	if off+sz > len(ico) {
		return 0
	}
	data := ico[off : off+sz]
	h, _, _ := pCreateIconFromResourceEx.Call(uintptr(unsafe.Pointer(&data[0])), uintptr(sz), 1, 0x00030000,
		uintptr(size), uintptr(size), 0)
	return h
}

func statePath() string { return filepath.Join(opts.StateDir, "window.json") }

func loadState() {
	b, err := os.ReadFile(statePath())
	if err != nil {
		return
	}
	stMu.Lock()
	json.Unmarshal(b, &st)
	stMu.Unlock()
}

func saveState() {
	stMu.Lock()
	b, _ := json.MarshalIndent(st, "", "  ")
	stMu.Unlock()
	os.MkdirAll(opts.StateDir, 0o755)
	if err := os.WriteFile(statePath(), b, 0o644); err != nil {
		slog.Warn("save window state failed", "err", err)
	}
}

func savePlacement(h uintptr) {
	p := windowPlacement{Length: uint32(unsafe.Sizeof(windowPlacement{}))}
	if r, _, _ := pGetWindowPlacement.Call(h, uintptr(unsafe.Pointer(&p))); r == 0 {
		return
	}
	if p.ShowCmd != swShowMax {
		p.ShowCmd = swShowNorm // never restore minimized
	}
	stMu.Lock()
	st.Placement = &p
	stMu.Unlock()
	saveState()
}

func restorePlacement(h uintptr) {
	stMu.Lock()
	p := st.Placement
	stMu.Unlock()
	if p == nil || p.Right-p.Left < 200 || p.Bottom-p.Top < 150 {
		return
	}
	q := *p
	q.Length = uint32(unsafe.Sizeof(q))
	q.Flags = 0
	pSetWindowPlacement.Call(h, uintptr(unsafe.Pointer(&q)))
}
