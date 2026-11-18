// Package server is the local HTTP API and media server behind the UI.
//
// It binds to 127.0.0.1 by default. Media is addressed by clip id
// ("<category>/<stem>"), never by raw path, and the card is only ever read.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

// Server routes requests to the API, media streams and the embedded UI.
type Server struct {
	State  *State
	static fs.FS
	mux    *http.ServeMux
	Debug  bool

	lastHit atomic.Int64 // unix nanos of the last request
	active  atomic.Int64 // requests in flight
}

// New builds the server. static holds the built UI (index.html + assets/).
func New(state *State, static fs.FS) *Server {
	s := &Server{State: state, static: static, mux: http.NewServeMux()}
	s.lastHit.Store(time.Now().UnixNano())
	s.routes()
	return s
}

func (s *Server) routes() {
	m := s.mux
	m.HandleFunc("GET /api/status", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, s.State.Status()) })
	m.HandleFunc("GET /api/library", s.library)
	m.HandleFunc("GET /api/tracks", s.tracks)
	m.HandleFunc("GET /api/favorites", s.favorites)
	m.HandleFunc("GET /api/drives", s.drives)
	m.HandleFunc("GET /api/browse", s.browse)
	m.HandleFunc("GET /api/clip", s.card(s.clip))
	m.HandleFunc("GET /api/thumb", s.card(s.thumb))
	m.HandleFunc("GET /media", s.card(s.media))
	m.HandleFunc("GET /api/transcode", s.card(s.transcode))
	m.HandleFunc("GET /api/zip", s.card(s.zip))

	m.HandleFunc("POST /api/open", s.open)
	m.HandleFunc("POST /api/favorites", s.setFavorites)
	m.HandleFunc("POST /api/export", s.export)
	m.HandleFunc("POST /api/disconnect", s.disconnect)
	m.HandleFunc("POST /api/rescan", s.rescan)

	m.HandleFunc("GET /", s.serveStatic)
	notFound := func(w http.ResponseWriter, r *http.Request) { writeError(w, 404, "Not found") }
	m.HandleFunc("GET /api/", notFound)
	m.HandleFunc("POST /", notFound)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.lastHit.Store(time.Now().UnixNano())
	s.active.Add(1)
	defer func() {
		s.active.Add(-1)
		s.lastHit.Store(time.Now().UnixNano())
		if v := recover(); v != nil && v != http.ErrAbortHandler {
			log.Printf("panic serving %s: %v", r.URL.Path, v)
		}
	}()
	if s.Debug {
		log.Printf("%s %s", r.Method, r.URL.RequestURI())
	}
	s.mux.ServeHTTP(w, r)
}

// IdleSince reports how long no request has been running.
func (s *Server) IdleSince() time.Duration {
	if s.active.Load() > 0 {
		return 0
	}
	return time.Since(time.Unix(0, s.lastHit.Load()))
}

// Listen binds host:port, trying the next ports when it is taken (port 0 = any free port).
func Listen(host string, port, tries int) (net.Listener, error) {
	var last error
	for p := port; p < port+max(1, tries); p++ {
		ln, err := net.Listen("tcp", net.JoinHostPort(host, fmt.Sprint(p)))
		if err == nil {
			return ln, nil
		}
		last = err
		if port == 0 {
			break
		}
	}
	return nil, fmt.Errorf("no free port from %d: %w", port, last)
}

// Serve runs the HTTP server on ln until ctx is cancelled.
func (s *Server) Serve(ctx context.Context, ln net.Listener) error {
	srv := &http.Server{Handler: s, ReadHeaderTimeout: 10 * time.Second, ErrorLog: log.New(io.Discard, "", 0)}
	go func() {
		<-ctx.Done()
		s.State.Release(2 * time.Second) // let go of the card and stop ffmpeg
		shut, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shut)
		_ = srv.Close()
	}()
	if err := srv.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// ------------------------------------------------------------------ helpers

func writeJSON(w http.ResponseWriter, status int, v any) {
	body, err := json.Marshal(v)
	if err != nil {
		status, body = 500, []byte(`{"error":"encoding failed"}`)
	}
	h := w.Header()
	h.Set("Content-Type", "application/json; charset=utf-8")
	h.Set("Content-Length", fmt.Sprint(len(body)))
	h.Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeDisconnected(w http.ResponseWriter) {
	w.Header().Set("Connection", "close")
	writeJSON(w, 503, map[string]string{"error": "Media disconnected", "code": "disconnected"})
}

// readJSON decodes a POST body into a map (an empty body is {}).
func readJSON(r *http.Request) (map[string]any, error) {
	data := map[string]any{}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		return data, nil
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	return data, nil
}

// asciiName keeps a download file name safe for the plain filename= parameter.
func asciiName(name string) string {
	var b strings.Builder
	under := false
	for _, r := range name {
		ok := r < 128 && (r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '.' || r == '_' || r == '-')
		if ok {
			b.WriteRune(r)
			under = false
		} else if !under {
			b.WriteByte('_')
			under = true
		}
	}
	return b.String()
}

// ------------------------------------------------------------- card routes

type cardHandler func(w http.ResponseWriter, r *http.Request, l *Lease) error

// trackingWriter remembers whether a response was started.
type trackingWriter struct {
	http.ResponseWriter
	started bool
}

func (t *trackingWriter) WriteHeader(code int) {
	t.started = true
	t.ResponseWriter.WriteHeader(code)
}

func (t *trackingWriter) Write(p []byte) (int, error) {
	t.started = true
	return t.ResponseWriter.Write(p)
}

func (t *trackingWriter) Unwrap() http.ResponseWriter { return t.ResponseWriter }

// card runs a route that reads the card. While it runs, the request holds a
// lease; a disconnect cancels the lease, which breaks any write that is
// blocked on a browser that stopped reading, so the card is always let go of
// quickly. A vanished card becomes a clean 503 {"code": "disconnected"}.
func (s *Server) card(h cardHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lease, err := s.State.UseCard()
		if err != nil {
			writeDisconnected(w)
			return
		}
		defer lease.Done()
		tw := &trackingWriter{ResponseWriter: w}
		rc := http.NewResponseController(tw)
		stop := context.AfterFunc(lease.Context(), func() { _ = rc.SetWriteDeadline(time.Now()) })
		defer stop()
		err = h(tw, r, lease)
		if err == nil {
			return
		}
		var pe *fs.PathError
		if errors.As(err, &pe) {
			// a read failed - maybe the card was just pulled; let the watcher decide now
			go s.State.CheckConnection()
		}
		if !tw.started {
			writeDisconnected(tw)
		}
	}
}

var errAborted = errors.New("card released")

// leaseWriter refuses to write once the lease is cancelled.
type leaseWriter struct {
	w io.Writer
	l *Lease
}

func (lw leaseWriter) Write(p []byte) (int, error) {
	if lw.l.Aborted() {
		return 0, errAborted
	}
	return lw.w.Write(p)
}

// leaseReader refuses to read once the lease is cancelled. It deliberately
// hides the *os.File so net/http cannot use sendfile, which a deadline cannot
// interrupt.
type leaseReader struct {
	f interface {
		io.Reader
		io.Seeker
	}
	l *Lease
}

func (lr leaseReader) Read(p []byte) (int, error) {
	if lr.l.Aborted() {
		return 0, errAborted
	}
	return lr.f.Read(p)
}

func (lr leaseReader) Seek(offset int64, whence int) (int64, error) { return lr.f.Seek(offset, whence) }
