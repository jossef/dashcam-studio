package desktop

import (
	"os/exec"
	"runtime"
)

// openBrowser opens url in the default browser. It uses exec directly, not
// proc.Command, which would pass "hidden" on to the browser's window.
func openBrowser(url string) error {
	var name string
	var args []string
	switch runtime.GOOS {
	case "windows":
		name, args = "rundll32", []string{"url.dll,FileProtocolHandler", url}
	case "darwin":
		name, args = "open", []string{url}
	default:
		name, args = "xdg-open", []string{url}
	}
	return exec.Command(name, args...).Start()
}
