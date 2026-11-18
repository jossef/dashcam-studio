// Package proc runs helper programs (ffmpeg, eject tools) without flashing a
// console window on Windows.
package proc

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
)

// Command is exec.CommandContext with the console window hidden on Windows.
func Command(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	hide(cmd)
	return cmd
}

// Find looks for a program next to the executable first, then on PATH, so
// users can drop e.g. ffmpeg.exe beside the app.
func Find(name string) string {
	if exe, err := os.Executable(); err == nil {
		for _, cand := range []string{name, name + ".exe"} {
			p := filepath.Join(filepath.Dir(exe), cand)
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				return p
			}
		}
	}
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	return ""
}
