package media

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"time"
)

// GPX renders a track ([t, lat, lon, kmh, heading] points) as a GPX 1.1 file.
// base is the clip start; when nil, points carry no timestamps.
func GPX(name string, track [][5]float64, base *time.Time) []byte {
	var b bytes.Buffer
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<gpx version="1.1" creator="Dashcam Studio" xmlns="http://www.topografix.com/GPX/1/1">` + "\n")
	b.WriteString("  <trk><name>")
	_ = xml.EscapeText(&b, []byte(name))
	b.WriteString("</name><trkseg>\n")
	for _, p := range track {
		t, lat, lon, kmh, hdg := p[0], p[1], p[2], p[3], p[4]
		timeTag := ""
		if base != nil {
			at := base.Add(time.Duration(t * float64(time.Second))).UTC()
			timeTag = "<time>" + at.Format("2006-01-02T15:04:05") + "</time>"
		}
		fmt.Fprintf(&b, `    <trkpt lat="%.6f" lon="%.6f">%s<extensions><speed>%.2f</speed><course>%.1f</course></extensions></trkpt>`+"\n",
			lat, lon, timeTag, kmh/3.6, hdg)
	}
	b.WriteString("  </trkseg></trk>\n</gpx>\n")
	return b.Bytes()
}
