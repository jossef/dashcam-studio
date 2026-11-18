package volume

import (
	"strings"
	"time"
)

// IsRemovable treats every volume under /Volumes as an external drive.
func IsRemovable(path string) bool {
	return strings.HasPrefix(MountPoint(path), "/Volumes/")
}

// Eject runs `diskutil eject` on the volume.
func Eject(path string) (bool, string) {
	if !IsRemovable(path) {
		return false, "not a removable drive"
	}
	mount := MountPoint(path)
	ok, msg := run(25*time.Second, "diskutil", "eject", mount)
	return ok && gone(mount, 5*time.Second), msg
}
