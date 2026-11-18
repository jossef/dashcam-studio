// Package paths locates the per-user folders the app writes to. Nothing is
// ever written to the dashcam card.
package paths

import (
	"os"
	"path/filepath"
	"sync"
)

const (
	appDir    = "dashcam-studio"
	legacyDir = "dashcam-master" // the app's previous name; moved to appDir on first use
)

var migrated sync.Map // base folder → struct{}

// ConfigDir holds favorites: %APPDATA%\dashcam-studio, ~/Library/Application
// Support/dashcam-studio or ~/.config/dashcam-studio. DASHCAM_STUDIO_HOME
// overrides it.
func ConfigDir() string {
	dir := os.Getenv("DASHCAM_STUDIO_HOME")
	if dir == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			base = os.TempDir()
		}
		dir = appFolder(base)
	}
	_ = os.MkdirAll(dir, 0o755)
	return dir
}

// CacheDir holds thumbnails, the log and the window's browser profile:
// %LOCALAPPDATA%\dashcam-studio, ~/Library/Caches/dashcam-studio or ~/.cache/dashcam-studio.
func CacheDir(sub ...string) string {
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	dir := filepath.Join(append([]string{appFolder(base)}, sub...)...)
	_ = os.MkdirAll(dir, 0o755)
	return dir
}

// appFolder returns base/dashcam-studio. The first time, a folder left by the
// old name (base/dashcam-master) is moved there, so favorites carry over.
func appFolder(base string) string {
	dir := filepath.Join(base, appDir)
	if _, done := migrated.LoadOrStore(base, struct{}{}); !done {
		migrate(filepath.Join(base, legacyDir), dir)
	}
	return dir
}

func migrate(old, dir string) {
	if _, err := os.Stat(dir); err == nil {
		return
	}
	if st, err := os.Stat(old); err != nil || !st.IsDir() {
		return
	}
	if os.Rename(old, dir) == nil {
		return
	}
	// The old folder is busy (e.g. an old version is still running): at least keep the favorites.
	if data, err := os.ReadFile(filepath.Join(old, "favorites.json")); err == nil && os.MkdirAll(dir, 0o755) == nil {
		_ = os.WriteFile(filepath.Join(dir, "favorites.json"), data, 0o644)
	}
}
