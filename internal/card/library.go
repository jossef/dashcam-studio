// Package card scans a dashcam card and groups matching Front / Rear / Cabin
// files into clips.
//
// Expected layout (camera folders up to 3 levels below the root):
//
//	<root>/<Category>/{Front,Rear,Cabin}/NO20261109-191423-003968F.MP4
//
// The root may also be a single category folder (e.g. E:\Normal) or the drive
// root (e.g. E:\). Nothing on the card is ever written.
package card

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Cameras in display order.
var Cameras = []string{"F", "R", "C"}

// CameraNames maps a camera letter to its English name.
var CameraNames = map[string]string{"F": "Front", "R": "Rear", "C": "Cabin"}

var cameraDirs = map[string]string{"front": "F", "rear": "R", "cabin": "C", "inside": "C", "interior": "C"}

var prefixLabels = map[string]string{"NO": "Normal", "EV": "Event", "PA": "Parking", "PK": "Parking", "LA": "Time-lapse"}

// NO20261109-191423-003968F.MP4 -> prefix, date, time, seq, cam, ext
var nameRE = regexp.MustCompile(`(?i)^([a-z]{2,3})(\d{8})-(\d{6})-(\d+)([frc])\.(mp4|mov|ts)$`)

// StartLayout is the file-name timestamp layout (local wall-clock time).
const StartLayout = "20060102150405"

// Name is a parsed clip file name.
type Name struct {
	Prefix string
	Start  *time.Time // nil when the date in the name is invalid
	Seq    int
	Cam    string // upper-case camera letter
	Stem   string // file name without camera letter and extension
}

// ParseName parses a dashcam file name, returning false for unrelated files.
func ParseName(filename string) (Name, bool) {
	m := nameRE.FindStringSubmatchIndex(filename)
	if m == nil {
		return Name{}, false
	}
	part := func(i int) string { return filename[m[2*i]:m[2*i+1]] }
	n := Name{Prefix: part(1), Cam: strings.ToUpper(part(5)), Stem: filename[:m[10]]}
	n.Seq, _ = strconv.Atoi(part(4))
	if t, err := time.Parse(StartLayout, part(2)+part(3)); err == nil {
		n.Start = &t
	}
	return n, true
}

// Clip is one recording: up to three camera files that share a stem.
type Clip struct {
	ID       string
	Category string
	Stem     string
	Prefix   string
	Start    *time.Time
	Seq      int
	Files    map[string]string // cam -> absolute path
	Sizes    map[string]int64
}

// Kind is the human label of the clip prefix (Normal, Event, ...).
func (c *Clip) Kind() string {
	p := strings.ToUpper(c.Prefix)
	if l, ok := prefixLabels[p]; ok {
		return l
	}
	return p
}

// JSON is the clip as sent to the UI.
func (c *Clip) JSON() map[string]any {
	var start any
	if c.Start != nil {
		start = c.Start.Format("2006-01-02T15:04:05")
	}
	cams := map[string]int64{}
	for _, cam := range Cameras {
		if _, ok := c.Files[cam]; ok {
			cams[cam] = c.Sizes[cam]
		}
	}
	return map[string]any{
		"id": c.ID, "category": c.Category, "stem": c.Stem, "prefix": c.Prefix,
		"kind": c.Kind(), "start": start, "seq": c.Seq, "cams": cams,
	}
}

// Less orders clips chronologically (unknown dates first), then by sequence number.
func (c *Clip) Less(o *Clip) bool {
	a, b := c.startOrZero(), o.startOrZero()
	if !a.Equal(b) {
		return a.Before(b)
	}
	if c.Seq != o.Seq {
		return c.Seq < o.Seq
	}
	return c.Category < o.Category
}

func (c *Clip) startOrZero() time.Time {
	if c.Start == nil {
		return time.Time{}
	}
	return *c.Start
}

type camDir struct{ category, cam, path string }

