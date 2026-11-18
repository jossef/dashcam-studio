package main

import (
	"log"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/jossef/dashcam-studio/internal/paths"
)

var (
	kernel32             = windows.NewLazySystemDLL("kernel32.dll")
	procAttachConsole    = kernel32.NewProc("AttachConsole")
	procGetConsoleWindow = kernel32.NewProc("GetConsoleWindow")
	procMessageBox       = windows.NewLazySystemDLL("user32.dll").NewProc("MessageBoxW")
)

// attachConsole makes output visible when the GUI-subsystem release build is
// started from a terminal (cmd / PowerShell). Double-clicked, there is no
// console, so the log goes to a file in the cache folder instead. It reports
// whether a terminal is attached.
func attachConsole() bool {
	if h, _, _ := procGetConsoleWindow.Call(); h != 0 {
		return true // console build (go run / go build without -H windowsgui)
	}
	if out, err := windows.GetStdHandle(windows.STD_OUTPUT_HANDLE); err == nil && out != 0 && out != windows.InvalidHandle {
		if t, _ := windows.GetFileType(out); t == windows.FILE_TYPE_PIPE || t == windows.FILE_TYPE_DISK {
			return true // output is redirected (a script or pipe captures it): keep it
		}
	}
	const attachParentProcess = ^uintptr(0) // (DWORD)-1
	if r, _, _ := procAttachConsole.Call(attachParentProcess); r != 0 {
		if out, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0); err == nil {
			os.Stdout, os.Stderr = out, out
			log.SetOutput(out)
			os.Stdout.WriteString("\n")
			return true
		}
	}
	logToFile()
	return false
}

func logToFile() {
	if f, err := os.Create(filepath.Join(paths.CacheDir(), "dashcam-studio.log")); err == nil {
		os.Stdout, os.Stderr = f, f
		log.SetOutput(f)
	}
}

// fatalDialog shows a startup error to a user who has no console to read it in.
func fatalDialog(msg string) {
	text, _ := windows.UTF16PtrFromString(msg)
	title, _ := windows.UTF16PtrFromString("Dashcam Studio")
	const mbIconError = 0x10
	procMessageBox.Call(0, uintptr(unsafe.Pointer(text)), uintptr(unsafe.Pointer(title)), mbIconError)
}
