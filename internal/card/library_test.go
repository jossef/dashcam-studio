package card

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/jossef/dashcam-studio/internal/testcard"
)

func TestParseNameExamplesFromTheCard(t *testing.T) {
	for _, cam := range []string{"C", "F", "R"} {
		n, ok := ParseName("NO20261109-191423-003968" + cam + ".MP4")
		if !ok || n.Cam != cam || n.Stem != "NO20261109-191423-003968" || n.Seq != 3968 {
			t.Fatalf("%s: %+v", cam, n)
		}
		if got := n.Start.Format("2006-01-02T15:04:05"); got != "2026-11-09T19:14:23" {
			t.Fatalf("start %s", got)
		}
	}
}

func TestParseNameRejectsOtherFiles(t *testing.T) {
	for _, name := range []string{"GPSData000001.txt", "NO20261109-191423-003968X.MP4", "foo.mp4", "NO2026-1.MP4"} {
		if _, ok := ParseName(name); ok {
			t.Errorf("%s should not parse", name)
		}
	}
}

func TestParseNameLowercaseAndEventPrefix(t *testing.T) {
	n, ok := ParseName("ev20261013-142336-000825f.mp4")
	if !ok || n.Cam != "F" || n.Prefix != "ev" {
		t.Fatalf("%+v", n)
	}
}

func makeLibrary(t *testing.T) string {
	root := testcard.Make(t, "", "Normal", []testcard.Clip{
		testcard.C("NO20261109-191423-003968", "FRC", 10),
		testcard.C("NO20261109-191523-003969", "FR", 10), // no cabin
		testcard.C("NO20261109-191623-003970", "F", 10),
	}, nil)
	testcard.Make(t, root, "Event", []testcard.Clip{testcard.C("EV20261013-142336-000825", "FRC", 5)}, nil)
	_ = os.MkdirAll(filepath.Join(root, "Parking", "Front"), 0o755)
	_ = os.WriteFile(filepath.Join(root, "Normal", "Front", "notes.txt"), []byte("ignore me"), 0o644)
	return root
}

func cams(c *Clip) []string {
	var out []string
	for k := range c.Files {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func TestGroupsMatchingCameras(t *testing.T) {
	lib := Scan(makeLibrary(t))
	if lib.Len() != 4 {
		t.Fatalf("got %d clips", lib.Len())
	}
	if got := cams(lib.Get("Normal/NO20261109-191423-003968")); !reflect.DeepEqual(got, []string{"C", "F", "R"}) {
		t.Fatal(got)
	}
	if got := cams(lib.Get("Normal/NO20261109-191523-003969")); !reflect.DeepEqual(got, []string{"F", "R"}) {
		t.Fatal(got)
	}
	if c := lib.Get("Event/EV20261013-142336-000825"); c == nil || c.Category != "Event" {
		t.Fatal(c)
	}
}

func TestCategoriesOrdered(t *testing.T) {
	cats := Scan(makeLibrary(t)).Categories()
	if len(cats) != 2 || cats[0].Name != "Normal" || cats[1].Name != "Event" {
		t.Fatal(cats)
	}
}

func TestRootCanBeACategoryFolder(t *testing.T) {
	lib := Scan(filepath.Join(makeLibrary(t), "Normal"))
	if lib.Len() != 3 {
		t.Fatalf("got %d", lib.Len())
	}
	for _, c := range lib.Sorted() {
		if c.Category != "Normal" {
			t.Fatal(c.Category)
		}
	}
}

func TestSortedChronologically(t *testing.T) {
	clips := Scan(makeLibrary(t)).Sorted()
	for i := 1; i < len(clips); i++ {
		if clips[i].Start.Before(*clips[i-1].Start) {
			t.Fatal("not sorted")
		}
	}
}

func TestScanIsReadOnly(t *testing.T) {
	root := makeLibrary(t)
	snap := func() map[string]time.Time {
		out := map[string]time.Time{}
		_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, _ error) error {
			// files only: NTFS updates folder times lazily after the fixture is written
			if info, err := d.Info(); err == nil && !d.IsDir() {
				out[p] = info.ModTime()
			}
			return nil
		})
		return out
	}
	before := snap()
	Scan(root)
	if !reflect.DeepEqual(before, snap()) {
		t.Fatal("scan modified the card")
	}
}

func TestLooksLikeDashcamRoot(t *testing.T) {
	root := makeLibrary(t)
	if !LooksLikeDashcamRoot(root) || !LooksLikeDashcamRoot(filepath.Join(root, "Normal")) {
		t.Fatal("card not recognized")
	}
	if LooksLikeDashcamRoot(t.TempDir()) {
		t.Fatal("empty folder recognized")
	}
}
