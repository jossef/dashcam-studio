package server

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jossef/dashcam-studio/internal/card"
	"github.com/jossef/dashcam-studio/internal/gps"
	"github.com/jossef/dashcam-studio/internal/media"
	"github.com/jossef/dashcam-studio/internal/zipstream"
)

type builtZip struct {
	name   string
	stream *zipstream.Stream
	count  int
}

// buildZip plans a ZIP of the given clips. A *UserError means a bad request
// (unknown clip, nothing to download); other errors come from reading the card.
func buildZip(lib *card.Library, g *gps.Index, ids []string, cams string, withGPX bool) (*builtZip, error) {
	var clips []*card.Clip
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		var c *card.Clip
		if lib != nil {
			c = lib.Get(id)
		}
		if c == nil {
			return nil, &UserError{"Unknown clip"}
		}
		clips = append(clips, c)
	}
	if len(clips) == 0 {
		return nil, &UserError{"Unknown clip"}
	}
	sort.SliceStable(clips, func(i, j int) bool {
		a, b := clips[i], clips[j]
		at, bt := startOf(a), startOf(b)
		if !at.Equal(bt) {
			return at.Before(bt)
		}
		return a.Seq < b.Seq
	})
	var camList []string
	for _, c := range card.Cameras {
		if strings.Contains(strings.ToUpper(cams), c) {
			camList = append(camList, c)
		}
	}
	if len(camList) == 0 {
		camList = card.Cameras
	}
	multi := len(clips) > 1
	var entries []zipstream.Entry
	for _, clip := range clips {
		folder := ""
		if multi {
			folder = clip.Stem + "/"
		}
		for _, cam := range camList {
			path, ok := clip.Files[cam]
			if !ok {
				continue
			}
			e, err := zipstream.FromFile(path, folder+filepath.Base(path))
			if err != nil {
				return nil, err
			}
			entries = append(entries, e)
		}
		if withGPX {
			if track := g.Track(clip.Stem); len(track) > 0 {
				mtime := time.Now()
				if clip.Start != nil {
					mtime = *clip.Start
				}
				entries = append(entries, zipstream.FromBytes(folder+clip.Stem+".gpx", media.GPX(clip.Stem, track, clip.Start), mtime))
			}
		}
	}
	if len(entries) == 0 {
		return nil, &UserError{"Nothing to download"}
	}
	return &builtZip{name: zipName(clips), stream: zipstream.New(entries), count: len(clips)}, nil
}

func startOf(c *card.Clip) time.Time {
	if c.Start == nil {
		return time.Time{}
	}
	return *c.Start
}

func zipName(clips []*card.Clip) string {
	first, last := clips[0], clips[len(clips)-1]
	switch {
	case len(clips) > 1 && first.Start != nil && last.Start != nil:
		f, l := *first.Start, *last.Start
		span := f.Format("2006-01-02") + "_to_" + l.Format("2006-01-02")
		if f.Format("20060102") == l.Format("20060102") {
			span = f.Format("2006-01-02_1504") + "-" + l.Format("1504")
		}
		return fmt.Sprintf("dashcam_%s_%dclips.zip", span, len(clips))
	case first.Start != nil:
		return fmt.Sprintf("dashcam_%s_%06d.zip", first.Start.Format("2006-01-02_15-04-05"), first.Seq)
	default:
		return first.Stem + ".zip"
	}
}
