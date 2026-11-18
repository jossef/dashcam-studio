// Package volume finds dashcam cards, tells whether a folder lives on a
// removable drive, and ejects that drive safely.
//
// Ejecting only asks the operating system to unmount the volume (like "Safely
// remove hardware"). It never writes to the card. Callers must release every
// open file first, or the OS refuses because the device is busy.
package volume

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jossef/dashcam-studio/internal/card"
	"github.com/jossef/dashcam-studio/internal/proc"
)

// Drive is a mounted folder that looks like a dashcam card.
type Drive struct {
	Path string `json:"path"`
	Name string `json:"name"`
}

// Detect lists mounted volumes that look like a dashcam card.
func Detect() []Drive {
	out := []Drive{}
	for _, path := range candidates() {
		if card.LooksLikeDashcamRoot(path) {
			out = append(out, Drive{Path: path, Name: Label(path)})
		}
	}
	return out
}

// Present reports whether a folder can still be listed (false once a card is pulled).
func Present(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	_, err = f.Readdirnames(1)
	return err == nil || errors.Is(err, io.EOF)
}

func baseName(path string) string {
	name := filepath.Base(strings.TrimRight(path, `\/`))
	if name == "." || name == "" || name == `\` || name == "/" {
		return path
	}
	return name
}

func gone(mount string, timeout time.Duration) bool {
	end := time.Now().Add(timeout)
	for time.Now().Before(end) {
		if st, err := os.Stat(mount); err != nil || !st.IsDir() {
			return true
		}
		time.Sleep(250 * time.Millisecond)
	}
	st, err := os.Stat(mount)
	return err != nil || !st.IsDir()
}

// run executes a helper with a timeout, returning success and its message.
func run(timeout time.Duration, name string, args ...string) (bool, string) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	out, err := proc.Command(ctx, name, args...).CombinedOutput()
	msg := strings.TrimSpace(string(out))
	if err != nil && msg == "" {
		msg = err.Error()
	}
	return err == nil, msg
}
