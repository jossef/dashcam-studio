//go:build !windows

package volume

import (
	"os"
	"path/filepath"
	"syscall"
)

func candidates() []string {
	var out []string
	for _, pat := range []string{"/Volumes/*", "/media/*", "/media/*/*", "/run/media/*/*", "/mnt/*"} {
		matches, _ := filepath.Glob(pat)
		for _, m := range matches {
			if st, err := os.Stat(m); err == nil && st.IsDir() {
				out = append(out, m)
			}
		}
	}
	return out
}

// Roots are the starting points shown in the folder browser.
func Roots() []string {
	out := []string{"/"}
	home, _ := os.UserHomeDir()
	for _, p := range []string{"/Volumes", "/media", "/mnt", home} {
		if st, err := os.Stat(p); p != "" && err == nil && st.IsDir() {
			out = append(out, p)
		}
	}
	return out
}

// Label is the folder name of the mount point.
func Label(path string) string { return baseName(path) }

func device(path string) (uint64, bool) {
	var st syscall.Stat_t
	if syscall.Stat(path, &st) != nil {
		return 0, false
	}
	return uint64(st.Dev), true //nolint:unconvert // Dev is int32 on darwin
}

// MountPoint walks up from path until the device changes.
func MountPoint(path string) string {
	path, _ = filepath.Abs(path)
	dev, ok := device(path)
	if !ok {
		return path
	}
	for {
		parent := filepath.Dir(path)
		if parent == path {
			return path
		}
		if pdev, ok := device(parent); !ok || pdev != dev {
			return path
		}
		path = parent
	}
}
