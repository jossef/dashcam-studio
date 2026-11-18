// Package gps parses the dashcam's GPSData*.txt log and indexes it per clip.
//
// Line format observed ($V02 header):
//
//	1794537432,A,32.074869,34.825951,17900,194,100,1,-11,NO20261113-143658-004641F.MP4,0,0,0
//	ts        ,fix,lat   ,lon      ,hdg*100,speed cm/s,accel x,y,z (g*100),file,...
//
// "V" (void) lines have no fix and are skipped. The device's timestamps are not
// plain UTC, so the offset between log time and the (local) time in the file
// name is learned from the data itself: the most common offset wins.
package gps

import (
	"bufio"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	fileRE = regexp.MustCompile(`^([A-Za-z]{2,3})(\d{8})-(\d{6})-(\d+)[FRCfrc]\.\w+$`)
	stemRE = regexp.MustCompile(`^[A-Za-z]{2,3}(\d{8})-(\d{6})`)
)

type point struct {
	ts                   int64
	lat, lon, hdg, speed float64 // heading in degrees, speed in km/h
}

// Summary is the per-clip GPS digest shown in the library.
type Summary struct {
	Km     float64 `json:"km"`
	MaxKmh float64 `json:"maxKmh"`
	Lat    float64 `json:"lat"`
	Lon    float64 `json:"lon"`
}

// Index holds every fix, grouped by clip stem.
type Index struct {
	raw    map[string][]point
	order  []string // stems in the order they first appear
	Offset float64  // name epoch - log timestamp, in seconds
	Files  []string

	mu      sync.Mutex
	summary map[string]*Summary
}

// Empty returns an index with no data.
func Empty() *Index {
	return &Index{raw: map[string][]point{}, summary: map[string]*Summary{}}
}

// FindFiles looks for GPS logs in root and up to two parents (root may be E:\Normal).
func FindFiles(root string) []string {
	path, _ := filepath.Abs(root)
	seen := map[string]bool{}
	var out []string
	for range 3 {
		entries, _ := os.ReadDir(path)
		var found []string
		for _, e := range entries {
			name := strings.ToLower(e.Name())
			if !e.IsDir() && strings.HasPrefix(name, "gpsdata") && strings.HasSuffix(name, ".txt") {
				found = append(found, filepath.Join(path, e.Name()))
			}
		}
		sort.Strings(found)
		for _, f := range found {
			if !seen[f] {
				seen[f] = true
				out = append(out, f)
			}
		}
		parent := filepath.Dir(path)
		if parent == path {
			break
		}
		path = parent
	}
	return out
}

// Load parses every GPS log that belongs to root.
func Load(root string) *Index {
	idx := Empty()
	for _, path := range FindFiles(root) {
		idx.Files = append(idx.Files, path)
		idx.parse(path)
	}
	idx.learnOffset()
	return idx
}

func (x *Index) parse(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		p := strings.Split(strings.TrimRight(sc.Text(), "\r"), ",")
		if len(p) < 10 || p[1] != "A" {
			continue
		}
		file := strings.TrimSpace(p[9])
		m := fileRE.FindStringSubmatchIndex(file)
		if m == nil {
			continue
		}
		ts, err1 := strconv.ParseInt(strings.TrimSpace(p[0]), 10, 64)
		lat, err2 := strconv.ParseFloat(strings.TrimSpace(p[2]), 64)
		lon, err3 := strconv.ParseFloat(strings.TrimSpace(p[3]), 64)
		hdg, err4 := strconv.Atoi(strings.TrimSpace(p[4]))
		spd, err5 := strconv.Atoi(strings.TrimSpace(p[5]))
		if err1 != nil || err2 != nil || err3 != nil || err4 != nil || err5 != nil {
			continue
		}
		if lat == 0 && lon == 0 {
			continue
		}
		stem := file[:m[9]] // up to the end of the sequence number
		if _, ok := x.raw[stem]; !ok {
			x.order = append(x.order, stem)
		}
		x.raw[stem] = append(x.raw[stem], point{ts, lat, lon, float64(hdg) / 100, float64(spd) * 0.036}) // cm/s -> km/h
	}
}

