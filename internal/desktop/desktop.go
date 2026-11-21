// Package desktop shows the web UI in a native window with a tray icon, so
// the program can be used like a desktop app instead of through a browser.
//
// Only Windows is supported (WebView2 + notification-area icon); on other
// systems Supported reports false and the program keeps serving the web UI.
package desktop

// Options configures Run.
type Options struct {
	URL      string // page shown in the window
	Title    string // window title and tray tooltip
	StateDir string // where window placement and the WebView2 profile are kept
	OnQuit   func() // called when the user picks 退出 in the tray menu
}

// openExternalJS routes links that leave the app (other sites, new-tab links)
// to the default browser instead of opening WebView2 popup windows.
const openExternalJS = `(function(){
  function abs(u){ try { return new URL(u, location.href); } catch (e) { return null; } }
  function external(u){ return u && /^https?:$/.test(u.protocol) && u.origin !== location.origin; }
  document.addEventListener('click', function(e){
    var a = e.target && e.target.closest ? e.target.closest('a[href]') : null;
    if (!a || a.hasAttribute('download')) return;
    var raw = a.getAttribute('href') || '';
    if (raw.charAt(0) === '#') return;
    var u = abs(raw);
    if (!u || !/^https?:$/.test(u.protocol)) return;
    if (external(u) || a.target === '_blank') { e.preventDefault(); window.__openExternal(u.href); }
  }, true);
  var orig = window.open;
  window.open = function(u){
    var x = abs(u);
    if (x && /^https?:$/.test(x.protocol)) { window.__openExternal(x.href); return null; }
    return orig.apply(window, arguments);
  };
})();`
