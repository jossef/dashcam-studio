---
name: dashcam-studio-dev
description: Run, test and visually verify the Dashcam Studio player in this repo. Use when changing cmd/dashcam-studio or internal/ (Go server, GPS, zip, favorites, window) or web/ (React UI), or when asked to run/screenshot/build/release the app.
---

# Dashcam Studio dev loop

Stack: Go (stdlib `net/http`) API + media server in `internal/server`, React 19 / TypeScript / Vite / zustand UI in `web/`, embedded into the binary from `internal/webui/ui/dist` (generated, git-ignored). The window is WebView2 on Windows (`internal/desktop`, pure Go via go-webview2), a Chromium `--app` window elsewhere. Everything builds with `CGO_ENABLED=0`.

Go lives at `%USERPROFILE%\sdk\go1.27.4\go\bin` (on the user PATH; a shell started before it was installed may need `export PATH="/c/Users/user/sdk/go1.27.4/go/bin:$PATH"`).

## Rules
- The dashcam card (e.g. `E:\`) is **read-only**. Never write, rename or delete anything under it. Caches go to `paths.CacheDir()` and favorites to `paths.ConfigDir()` (override with `DASHCAM_STUDIO_HOME`).
- No cgo, no runtime dependencies. ffmpeg stays optional (`media.FFmpeg`, found next to the exe or on PATH).
- Media is addressed by clip id (`<category>/<stem>`), never by raw path.
- UI is Hebrew RTL. Put every string in `web/src/i18n/he.ts`. Use logical CSS properties (`inset-inline-start`, `margin-inline-end`…). Timeline, trip ribbon and ⏮▶⏭ stay `dir="ltr"`. Wrap Latin/number runs that sit inside Hebrew in `dir="ltr"`; `fmtBytes` already isolates sizes.
- Lists are newest → oldest (days, trips, clips). Playback/next-clip order is chronological.
- Keyboard shortcuts match `e.code` (physical keys) so they work with a Hebrew layout.
- Keep the JSON API identical to what `web/src/lib/api.ts` and `web/src/types.ts` expect.

## Build / run
```
go run ./tools/build              # full release → dist/ (tests ‖ UI, then all targets)
go run ./tools/build build        # UI + host binary only
go run ./tools/build dev -no-open # Vite :5173 + Go :8765 with restart on .go changes
set DASHCAM_STUDIO_HOME=<scratch dir>
dist\dashcam-studio-<ver>-windows-x64.exe E:\ --ui none --port 8765   # headless, for Playwright
```
`dev` refuses a busy port. On Windows check `Get-NetTCPConnection -LocalPort 8765` for stale servers; the app itself silently moves to the next free port.

The release `.exe` is a GUI-subsystem binary: it attaches to the parent console (or keeps a piped stdout); double-clicked it logs to `%LOCALAPPDATA%\dashcam-studio\dashcam-studio.log` and shows fatal errors in a message box. Don't launch it with bad flags from a pipe-less context expecting console output.

## Test
1. `go run ./tools/build test` (Go vet/tests ‖ `tsc` + vitest). Server tests use `httptest` + synthetic cards (`internal/testcard`).
2. If a card is mounted: `DASHCAM_CARD='E:\' DASHCAM_LIVE_ZIP=1 go test ./internal/... -run Live -v`.
3. UI in Chrome: `PYTHONIOENCODING=utf-8 DASHCAM_URL=http://localhost:<port> python tests/e2e/smoke_chrome.py <outdir>` (real Chrome, `headless=False`; headless Chromium lacks HEVC). Look at the screenshots. `window.dashcamStudio` exposes `{engine, store}`; sync drift should stay within ±0.05 s.
4. The WebView2 window: start the exe with `WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS=--remote-debugging-port=9333` and attach Playwright via `chromium.connect_over_cdp("http://127.0.0.1:9333")`. Close it by posting `WM_CLOSE` to the `webview`-class window titled "Dashcam Studio"; the process must exit.

## Media connection
- Every route that reads the card is wrapped by `Server.card(...)`, which takes a `Lease` from `State.UseCard()` and returns 503 `{"code":"disconnected"}` when the card is gone or closed.
- `State.Release` cancels the lease context. `card()` turns that into `SetWriteDeadline(now)` on the connection, which unblocks a write stuck on a browser that stopped reading; `leaseReader` hides the `*os.File` so net/http can't use sendfile (which a deadline can't interrupt). ffmpeg runs under the lease context, so it is killed too.
- `State.CheckConnection()` is one watcher step (lost → reconnected); tests call it directly, `State.Watch` runs it every 1.5 s.
- Never test eject on the user's real card. Simulate pull/re-insert by mounting a scratch copy with `subst Q: <dir>` and removing it with `subst Q: /D` (`tests/e2e/disconnect_chrome.py`).

## Gotchas
- Leaflet's container is forced `dir="ltr"`. OSM tiles are darkened with a CSS filter, because CARTO needs a key.
- Overlays (`.welcome`, `.modal`, `.toasts`) need a z-index above Leaflet's (1000).
- GPS log time is not UTC. The offset is learned in `gps.Index.learnOffset`. Speed is in cm/s and heading is degrees×100.
- `useEngine` selectors must return primitives or stable snapshot sub-objects (useSyncExternalStore).
- Never start browsers with `proc.Command` (it passes SW_HIDE, so the window stays invisible); use plain `exec.Command`.
- WebView2 leaves Esc-to-exit-fullscreen to the host; `internal/desktop/webview_windows.go` injects it, and syncs the window's borderless fullscreen with `fullscreenchange`.
