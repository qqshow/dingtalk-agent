//go:build windows

package desktop

import (
	_ "embed"
	"encoding/binary"
	"encoding/json"
	"fmt"
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
	"golang.org/x/sys/windows/registry"
)

//go:embed icon.ico
var iconICO []byte

var (
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")
	dwmapi   = windows.NewLazySystemDLL("dwmapi.dll")

	pDwmSetWindowAttribute = dwmapi.NewProc("DwmSetWindowAttribute")
	pSetWindowPos          = user32.NewProc("SetWindowPos")
	user32                 = windows.NewLazySystemDLL("user32.dll")

	pGetConsoleWindow      = kernel32.NewProc("GetConsoleWindow")
	pGetConsoleProcessList = kernel32.NewProc("GetConsoleProcessList")
	pFreeConsole           = kernel32.NewProc("FreeConsole")
	pAttachConsole         = kernel32.NewProc("AttachConsole")

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

// Prepare sets up the standard streams and reports whether the program was
// started without a terminal of its own, e.g. by double-clicking the exe.
// It must run before anything is printed.
//
// Release builds use the GUI subsystem (-H windowsgui) so double-clicking
// does not flash a console window. Then:
//   - stdio already redirected (pipes from an MCP client, `> file`): kept,
//     not detached;
//   - started from cmd/PowerShell/Windows Terminal: attach to that console
//     and point stdout/stderr at it, not detached;
//   - otherwise (Explorer, a restart without console): detached.
//
// Console-subsystem builds (plain `go build`) are detached when the console
// was created just for this process; it is released so its window closes.
func Prepare() bool {
	if h, _, _ := pGetConsoleWindow.Call(); h != 0 {
		var pids [4]uint32
		n, _, _ := pGetConsoleProcessList.Call(uintptr(unsafe.Pointer(&pids[0])), uintptr(len(pids)))
		if n == 1 {
			pFreeConsole.Call()
			return true
		}
		return false
	}
	if validHandle(os.Stdout) || validHandle(os.Stderr) {
		return false
	}
	if r, _, _ := pAttachConsole.Call(^uintptr(0)); r == 0 { // ATTACH_PARENT_PROCESS
		return true
	}
	// os.NewFile (unlike os.OpenFile) detects console handles, so output is
	// written as Unicode (WriteConsoleW) and Chinese text is not garbled.
	reopen := func(f **os.File, name string, std uint32) {
		if validHandle(*f) {
			return
		}
		p, _ := windows.UTF16PtrFromString(name)
		h, err := windows.CreateFile(p, windows.GENERIC_READ|windows.GENERIC_WRITE,
			windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, 0, 0)
		if err != nil {
			return
		}
		*f = os.NewFile(uintptr(h), name)
		windows.SetStdHandle(std, h)
	}
	reopen(&os.Stdout, "CONOUT$", windows.STD_OUTPUT_HANDLE)
	reopen(&os.Stderr, "CONOUT$", windows.STD_ERROR_HANDLE)
	reopen(&os.Stdin, "CONIN$", windows.STD_INPUT_HANDLE)
	// The shell has already printed its prompt (it does not wait for GUI
	// programs); start on a fresh line.
	os.Stderr.WriteString("\r\n")
	return false
}

func validHandle(f *os.File) bool {
	if f == nil {
		return false
	}
	h := windows.Handle(f.Fd())
	if h == 0 || h == windows.InvalidHandle {
		return false
	}
	t, err := windows.GetFileType(h)
	return err == nil && t != windows.FILE_TYPE_UNKNOWN
}

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
	// Last page theme, so the next start paints the right colors before the
	// page has loaded.
	Dark *bool   `json:"dark,omitempty"`
	BG   *[3]int `json:"bg,omitempty"`
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

	dark, bg := startTheme()
	// Paint the theme color (not white) until the page has rendered.
	os.Setenv("WEBVIEW2_DEFAULT_BACKGROUND_COLOR", fmt.Sprintf("FF%02X%02X%02X", bg[0], bg[1], bg[2]))
	unhook := hookCreate(dark, bg)

	w, h := defaultSize()
	wv := webview2.NewWithOptions(webview2.WebViewOptions{
		AutoFocus: true,
		DataPath:  filepath.Join(opts.StateDir, "webview"),
		WindowOptions: webview2.WindowOptions{
			Title: opts.Title, Width: uint(w), Height: uint(h), Center: true,
		},
	})
	unhook()
	if wv == nil {
		slog.Warn("WebView2 is not available, opening the browser instead")
		noWebView.Store(true)
		go func() { waitReady(); OpenBrowser(opts.URL) }()
		return
	}
	handle := uintptr(wv.Window())
	setIcons(handle)
	setTitleBar(handle, dark, &bg)
	restorePlacement(handle)
	origProc, _, _ = pSetWindowLongPtrW.Call(handle, gwlpWndProc, windows.NewCallback(wndProc))
	wv.Bind("__openExternal", func(u string) { OpenBrowser(u) })
	wv.Bind("__setTheme", func(dark bool, r, g, b int) {
		c := [3]int{r, g, b}
		setTitleBar(handle, dark, &c)
		rememberTheme(dark, c)
	})
	wv.Init(openExternalJS)
	wv.Init(themeJS)
	if isReady() {
		wv.Navigate(opts.URL)
	} else {
		// The server is still starting: show a loading page right away.
		wv.SetHtml(loadingHTML(dark, bg))
		go func() {
			waitReady()
			wv.Dispatch(func() { wv.Navigate(opts.URL) })
		}()
	}
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

// ---- title bar theme -------------------------------------------------------

const (
	dwmaDarkModeOld  = 19 // Windows 10 before 20H1
	dwmaDarkMode     = 20
	dwmaBorderColor  = 34 // Windows 11
	dwmaCaptionColor = 35
	dwmaTextColor    = 36
)

var lastTitleBar atomic.Value // string key of the last applied theme

// setTitleBar makes the window frame follow the page theme: dark or light
// title bar (Windows 10 1809+), and on Windows 11 the exact page background
// as caption color. bg is nil to use the system default caption color.
func setTitleBar(h uintptr, dark bool, bg *[3]int) {
	key := fmt.Sprint(dark, bg)
	if bg != nil {
		key = fmt.Sprint(dark, *bg)
	}
	if lastTitleBar.Load() == key {
		return
	}
	lastTitleBar.Store(key)

	set := func(attr uintptr, v uint32) bool {
		r, _, _ := pDwmSetWindowAttribute.Call(h, attr, uintptr(unsafe.Pointer(&v)), 4)
		return r == 0
	}
	var d uint32
	if dark {
		d = 1
	}
	if !set(dwmaDarkMode, d) {
		set(dwmaDarkModeOld, d)
	}
	if bg != nil {
		c := uint32(bg[0]&0xff) | uint32(bg[1]&0xff)<<8 | uint32(bg[2]&0xff)<<16 // COLORREF 0x00BBGGRR
		set(dwmaCaptionColor, c)
		set(dwmaBorderColor, c)
		text := uint32(0x00202020)
		if dark {
			text = 0x00E8E8E8
		}
		set(dwmaTextColor, text)
	}
	// Windows 10 only repaints the frame on a frame change.
	const swpFlags = 0x0001 | 0x0002 | 0x0004 | 0x0010 | 0x0020 // NOSIZE|NOMOVE|NOZORDER|NOACTIVATE|FRAMECHANGED
	pSetWindowPos.Call(h, 0, 0, 0, 0, 0, swpFlags)
}

// systemDark reports whether Windows apps use the dark theme.
func systemDark() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	v, _, err := k.GetIntegerValue("AppsUseLightTheme")
	return err == nil && v == 0
}