// cameraDirs finds every camera folder under root, at most maxDepth levels deep.
func findCameraDirs(root string, maxDepth int, stopAtFirst bool) []camDir {
	root, _ = filepath.Abs(root)
	rootName := filepath.Base(strings.TrimRight(root, `\/`))
	if rootName == "." || rootName == "" || rootName == `\` || rootName == "/" || strings.HasSuffix(rootName, ":") {
		rootName = root
	}
	var out []camDir
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() && path != root {
				return fs.SkipDir
			}
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		depth := 0
		if rel != "." {
			depth = strings.Count(rel, string(filepath.Separator)) + 1
			name := strings.ToLower(d.Name())
			if strings.HasPrefix(name, ".") || name == "system volume information" || name == "$recycle.bin" {
				return fs.SkipDir
			}
			if cam, ok := cameraDirs[name]; ok {
				category := rootName
				if parent := filepath.Dir(rel); parent != "." {
					category = filepath.ToSlash(parent)
				}
				out = append(out, camDir{category, cam, path})
				if stopAtFirst {
					return fs.SkipAll
				}
				return fs.SkipDir
			}
		}
		if depth >= maxDepth {
			return fs.SkipDir
		}
		return nil
	})
	return out
}

// LooksLikeDashcamRoot is a cheap check: are there camera folders within 2 levels?
func LooksLikeDashcamRoot(path string) bool {
	return len(findCameraDirs(path, 2, true)) > 0
}

// Library is the set of clips found on a card.
type Library struct {
	Root  string
	mu    sync.RWMutex
	clips map[string]*Clip
}

// Scan reads the card (read-only) and returns its library.
func Scan(root string) *Library {
	abs, _ := filepath.Abs(root)
	clips := map[string]*Clip{}
	for _, cd := range findCameraDirs(abs, 3, false) {
		entries, err := os.ReadDir(cd.path)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			n, ok := ParseName(e.Name())
			if !ok {
				continue
			}
			cam := n.Cam
			if cam == "" {
				cam = cd.cam
			}
			id := cd.category + "/" + n.Stem
			clip := clips[id]
			if clip == nil {
				clip = &Clip{ID: id, Category: cd.category, Stem: n.Stem, Prefix: n.Prefix, Start: n.Start, Seq: n.Seq,
					Files: map[string]string{}, Sizes: map[string]int64{}}
				clips[id] = clip
			}
			clip.Files[cam] = filepath.Join(cd.path, e.Name())
			if info, err := e.Info(); err == nil {
				clip.Sizes[cam] = info.Size()
			} else {
				clip.Sizes[cam] = 0
			}
		}
	}
	return &Library{Root: abs, clips: clips}
}

// Len is the number of clips.
func (l *Library) Len() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.clips)
}

// Get returns a clip by id, or nil.
func (l *Library) Get(id string) *Clip {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.clips[id]
}

// Sorted returns all clips, oldest first.
func (l *Library) Sorted() []*Clip {
	l.mu.RLock()
	out := make([]*Clip, 0, len(l.clips))
	for _, c := range l.clips {
		out = append(out, c)
	}
	l.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].Less(out[j]) })
	return out
}

// Category is a folder name and its clip count.
type Category struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// Categories returns the categories ordered Normal, Event, Parking, Lapse, others.
func (l *Library) Categories() []Category {
	counts := map[string]int{}
	l.mu.RLock()
	for _, c := range l.clips {
		counts[c.Category]++
	}
	l.mu.RUnlock()
	order := map[string]int{"normal": 0, "event": 1, "parking": 2, "lapse": 3}
	rank := func(name string) int {
		parts := strings.Split(name, "/")
		if r, ok := order[strings.ToLower(parts[len(parts)-1])]; ok {
			return r
		}
		return 9
	}
	out := make([]Category, 0, len(counts))
	for k, v := range counts {
		out = append(out, Category{k, v})
	}
	sort.Slice(out, func(i, j int) bool {
		ri, rj := rank(out[i].Name), rank(out[j].Name)
		if ri != rj {
			return ri < rj
		}
		return out[i].Name < out[j].Name
	})
	return out
}
