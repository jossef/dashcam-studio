package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jossef/dashcam-studio/internal/card"
	"github.com/jossef/dashcam-studio/internal/favorites"
	"github.com/jossef/dashcam-studio/internal/gps"
	"github.com/jossef/dashcam-studio/internal/media"
	"github.com/jossef/dashcam-studio/internal/paths"
	"github.com/jossef/dashcam-studio/internal/volume"
)

// ErrMediaGone means the card was pulled out, or closed by the user.
var ErrMediaGone = errors.New("media disconnected")

// WatchInterval is the delay between "is the card still there?" checks.
const WatchInterval = 1500 * time.Millisecond

// Connection states reported to the UI.
const (
	ConnNone      = "none"      // nothing opened yet
	ConnConnected = "connected" // card open and readable
	ConnLost      = "lost"      // card vanished; reopened automatically when it returns
	ConnClosed    = "closed"    // user disconnected; not reopened automatically
)

type exportJob struct {
	at   time.Time
	ids  []string
	cams string
	gpx  bool
}

// State is everything the server knows about the open card. It is safe for
// concurrent use.
type State struct {
	mu        sync.Mutex
	root      string
	lib       *card.Library
	gps       *gps.Index
	loading   bool
	err       string
	fav       *favorites.Store
	exports   map[string]exportJob
	conn      string
	lastRoot  string
	removable bool
	label     string

	// Every reader of the card holds a lease. Release cancels genCtx, which
	// makes streams stop and ffmpeg exit, then waits for busy to reach zero.
	genCtx    context.Context
	cancelGen context.CancelFunc
	busy      atomic.Int64
}

// NewState returns an empty state (no card open).
func NewState() *State {
	s := &State{gps: gps.Empty(), exports: map[string]exportJob{}, conn: ConnNone}
	s.genCtx, s.cancelGen = context.WithCancel(context.Background())
	return s
}

// ------------------------------------------------------------- card access

// Lease is held while a request reads the card.
type Lease struct {
	s    *State
	ctx  context.Context
	once sync.Once
}

// Context is cancelled when the card must be let go of.
func (l *Lease) Context() context.Context { return l.ctx }

// Aborted is true once a disconnect asked readers to stop.
func (l *Lease) Aborted() bool { return l.ctx.Err() != nil }

// Done returns the lease.
func (l *Lease) Done() { l.once.Do(func() { l.s.busy.Add(-1) }) }

// UseCard takes a lease, or fails with ErrMediaGone when no card is connected.
func (s *State) UseCard() (*Lease, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn != ConnConnected {
		return nil, ErrMediaGone
	}
	s.busy.Add(1)
	return &Lease{s: s, ctx: s.genCtx}, nil
}

// Busy is the number of active card readers.
func (s *State) Busy() int { return int(s.busy.Load()) }

// Release stops streams, kills transcoders and waits until no file on the
// card is open. It returns false if readers were still busy at the timeout.
func (s *State) Release(timeout time.Duration) bool {
	s.mu.Lock()
	s.cancelGen()
	s.genCtx, s.cancelGen = context.WithCancel(context.Background())
	s.mu.Unlock()
	end := time.Now().Add(timeout)
	for s.busy.Load() > 0 && time.Now().Before(end) {
		time.Sleep(20 * time.Millisecond)
	}
	return s.busy.Load() == 0
}

// Close is a user-initiated disconnect. It returns the card root and whether
// every file was released.
func (s *State) Close() (string, bool) {
	s.mu.Lock()
	root := s.root
	if root == "" {
		root = s.lastRoot
	}
	s.conn, s.lastRoot = ConnClosed, root
	s.mu.Unlock()
	released := s.Release(8 * time.Second)
	s.mu.Lock()
	s.root, s.lib, s.gps = "", nil, gps.Empty()
	clear(s.exports)
	s.mu.Unlock()
	return root, released
}

// CheckConnection is one watcher step: notice a pulled card, or reopen it
// when it comes back. It returns "lost", "reconnected" or "".
func (s *State) CheckConnection() string {
	s.mu.Lock()
	conn, root, last := s.conn, s.root, s.lastRoot
	s.mu.Unlock()
	if conn == ConnConnected && root != "" && !volume.Present(root) {
		s.mu.Lock()
		if s.conn != ConnConnected {
			s.mu.Unlock()
			return ""
		}
		s.conn = ConnLost
		s.mu.Unlock()
		s.Release(2 * time.Second)
		return "lost"
	}
	if conn == ConnLost && last != "" && volume.Present(last) {
		if err := s.Open(last); err != nil {
			return "" // still mounting, or a different card
		}
		return "reconnected"
	}
	return ""
}