// ---- startup appearance ------------------------------------------------------

func isReady() bool {
	if opts.Ready == nil {
		return true
	}
	select {
	case <-opts.Ready:
		return true
	default:
		return false
	}
}

func waitReady() {
	if opts.Ready != nil {
		<-opts.Ready
	}
}

var (
	lightBG = [3]int{0xf2, 0xf3, 0xf5}
	darkBG  = [3]int{0x13, 0x14, 0x16}
)

// startTheme returns the theme to paint before the page reports its own:
// the last page theme, else the Windows app theme.
func startTheme() (bool, [3]int) {
	stMu.Lock()
	d, c := st.Dark, st.BG
	stMu.Unlock()
	if d != nil && c != nil {
		return *d, *c
	}
	if systemDark() {
		return true, darkBG
	}
	return false, lightBG
}

func rememberTheme(dark bool, c [3]int) {
	stMu.Lock()
	changed := st.Dark == nil || *st.Dark != dark || st.BG == nil || *st.BG != c
	if changed {
		st.Dark, st.BG = &dark, &c
	}
	stMu.Unlock()
	if changed {
		saveState()
	}
}

func loadingHTML(dark bool, bg [3]int) string {
	fg := "#6b7280"
	if dark {
		fg = "#8a8f98"
	}
	return fmt.Sprintf(`<!doctype html><html><head><meta charset="utf-8"><style>
html,body{margin:0;height:100%%;background:rgb(%d,%d,%d);color:%s;font:14px "Microsoft YaHei UI","Segoe UI",sans-serif}
body{display:flex;align-items:center;justify-content:center;gap:10px;user-select:none}
.s{width:18px;height:18px;border:2px solid currentColor;border-top-color:transparent;border-radius:50%%;animation:r .8s linear infinite}
@keyframes r{to{transform:rotate(360deg)}}
</style></head><body><div class="s"></div>正在启动…</body></html>`, bg[0], bg[1], bg[2], fg)
}

