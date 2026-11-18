package gps

import (
	"fmt"
	"math"
	"path/filepath"
	"testing"

	"github.com/jossef/dashcam-studio/internal/testcard"
)

const stem = "NO20261113-143658-004641"

func makeCard(t *testing.T) string {
	// Device clock is 12h behind the file names (as on the real card).
	base := 1794537418 // == 2026-11-13 14:36:58 "local" minus 12h
	var lines []string
	for i := 0; i < 60; i += 2 {
		lines = append(lines, fmt.Sprintf("%d,A,32.0748%02d,34.825951,9000,%d,100,1,-11,%sF.MP4,0,0,0", base+i, i, 1000+i, stem))
	}
	lines = append(lines, "0,V, 0.000000, 0.000000,0,0,93,2,-10,0,0,0,0", "garbage line")
	// A clip with an unset clock: falls back to first-fix == t0
	lines = append(lines, "1794537500,A,32.1,34.8,0,0,0,0,0,NO20250101-000106-000022F.MP4,0,0,0")
	return testcard.Make(t, "", "Normal", []testcard.Clip{testcard.C(stem, "F", 4)}, lines)
}

func TestLearnsOffsetAndAlignsTrack(t *testing.T) {
	g := Load(makeCard(t))
	if g.Offset != 43200 {
		t.Fatalf("offset %v", g.Offset)
	}
	tr := g.Track(stem)
	if len(tr) != 30 || tr[0][0] != 0 || tr[len(tr)-1][0] != 58 {
		t.Fatalf("track %v", tr)
	}
	if math.Abs(tr[0][3]-36.0) > 0.05 { // 1000 cm/s = 36 km/h
		t.Fatalf("speed %v", tr[0][3])
	}
	if tr[0][4] != 90 {
		t.Fatalf("heading %v", tr[0][4])
	}
}

func TestUnsetClockFallsBack(t *testing.T) {
	if tr := Load(makeCard(t)).Track("NO20250101-000106-000022"); len(tr) != 1 || tr[0][0] != 0 {
		t.Fatal(tr)
	}
}

func TestFindsLogFromCategoryFolder(t *testing.T) {
	if n := Load(filepath.Join(makeCard(t), "Normal")).Len(); n != 2 {
		t.Fatalf("got %d", n)
	}
}

func TestSummary(t *testing.T) {
	s := Load(makeCard(t)).Summary(stem)
	if s == nil || s.Km <= 0 || math.Abs(s.MaxKmh-(1000+58)*0.036) > 0.05 {
		t.Fatalf("%+v", s)
	}
	if Empty().Summary(stem) != nil {
		t.Fatal("summary without data")
	}
}
