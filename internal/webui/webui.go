// Package webui embeds the built React app (web/ -> internal/webui/ui/dist)
// into the executable.
//
// `go run ./tools/build` (or `npm run build` in web/) writes the bundle to
// ui/dist. Until then the server shows a placeholder page explaining how to
// build the UI, so Go code always compiles and tests run without Node.
package webui

import (
	"embed"
	"io/fs"
)

//go:embed all:ui
var files embed.FS

// FS returns the built UI, or a placeholder when it has not been built yet.
func FS() fs.FS {
	if dist, err := fs.Sub(files, "ui/dist"); err == nil {
		if _, err := fs.Stat(dist, "index.html"); err == nil {
			return dist
		}
	}
	placeholder, _ := fs.Sub(files, "ui/placeholder")
	return placeholder
}

// Built reports whether the real UI bundle is embedded.
func Built() bool {
	_, err := fs.Stat(files, "ui/dist/index.html")
	return err == nil
}
