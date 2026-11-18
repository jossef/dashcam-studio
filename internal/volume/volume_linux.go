//go:build !windows && !darwin

package volume

import (
	"bufio"
	"os"
	"os/exec"
	"strings"
	"time"
)

// IsRemovable treats desktop automounts (/media, /run/media, /mnt) as removable.
func IsRemovable(path string) bool {
	m := MountPoint(path)
	return strings.HasPrefix(m, "/media/") || strings.HasPrefix(m, "/run/media/") || strings.HasPrefix(m, "/mnt/")
}

// Eject prefers the desktop's own eject (no root needed), then udisks.
func Eject(path string) (bool, string) {
	if !IsRemovable(path) {
		return false, "not a removable drive"
	}
	mount := MountPoint(path)
	if _, err := exec.LookPath("gio"); err == nil {
		if ok, _ := run(25*time.Second, "gio", "mount", "-e", mount); ok && gone(mount, 5*time.Second) {
			return true, ""
		}
	}
	if _, err := exec.LookPath("udisksctl"); err == nil {
		if dev := deviceOf(mount); dev != "" {
			ok, msg := run(25*time.Second, "udisksctl", "unmount", "-b", dev)
			if ok {
				run(25*time.Second, "udisksctl", "power-off", "-b", dev)
				return true, ""
			}
			return false, msg
		}
	}
	return false, "no eject tool found (install gio or udisks2)"
}

func deviceOf(mount string) string {
	f, err := os.Open("/proc/mounts")
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		parts := strings.Fields(sc.Text())
		if len(parts) > 1 && strings.ReplaceAll(parts[1], `\040`, " ") == mount {
			return parts[0]
		}
	}
	return ""
}
