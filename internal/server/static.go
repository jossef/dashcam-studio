package server

import (
	"io/fs"
	"net/http"
	"path"
	"strconv"
	"strings"
)

// Explicit types: on Windows the registry can map .js to text/plain.
var contentTypes = map[string]string{
	".html": "text/html; charset=utf-8", ".js": "text/javascript; charset=utf-8", ".mjs": "text/javascript; charset=utf-8",
	".css": "text/css; charset=utf-8", ".json": "application/json; charset=utf-8", ".svg": "image/svg+xml",
	".png": "image/png", ".jpg": "image/jpeg", ".ico": "image/x-icon", ".woff": "font/woff", ".woff2": "font/woff2",
	".txt": "text/plain; charset=utf-8", ".map": "application/json; charset=utf-8", ".webp": "image/webp",
}

func (s *Server) serveStatic(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/")
	if name == "" {
		name = "index.html"
	}
	if !fs.ValidPath(name) || s.static == nil {
		writeError(w, 404, "Not found")
		return
	}
	body, err := fs.ReadFile(s.static, name)
	if err != nil {
		writeError(w, 404, "Not found")
		return
	}
	ctype := contentTypes[strings.ToLower(path.Ext(name))]
	if ctype == "" {
		ctype = "application/octet-stream"
	}
	h := w.Header()
	h.Set("Content-Type", ctype)
	h.Set("Content-Length", strconv.Itoa(len(body)))
	if strings.HasPrefix(name, "assets/") {
		h.Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		h.Set("Cache-Control", "no-cache")
	}
	w.WriteHeader(200)
	if r.Method != http.MethodHead {
		_, _ = w.Write(body)
	}
}
