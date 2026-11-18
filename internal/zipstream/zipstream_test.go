package zipstream

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jossef/dashcam-studio/internal/testcard"
)

func files(t *testing.T) []string {
	root := testcard.Make(t, "", "Normal", []testcard.Clip{testcard.C("NO20261109-191423-003968", "FRC", 300_000)}, nil)
	var out []string
	for _, d := range []string{"Front", "Rear", "Cabin"} {
		out = append(out, filepath.Join(root, "Normal", d, "NO20261109-191423-003968"+d[:1]+".MP4"))
	}
	return out
}

// check writes the archive, verifies the announced size and every member's
// content (archive/zip verifies the CRCs while reading).
func check(t *testing.T, z *Stream) *zip.Reader {
	t.Helper()
	var buf bytes.Buffer
	n, err := z.WriteTo(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if n != z.Size() || int64(buf.Len()) != z.Size() {
		t.Fatalf("Content-Length must be exact: wrote %d, size %d", n, z.Size())
	}
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	for i, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		got, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("%s: %v", f.Name, err)
		}
		e := z.Entries[i]
		want := e.Data
		if want == nil {
			want, _ = os.ReadFile(e.Path)
		}
		if f.Name != e.Name || !bytes.Equal(got, want) {
			t.Fatalf("%s: content mismatch", f.Name)
		}
	}
	return zr
}

func TestThreeCamerasPlusGPX(t *testing.T) {
	var entries []Entry
	for _, p := range files(t) {
		e, err := FromFile(p, filepath.Base(p))
		if err != nil {
			t.Fatal(err)
		}
		entries = append(entries, e)
	}
	entries = append(entries, FromBytes("track.gpx", []byte("<gpx/>"), time.Now()))
	z := New(entries)
	if z.Zip64 {
		t.Fatal("small archive should not be zip64")
	}
	if zr := check(t, z); len(zr.File) != 4 {
		t.Fatal(len(zr.File))
	}
}

func TestZip64Layout(t *testing.T) {
	var entries []Entry
	for _, p := range files(t) {
		e, _ := FromFile(p, "folder/"+filepath.Base(p))
		entries = append(entries, e)
	}
	z := New(entries)
	z.Zip64 = true // force the >4 GiB code path on small data
	check(t, z)
}

func TestUnicodeName(t *testing.T) {
	check(t, New([]Entry{FromBytes("נסיעה.gpx", bytes.Repeat([]byte("x"), 10), time.Now())}))
}
