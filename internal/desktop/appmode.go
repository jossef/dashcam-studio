package desktop

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// chromiumCandidates lists Chromium-family browsers that support --app.
func chromiumCandidates() []string {
	switch runtime.GOOS {
	case "windows":
		var out []string
		for _, env := range []string{"ProgramFiles(x86)", "ProgramFiles", "LocalAppData"} {
			base := os.Getenv(env)
			if base == "" {
				continue
			}
			out = append(out,
				filepath.Join(base, `Microsoft\Edge\Application\msedge.exe`),
				filepath.Join(base, `Google\Chrome\Application\chrome.exe`),
				filepath.Join(base, `BraveSoftware\Brave-Browser\Application\brave.exe`),
				filepath.Join(base, `Chromium\Application\chrome.exe`))
		}
		return out
	case "darwin":
		home, _ := os.UserHomeDir()
		var out []string
		for _, app := range []string{"Google Chrome", "Microsoft Edge", "Brave Browser", "Chromium", "Vivaldi"} {
			for _, root := range []string{"/Applications", filepath.Join(home, "Applications")} {
				out = append(out, filepath.Join(root, app+".app", "Contents", "MacOS", app))
			}
		}
		return out
	default:
		var out []string
		for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser",
			"microsoft-edge", "microsoft-edge-stable", "brave-browser", "vivaldi"} {
			if p, err := exec.LookPath(name); err == nil {
				out = append(out, p)
			}
		}
		return out
	}
}

func findChromium() string {
	for _, p := range chromiumCandidates() {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

// runAppMode opens the UI in a Chromium --app window with its own profile, so
// it runs as a separate browser process whose exit means "window closed".
func runAppMode(ctx context.Context, o Options) error {
	bin := findChromium()
	if bin == "" {
		return ErrUnavailable
	}
	profile := filepath.Join(o.DataDir, "app-window")
	args := []string{
		"--app=" + o.URL,
		"--user-data-dir=" + profile,
		"--no-first-run", "--no-default-browser-check",
		"--window-size=1600,950",
		"--disable-features=Translate",
	}
	// Not proc.Command: that hides the window, and the browser would honour it.
	cmd := exec.Command(bin, args...)
	if err := cmd.Start(); err != nil {
		return ErrUnavailable
	}
	started := time.Now()
	done := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(done)
	}()
	select {
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		return nil
	case <-done:
	}
	// A very quick exit means the profile was already open in another window,
	// which took over our URL. Keep serving while that page uses the server.
	if time.Since(started) < 3*time.Second {
		waitForPage(ctx, Options{Idle: o.Idle})
	}
	return nil
}
