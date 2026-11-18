package volume

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/windows"
)

const driveRemovable, driveFixed = 2, 3

func systemDrive() string {
	if d := os.Getenv("SystemDrive"); len(d) >= 2 {
		return strings.ToUpper(d[:2])
	}
	return "C:"
}

func driveType(root string) uint32 {
	p, err := windows.UTF16PtrFromString(root)
	if err != nil {
		return 0
	}
	return windows.GetDriveType(p)
}

func candidates() []string {
	var out []string
	mask, err := windows.GetLogicalDrives()
	if err != nil {
		return out
	}
	for i := range 26 {
		if mask&(1<<i) == 0 {
			continue
		}
		root := fmt.Sprintf("%c:\\", 'A'+i)
		if strings.EqualFold(root[:2], systemDrive()) {
			continue
		}
		if t := driveType(root); t == driveRemovable || t == driveFixed {
			out = append(out, root)
		}
	}
	return out
}

// Roots are the drive letters shown at the top of the folder browser.
func Roots() []string {
	var out []string
	mask, _ := windows.GetLogicalDrives()
	for i := range 26 {
		if mask&(1<<i) != 0 {
			out = append(out, fmt.Sprintf("%c:\\", 'A'+i))
		}
	}
	return out
}

// Label is the volume name (e.g. "DASHCAM"), or the path when there is none.
func Label(path string) string {
	root := MountPoint(path)
	p, err := windows.UTF16PtrFromString(root)
	if err == nil {
		buf := make([]uint16, 256)
		if windows.GetVolumeInformation(p, &buf[0], uint32(len(buf)), nil, nil, nil, nil, 0) == nil {
			if name := windows.UTF16ToString(buf); name != "" {
				return name
			}
			return path
		}
	}
	return baseName(path)
}

// MountPoint is the drive root holding path (E:\ or \\server\share\).
func MountPoint(path string) string {
	abs, _ := filepath.Abs(path)
	if vol := filepath.VolumeName(abs); vol != "" {
		return vol + `\`
	}
	return abs
}

// IsRemovable is true for USB sticks and SD/microSD readers (not network shares or the system drive).
func IsRemovable(path string) bool {
	mount := MountPoint(path)
	if strings.HasPrefix(mount, `\\`) {
		return false
	}
	return !strings.EqualFold(mount[:2], systemDrive()) && driveType(mount) == driveRemovable
}

// Eject does what Explorer's "Eject" does on the drive (works for USB sticks and card readers).
func Eject(path string) (bool, string) {
	mount := MountPoint(path)
	if !IsRemovable(path) {
		return false, "not a removable drive"
	}
	drive := mount[:2]
	script := fmt.Sprintf("$d=(New-Object -ComObject Shell.Application).Namespace(17).ParseName('%s');"+
		"if ($d) { $d.InvokeVerb('Eject') } else { exit 2 }", drive)
	run(25*time.Second, "powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	if gone(mount, 10*time.Second) {
		return true, ""
	}
	return false, "Windows did not eject the drive - another program may be using it"
}