// Watch runs CheckConnection every interval until ctx ends.
func (s *State) Watch(ctx context.Context, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			func() {
				defer func() { _ = recover() }() // never let the watcher die
				s.CheckConnection()
			}()
		}
	}
}

// ------------------------------------------------------------ open / status

// UserError is a problem worth showing to the user as-is (HTTP 400).
type UserError struct{ Msg string }

func (e *UserError) Error() string { return e.Msg }

// Open scans root and makes it the current card.
func (s *State) Open(root string) error {
	root = strings.Trim(strings.TrimSpace(root), `"`)
	if strings.HasPrefix(root, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			root = filepath.Join(home, root[1:])
		}
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		abs = root
	}
	s.setLoading(true, "")
	err = s.open(abs)
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	s.setLoading(false, msg)
	return err
}

func (s *State) open(root string) error {
	if st, err := os.Stat(root); err != nil || !st.IsDir() {
		return &UserError{"Folder not found: " + root}
	}
	if _, err := os.ReadDir(root); err != nil {
		return &UserError{fmt.Sprintf("Can't read folder: %v", err)}
	}
	lib := card.Scan(root)
	if lib.Len() == 0 {
		return &UserError{"No dashcam clips found here. Pick the drive root or a folder that contains " +
			"Front / Rear / Cabin sub-folders."}
	}
	g := gps.Load(root)
	removable := volume.IsRemovable(root)
	label := ""
	if removable {
		label = volume.Label(volume.MountPoint(root))
	}
	s.mu.Lock()
	s.root, s.lib, s.gps = root, lib, g
	s.conn, s.lastRoot = ConnConnected, root
	s.removable, s.label = removable, label
	s.mu.Unlock()
	return nil
}

func (s *State) setLoading(on bool, err string) {
	s.mu.Lock()
	s.loading = on
	if !on {
		s.err = err
	} else {
		s.err = ""
	}
	s.mu.Unlock()
}

// Root is the open card folder ("" when none).
func (s *State) Root() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.root
}

// snapshot returns the current library (nil when none) and GPS index.
func (s *State) snapshot() (*card.Library, *gps.Index) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lib, s.gps
}

// Status is what /api/status reports.
type Status struct {
	Root       *string  `json:"root"`
	Loading    bool     `json:"loading"`
	Error      *string  `json:"error"`
	FFmpeg     bool     `json:"ffmpeg"`
	GPSFiles   []string `json:"gpsFiles"`
	ClipCount  int      `json:"clipCount"`
	Platform   string   `json:"platform"`
	Sep        string   `json:"sep"`
	Connection string   `json:"connection"`
	LastRoot   *string  `json:"lastRoot"`
	Removable  bool     `json:"removable"`
	Label      string   `json:"label"`
}

func strPtr(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

// Platform uses the same names as the original Python server (sys.platform).
func platform() string {
	if runtime.GOOS == "windows" {
		return "win32"
	}
	return runtime.GOOS
}

// Status reports the current state.
func (s *State) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	files := []string{}
	for _, f := range s.gps.Files {
		files = append(files, filepath.Base(f))
	}
	count := 0
	if s.lib != nil {
		count = s.lib.Len()
	}
	return Status{
		Root: strPtr(s.root), Loading: s.loading, Error: strPtr(s.err), FFmpeg: media.FFmpeg != "",
		GPSFiles: files, ClipCount: count, Platform: platform(), Sep: string(filepath.Separator),
		Connection: s.conn, LastRoot: strPtr(s.lastRoot), Removable: s.removable, Label: s.label,
	}
}

// --------------------------------------------------------- favorites/exports

// Favorites opens the favorites store on first use.
func (s *State) Favorites() *favorites.Store {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fav == nil {
		s.fav = favorites.Open(filepath.Join(paths.ConfigDir(), "favorites.json"))
	}
	return s.fav
}

func (s *State) addExport(ids []string, cams string, withGPX bool) string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	token := base64.RawURLEncoding.EncodeToString(b)
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range s.exports {
		if now.Sub(v.at) >= 6*time.Hour {
			delete(s.exports, k)
		}
	}
	s.exports[token] = exportJob{now, ids, cams, withGPX}
	return token
}

func (s *State) export(token string) (exportJob, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.exports[token]
	return job, ok
}