var (
	gdi32                = windows.NewLazySystemDLL("gdi32.dll")
	pCreateSolidBrush    = gdi32.NewProc("CreateSolidBrush")
	pSetWindowsHookExW   = user32.NewProc("SetWindowsHookExW")
	pUnhookWindowsHookEx = user32.NewProc("UnhookWindowsHookEx")
	pCallNextHookEx      = user32.NewProc("CallNextHookEx")
	pSetClassLongPtrW    = user32.NewProc("SetClassLongPtrW")
	pGetClassNameW       = user32.NewProc("GetClassNameW")
)

// createStruct is CREATESTRUCTW.
type createStruct struct {
	CreateParams, Instance, Menu, Parent uintptr
	Cy, Cx, Y, X                         int32
	Style                                int32
	Name, Class                          uintptr
	ExStyle                              uint32
}

// cbtCreateWnd is CBT_CREATEWNDW.
type cbtCreateWnd struct {
	Lpcs        *createStruct
	InsertAfter uintptr
}

// hookCreate watches the creation of the webview window on this thread
// (the library creates and shows it before WebView2 has started, which
// takes a moment): it gets the saved size and position, a background in
// the theme color, the app icon and the themed title bar from the first
// frame on. The returned function removes the hook.
func hookCreate(dark bool, bg [3]int) func() {
	stMu.Lock()
	p := st.Placement
	stMu.Unlock()
	color := uintptr(bg[0]&0xff) | uintptr(bg[1]&0xff)<<8 | uintptr(bg[2]&0xff)<<16
	brush, _, _ := pCreateSolidBrush.Call(color)
	var hook uintptr
	isWebview := func(h uintptr) bool {
		buf := make([]uint16, 32)
		pGetClassNameW.Call(h, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
		return windows.UTF16ToString(buf) == "webview"
	}
	cb := windows.NewCallback(func(code int32, wp, lp uintptr) uintptr {
		const hcbtCreateWnd, hcbtActivate = 3, 5
		switch code {
		case hcbtCreateWnd:
			if isWebview(wp) {
				if brush != 0 {
					pSetClassLongPtrW.Call(wp, ^uintptr(9), brush) // GCLP_HBRBACKGROUND = -10
				}
				if p != nil && p.Right-p.Left >= 200 && p.Bottom-p.Top >= 150 {
					// lp points to CBT_CREATEWND; reinterpret it without a
					// uintptr->Pointer conversion.
					cbt := *(**cbtCreateWnd)(unsafe.Pointer(&lp))
					cs := cbt.Lpcs
					cs.X, cs.Y = p.Left, p.Top
					cs.Cx, cs.Cy = p.Right-p.Left, p.Bottom-p.Top
				}
			}
		case hcbtActivate:
			if isWebview(wp) {
				setIcons(wp)
				setTitleBar(wp, dark, &bg)
			}
		}
		r, _, _ := pCallNextHookEx.Call(hook, uintptr(code), wp, lp)
		return r
	})
	const whCBT = 5
	tid := windows.GetCurrentThreadId()
	hook, _, _ = pSetWindowsHookExW.Call(whCBT, cb, 0, uintptr(tid))
	return func() {
		if hook != 0 {
			pUnhookWindowsHookEx.Call(hook)
			hook = 0
		}
	}
}
