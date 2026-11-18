//go:build !windows

package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/jossef/dashcam-studio/internal/paths"
)

// attachConsole reports whether the app runs in a terminal. Started from
// Finder or a desktop launcher there is none, so the log goes to a file in
// the cache folder.
func attachConsole() bool {
	if st, err := os.Stdin.Stat(); err == nil && st.Mode()&os.ModeCharDevice != 0 {
		return true
	}
	if f, err := os.Create(filepath.Join(paths.CacheDir(), "dashcam-studio.log")); err == nil {
		os.Stdout, os.Stderr = f, f
		log.SetOutput(f)
	}
	return false
}

func fatalDialog(string) {}
