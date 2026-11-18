package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/tc-hib/winres"
	"github.com/tc-hib/winres/version"
)

// The app icon is drawn in code (no binary assets in the repo): the UI's
// favicon - an amber lens pill - on a dark rounded tile.
var (
	tileColor = color.NRGBA{0x14, 0x17, 0x1c, 0xff}
	amber     = color.NRGBA{0xff, 0xb2, 0x38, 0xff}
	lensColor = color.NRGBA{0x0b, 0x0d, 0x10, 0xff}
	lensGlint = color.NRGBA{0xff, 0xd8, 0x8a, 0xff}
)

// drawIcon renders the icon at size×size with 4×4 supersampling.
func drawIcon(size int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	s := float64(size)
	inRoundRect := func(x, y, x0, y0, x1, y1, r float64) bool {
		cx, cy := math.Max(x0+r, math.Min(x, x1-r)), math.Max(y0+r, math.Min(y, y1-r))
		return x >= x0 && x <= x1 && y >= y0 && y <= y1 && (x-cx)*(x-cx)+(y-cy)*(y-cy) <= r*r
	}
	const n = 4
	for py := range size {
		for px := range size {
			var r, g, b, a float64
			for sy := range n {
				for sx := range n {
					x := (float64(px) + (float64(sx)+0.5)/n) / s
					y := (float64(py) + (float64(sy)+0.5)/n) / s
					var c color.NRGBA
					switch {
					case (x-0.5)*(x-0.5)+(y-0.5)*(y-0.5) <= 0.115*0.115:
						c = lensColor
						if (x-0.47)*(x-0.47)+(y-0.47)*(y-0.47) <= 0.035*0.035 {
							c = lensGlint
						}
					case inRoundRect(x, y, 0.12, 0.29, 0.88, 0.71, 0.21):
						c = amber
					case inRoundRect(x, y, 0.02, 0.02, 0.98, 0.98, 0.22):
						c = tileColor
					default:
						continue
					}
					r, g, b, a = r+float64(c.R), g+float64(c.G), b+float64(c.B), a+1
				}
			}
			if a > 0 {
				img.SetNRGBA(px, py, color.NRGBA{uint8(r / a), uint8(g / a), uint8(b / a), uint8(a / (n * n) * 255)})
			}
		}
	}
	return img
}

// writeWindowsResources writes cmd/dashcam-studio/rsrc_windows_<arch>.syso:
// the icon, a manifest (per-monitor DPI awareness, modern controls) and
// version info. `go build` links it automatically for that GOARCH.
func writeWindowsResources(c config, goarch string) error {
	var images []image.Image
	for _, size := range []int{256, 64, 48, 32, 24, 16} {
		images = append(images, drawIcon(size))
	}
	icon, err := winres.NewIconFromImages(images)
	if err != nil {
		return err
	}
	rs := winres.ResourceSet{}
	if err := rs.SetIcon(winres.ID(1), icon); err != nil {
		return err
	}
	rs.SetManifest(winres.AppManifest{
		Description:         "Dashcam Studio",
		Compatibility:       winres.Win10AndAbove,
		DPIAwareness:        winres.DPIPerMonitorV2,
		UseCommonControlsV6: true,
		LongPathAware:       true,
	})
	vi := version.Info{}
	numeric := numericVersion(c.version)
	vi.SetFileVersion(numeric)
	vi.SetProductVersion(numeric)
	for k, v := range map[string]string{
		version.ProductName: "Dashcam Studio", version.FileDescription: "Dashcam Studio",
		version.OriginalFilename: appName + ".exe", version.InternalName: appName,
		version.ProductVersion: c.version, version.FileVersion: c.version,
	} {
		_ = vi.Set(version.LangDefault, k, v)
	}
	rs.SetVersionInfo(vi)
	f, err := os.Create(filepath.Join(c.root, "cmd", appName, fmt.Sprintf("rsrc_windows_%s.syso", goarch)))
	if err != nil {
		return err
	}
	defer f.Close()
	return rs.WriteObject(f, winres.Arch(goarch))
}

// numericVersion turns "2.1.0-beta" into "2.1.0.0".
func numericVersion(v string) string {
	v = strings.TrimPrefix(v, "v")
	if i := strings.IndexAny(v, "-+ "); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	for len(parts) < 4 {
		parts = append(parts, "0")
	}
	return strings.Join(parts[:4], ".")
}

// icns builds a macOS .icns file from PNG renditions.
func icns() ([]byte, error) {
	var body bytes.Buffer
	for _, e := range []struct {
		typ  string
		size int
	}{{"ic07", 128}, {"ic08", 256}, {"ic09", 512}, {"ic10", 1024}} {
		var p bytes.Buffer
		if err := png.Encode(&p, drawIcon(e.size)); err != nil {
			return nil, err
		}
		body.WriteString(e.typ)
		_ = binary.Write(&body, binary.BigEndian, uint32(8+p.Len()))
		body.Write(p.Bytes())
	}
	var out bytes.Buffer
	out.WriteString("icns")
	_ = binary.Write(&out, binary.BigEndian, uint32(8+body.Len()))
	out.Write(body.Bytes())
	return out.Bytes(), nil
}
