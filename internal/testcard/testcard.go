// Package testcard builds fake dashcam cards for tests.
package testcard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// CamDir maps a camera letter to its folder name.
var CamDir = map[rune]string{'F': "Front", 'R': "Rear", 'C': "Cabin"}

// Clip is one fake recording: stem, cameras ("FRC") and file size.
type Clip struct {
	Stem string
	Cams string
	Size int
}

// Make writes clips into root/category/<Camera>/ (root is created when "").
// gpsLines, when not nil, become root/GPSData000001.txt.
func Make(t testing.TB, root, category string, clips []Clip, gpsLines []string) string {
	t.Helper()
	if root == "" {
		root = t.TempDir()
	}
	for _, d := range CamDir {
		if err := os.MkdirAll(filepath.Join(root, category, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, c := range clips {
		for _, cam := range c.Cams {
			data := make([]byte, c.Size)
			for i := range data {
				data[i] = byte((i*7 + int(cam)) % 256)
			}
			path := filepath.Join(root, category, CamDir[cam], c.Stem+string(cam)+".MP4")
			if err := os.WriteFile(path, data, 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	if gpsLines != nil {
		body := "$V02\n" + strings.Join(gpsLines, "\n") + "\n"
		if err := os.WriteFile(filepath.Join(root, "GPSData000001.txt"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// C is shorthand for a Clip.
func C(stem, cams string, size int) Clip { return Clip{Stem: stem, Cams: cams, Size: size} }
