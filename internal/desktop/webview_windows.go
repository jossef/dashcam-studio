package desktop

import (
	"context"
	"path/filepath"
	"runtime"
	"unsafe"

	webview2 "github.com/jchv/go-webview2"
	"golang.org/x/sys/windows"
)

var (
	user32                     = windows.NewLazySystemDLL("user32.dll")
	procSetDpiAwarenessContext = user32.NewProc("SetProcessDpiAwarenessContext")
	procGetDpiForSystem        = user32.NewProc("GetDpiForSystem")
	procSystemParametersInfo   = user32.NewProc("SystemParametersInfoW")
	procGetWindowLongPtr       = user32.NewProc("GetWindowLongPtrW")
	procSetWindowLongPtr       = user32.NewProc("SetWindowLongPtrW")
	procGetWindowPlacement     = user32.NewProc("GetWindowPlacement")
	procSetWindowPlacement     = user32.NewProc("SetWindowPlacement")
	procMonitorFromWindow      = user32.NewProc("MonitorFromWindow")
	procGetMonitorInfo         = user32.NewProc("GetMonitorInfoW")
	procSetWindowPos           = user32.NewProc("SetWindowPos")
)

const (
	gwlStyle             = ^uintptr(15) // -16
	wsOverlappedWindow   = 0x00CF0000
	monitorNearest       = 2
	swpNoZOrder          = 0x0004
	swpNoOwnerZOrder     = 0x0200
	swpFrameChanged      = 0x0020
	swpNoMove            = 0x0002
	swpNoSize            = 0x0001
	spiGetWorkArea       = 0x0030
	dpiPerMonitorAwareV2 = ^uintptr(3) // -4
)

type rect struct{ Left, Top, Right, Bottom int32 }

type monitorInfo struct {
	Size    uint32
	Monitor rect
	Work    rect
	Flags   uint32
}

type windowPlacement struct {
	Length, Flags, ShowCmd uint32
	MinPos, MaxPos         struct{ X, Y int32 }
	Normal                 rect
	Device                 rect
}

// fullscreenJS keeps the window in step with the page's Fullscreen API: when
// the player goes fullscreen (F key / button), so does the window. An
// embedded WebView2 leaves Esc-to-exit to the host, so handle it here too.
const fullscreenJS = `(() => {
  const sync = () => window.__dmFullscreen && window.__dmFullscreen(!!document.fullscreenElement);
  document.addEventListener('fullscreenchange', sync);
  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape' && document.fullscreenElement) document.exitFullscreen();
  }, true);
})();`

func runWebview(ctx context.Context, o Options) error {
	runtime.LockOSThread() // the window and its message loop live on this thread
	defer runtime.UnlockOSThread()
	procSetDpiAwarenessContext.Call(dpiPerMonitorAwareV2) // crisp text on high-DPI screens

	width, height := windowSize(1600, 950)
	w := webview2.NewWithOptions(webview2.WebViewOptions{
		Debug:     o.Debug,
		AutoFocus: true,
		DataPath:  filepath.Join(o.DataDir, "webview2"),
		WindowOptions: webview2.WindowOptions{
			Title: o.Title, Width: width, Height: height, IconId: appIconID(), Center: true,
		},
	})
	if w == nil {
		return ErrUnavailable // WebView2 runtime missing
	}
	defer w.Destroy()
	hwnd := uintptr(w.Window())
	fs := &fullscreen{hwnd: hwnd}
	_ = w.Bind("__dmFullscreen", fs.set)
	w.Init(fullscreenJS)
	w.Navigate(o.URL)

	stop := context.AfterFunc(ctx, func() { w.Destroy() }) // closes the window from any thread
	defer stop()
	w.Run()
	return nil
}

// appIconID is the icon resource the release build embeds (tools/build), or 0
// for the default icon in a plain `go build`.
func appIconID() uint {
	const rtGroupIcon = 14
	var self windows.Handle
	if windows.GetModuleHandleEx(0, nil, &self) != nil {
		return 0
	}
	if _, err := windows.FindResource(self, windows.ResourceID(1), windows.ResourceID(rtGroupIcon)); err != nil {
		return 0
	}
	return 1
}

// windowSize scales a size in CSS pixels by the screen DPI, capped to 92% of the work area.
func windowSize(cssW, cssH uint) (uint, uint) {
	dpi, _, _ := procGetDpiForSystem.Call()
	if dpi == 0 {
		dpi = 96
	}
	w, h := cssW*uint(dpi)/96, cssH*uint(dpi)/96
	var work rect
	if r, _, _ := procSystemParametersInfo.Call(spiGetWorkArea, 0, uintptr(unsafe.Pointer(&work)), 0); r != 0 {
		maxW, maxH := uint(work.Right-work.Left)*92/100, uint(work.Bottom-work.Top)*92/100
		w, h = min(w, maxW), min(h, maxH)
	}
	return w, h
}

type fullscreen struct {
	hwnd      uintptr
	on        bool
	style     uintptr
	placement windowPlacement
}

// set switches the window between borderless fullscreen on its monitor and
// its previous size and position. Called on the UI thread.
func (f *fullscreen) set(on bool) {
	if on == f.on {
		return
	}
	f.on = on
	if on {
		f.style, _, _ = procGetWindowLongPtr.Call(f.hwnd, gwlStyle)
		f.placement.Length = uint32(unsafe.Sizeof(f.placement))
		procGetWindowPlacement.Call(f.hwnd, uintptr(unsafe.Pointer(&f.placement)))
		mon, _, _ := procMonitorFromWindow.Call(f.hwnd, monitorNearest)
		mi := monitorInfo{Size: uint32(unsafe.Sizeof(monitorInfo{}))}
		procGetMonitorInfo.Call(mon, uintptr(unsafe.Pointer(&mi)))
		procSetWindowLongPtr.Call(f.hwnd, gwlStyle, f.style&^wsOverlappedWindow)
		m := mi.Monitor
		procSetWindowPos.Call(f.hwnd, 0, uintptr(m.Left), uintptr(m.Top), uintptr(m.Right-m.Left), uintptr(m.Bottom-m.Top),
			swpNoOwnerZOrder|swpFrameChanged)
		return
	}
	procSetWindowLongPtr.Call(f.hwnd, gwlStyle, f.style)
	procSetWindowPlacement.Call(f.hwnd, uintptr(unsafe.Pointer(&f.placement)))
	procSetWindowPos.Call(f.hwnd, 0, 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoZOrder|swpNoOwnerZOrder|swpFrameChanged)
}