// nameEpoch is the wall-clock time in the stem, read as if it were UTC.
func nameEpoch(stem string) (float64, bool) {
	m := stemRE.FindStringSubmatch(stem)
	if m == nil {
		return 0, false
	}
	t, err := time.Parse("20060102150405", m[1]+m[2])
	if err != nil {
		return 0, false
	}
	return float64(t.Unix()), true
}

func (x *Index) learnOffset() {
	votes := map[float64]int{}
	var first []float64 // first-seen order breaks ties
	for _, stem := range x.order {
		ne, ok := nameEpoch(stem)
		if !ok {
			continue
		}
		v := math.RoundToEven((ne-float64(x.raw[stem][0].ts))/900) * 900
		if votes[v] == 0 {
			first = append(first, v)
		}
		votes[v]++
	}
	best := -1
	for _, v := range first {
		if votes[v] > best {
			best, x.Offset = votes[v], v
		}
	}
}

// Len is the number of clips with GPS data.
func (x *Index) Len() int { return len(x.raw) }

func round(v float64, places int) float64 {
	p := math.Pow(10, float64(places))
	return math.Round(v*p) / p
}

// Track returns points as [t_in_clip, lat, lon, speed_kmh, heading], sorted by time.
func (x *Index) Track(stem string) [][5]float64 {
	pts := x.raw[stem]
	out := make([][5]float64, 0, len(pts))
	if len(pts) == 0 {
		return out
	}
	first := float64(pts[0].ts)
	base := first
	// Clips whose clock was unset (e.g. 2025-01-01) don't line up:
	// fall back to "first fix is second 0".
	if ne, ok := nameEpoch(stem); ok {
		if b := ne - x.Offset; first-b >= -5 && first-b <= 90 {
			base = b
		}
	}
	sorted := append([]point(nil), pts...)
	sort.SliceStable(sorted, func(i, j int) bool {
		a, b := sorted[i], sorted[j]
		if a.ts != b.ts {
			return a.ts < b.ts
		}
		if a.lat != b.lat {
			return a.lat < b.lat
		}
		return a.lon < b.lon
	})
	for _, p := range sorted {
		out = append(out, [5]float64{round(float64(p.ts)-base, 2), p.lat, p.lon, round(p.speed, 1), round(p.hdg, 1)})
	}
	return out
}

// Haversine returns the distance between two coordinates in meters.
func Haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const r = 6371000.0
	rad := math.Pi / 180
	p1, p2 := lat1*rad, lat2*rad
	dp, dl := p2-p1, (lon2-lon1)*rad
	a := math.Pow(math.Sin(dp/2), 2) + math.Cos(p1)*math.Cos(p2)*math.Pow(math.Sin(dl/2), 2)
	return 2 * r * math.Asin(math.Sqrt(math.Min(1, a)))
}

// Summary returns distance, top speed and a representative position, or nil.
func (x *Index) Summary(stem string) *Summary {
	x.mu.Lock()
	defer x.mu.Unlock()
	if s, ok := x.summary[stem]; ok {
		return s
	}
	pts := x.raw[stem]
	if len(pts) == 0 {
		return nil
	}
	dist, maxKmh := 0.0, pts[0].speed
	for i, p := range pts {
		maxKmh = math.Max(maxKmh, p.speed)
		if i > 0 {
			if d := Haversine(pts[i-1].lat, pts[i-1].lon, p.lat, p.lon); d < 200 { // ignore GPS jumps
				dist += d
			}
		}
	}
	mid := pts[len(pts)/2]
	s := &Summary{Km: round(dist/1000, 3), MaxKmh: round(maxKmh, 1), Lat: mid.lat, Lon: mid.lon}
	x.summary[stem] = s
	return s
}
