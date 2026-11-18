package server

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/jossef/dashcam-studio/internal/card"
	"github.com/jossef/dashcam-studio/internal/gps"
)

// Read-only checks against a real card, skipped unless DASHCAM_CARD is set:
//
//	DASHCAM_CARD=E:\ go test ./internal/server -run Live -v
//	DASHCAM_LIVE_ZIP=1 additionally streams one real 3-camera ZIP and verifies every CRC.
func liveCard(t *testing.T) string {
	root := os.Getenv("DASHCAM_CARD")
	if root == "" {
		t.Skip("set DASHCAM_CARD to run against a real card")
	}
	if _, err := os.Stat(filepath.Join(root, "Normal", "Front")); err != nil {
		t.Skipf("no dashcam card at %s", root)
	}
	return root
}

func TestLiveEveryFrontFileIsAClip(t *testing.T) {
	root := liveCard(t)
	entries, _ := os.ReadDir(filepath.Join(root, "Normal", "Front"))
	front := 0
	for _, e := range entries {
		if strings.EqualFold(filepath.Ext(e.Name()), ".mp4") {
			front++
		}
	}
	lib := card.Scan(root)
	n := 0
	for _, c := range lib.Sorted() {
		if _, ok := c.Files["F"]; ok && c.Category == "Normal" {
			n++
		}
	}
	if n != front {
		t.Fatalf("%d front files, %d clips with a front camera", front, n)
	}
	g := gps.Load(root)
	t.Logf("%d clips, %d with GPS, offset %vs", lib.Len(), g.Len(), g.Offset)
}

func TestLiveZipCRC(t *testing.T) {
	root := liveCard(t)
	if os.Getenv("DASHCAM_LIVE_ZIP") == "" {
		t.Skip("set DASHCAM_LIVE_ZIP=1 to stream a real ZIP")
	}
	lib := card.Scan(root)
	var clip *card.Clip
	for _, c := range lib.Sorted() {
		if len(c.Files) == 3 {
			clip = c
		}
	}
	if clip == nil {
		t.Skip("no 3-camera clip")
	}
	z, err := buildZip(lib, gps.Load(root), []string{clip.ID}, "FRC", true)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), z.name) // the copy goes to a temp folder, never the card
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	n, err := z.stream.WriteTo(f)
	f.Close()
	if err != nil || n != z.stream.Size() {
		t.Fatalf("wrote %d of %d: %v", n, z.stream.Size(), err)
	}
	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	var names []string
	for _, e := range zr.File {
		rc, _ := e.Open()
		if _, err := io.Copy(io.Discard, rc); err != nil { // archive/zip checks the CRC at EOF
			t.Fatalf("%s: %v", e.Name, err)
		}
		rc.Close()
		names = append(names, e.Name)
	}
	sort.Strings(names)
	t.Logf("%s: %v (%d MB)", z.name, names, n>>20)
}
