// Package media reads video metadata, makes thumbnails, transcodes to H.264
// and writes GPX tracks.
//
// Probing is pure Go (MP4/MOV box parsing), so codec detection works without
// any external tool. ffmpeg is optional: without it the player still plays the
// files directly; you only lose thumbnails and the transcoding fallback.
package media

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"sync"
	"time"
)

// Info describes a video file. Fields are omitted from JSON when unknown.
type Info struct {
	Duration float64 `json:"duration,omitempty"`
	Codec    string  `json:"codec,omitempty"`
	Tag      string  `json:"tag,omitempty"`
	Profile  string  `json:"profile,omitempty"`
	Width    int     `json:"width,omitempty"`
	Height   int     `json:"height,omitempty"`
	FPS      float64 `json:"fps,omitempty"`
	Audio    bool    `json:"audio,omitempty"`
}

type probeKey struct {
	path  string
	size  int64
	mtime time.Time
}

var (
	probeMu    sync.Mutex
	probeCache = map[probeKey]Info{}
)

// Probe returns what can be learned about a video (best effort, cached).
func Probe(path string) Info {
	st, err := os.Stat(path)
	if err != nil {
		return Info{}
	}
	key := probeKey{path, st.Size(), st.ModTime()}
	probeMu.Lock()
	info, ok := probeCache[key]
	probeMu.Unlock()
	if ok {
		return info
	}
	info, err = probeMP4(path)
	if err != nil || info.Codec == "" {
		if alt, ok := ffprobe(path); ok {
			info = alt
		}
	}
	probeMu.Lock()
	probeCache[key] = info
	probeMu.Unlock()
	return info
}

// MimeCodec is a codecs= string the browser can test with canPlayType / isTypeSupported.
func MimeCodec(info Info) string {
	switch info.Codec {
	case "hevc":
		tag := info.Tag
		if tag != "hvc1" && tag != "hev1" {
			tag = "hvc1"
		}
		prof := 1
		if len(info.Profile) >= 7 && (info.Profile[:7] == "Main 10" || info.Profile[:7] == "main 10") {
			prof = 2
		}
		return fmt.Sprintf("%s.%d.6.L150.B0", tag, prof)
	case "h264":
		return "avc1.640033"
	case "av1":
		return "av01.0.08M.08"
	}
	return ""
}

// ----------------------------------------------------------------- MP4 boxes

const maxMoov = 64 << 20

var errNoMoov = errors.New("no moov box")

type box struct {
	typ  string
	body []byte
}

// children splits a box body into its child boxes.
func children(b []byte) []box {
	var out []box
	for len(b) >= 8 {
		size := uint64(binary.BigEndian.Uint32(b))
		typ := string(b[4:8])
		hdr := uint64(8)
		if size == 1 {
			if len(b) < 16 {
				break
			}
			size, hdr = binary.BigEndian.Uint64(b[8:]), 16
		} else if size == 0 {
			size = uint64(len(b))
		}
		if size < hdr || size > uint64(len(b)) {
			break
		}
		out = append(out, box{typ, b[hdr:size]})
		b = b[size:]
	}
	return out
}

func find(b []byte, path ...string) []byte {
	for _, name := range path {
		var next []byte
		for _, c := range children(b) {
			if c.typ == name {
				next = c.body
				break
			}
		}
		if next == nil {
			return nil
		}
		b = next
	}
	return b
}

// readMoov finds the top-level moov box (often at the end of dashcam files).
func readMoov(f *os.File, size int64) ([]byte, error) {
	var off int64
	hdr := make([]byte, 16)
	for off+8 <= size {
		if _, err := f.ReadAt(hdr[:8], off); err != nil {
			return nil, err
		}
		boxSize := int64(binary.BigEndian.Uint32(hdr))
		typ := string(hdr[4:8])
		hlen := int64(8)
		if boxSize == 1 {
			if _, err := f.ReadAt(hdr[8:16], off+8); err != nil {
				return nil, err
			}
			boxSize, hlen = int64(binary.BigEndian.Uint64(hdr[8:])), 16
		} else if boxSize == 0 {
			boxSize = size - off
		}
		if boxSize < hlen || off+boxSize > size {
			return nil, errNoMoov
		}
		if typ == "moov" {
			if boxSize-hlen > maxMoov {
				return nil, errors.New("moov too large")
			}
			body := make([]byte, boxSize-hlen)
			if _, err := f.ReadAt(body, off+hlen); err != nil && err != io.EOF {
				return nil, err
			}
			return body, nil
		}
		off += boxSize
	}
	return nil, errNoMoov
}

// timing reads (timescale, duration) from an mvhd or mdhd body.
func timing(b []byte) (uint32, uint64) {
	if len(b) < 4 {
		return 0, 0
	}
	if b[0] == 1 && len(b) >= 32 {
		return binary.BigEndian.Uint32(b[20:]), binary.BigEndian.Uint64(b[24:])
	}
	if len(b) >= 20 {
		return binary.BigEndian.Uint32(b[12:]), uint64(binary.BigEndian.Uint32(b[16:]))
	}
	return 0, 0
}

var hevcProfiles = map[byte]string{1: "Main", 2: "Main 10", 3: "Main Still Picture", 4: "Rext"}
var avcProfiles = map[byte]string{66: "Baseline", 77: "Main", 88: "Extended", 100: "High", 110: "High 10", 122: "High 4:2:2", 244: "High 4:4:4 Predictive"}

func probeMP4(path string) (Info, error) {
	var info Info
	f, err := os.Open(path)
	if err != nil {
		return info, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return info, err
	}
	moov, err := readMoov(f, st.Size())
	if err != nil {
		return info, err
	}
	if scale, dur := timing(find(moov, "mvhd")); scale > 0 {
		info.Duration = math.Round(float64(dur)/float64(scale)*1000) / 1000
	}
	for _, trak := range children(moov) {
		if trak.typ != "trak" {
			continue
		}
		hdlr := find(trak.body, "mdia", "hdlr")
		if len(hdlr) < 12 {
			continue
		}
		switch string(hdlr[8:12]) {
		case "soun":
			info.Audio = true
		case "vide":
			if info.Codec != "" {
				continue
			}
			parseVideoTrack(trak.body, &info)
		}
	}
	return info, nil
}

func parseVideoTrack(trak []byte, info *Info) {
	stbl := find(trak, "mdia", "minf", "stbl")
	stsd := find(stbl, "stsd")
	if len(stsd) < 8+86 {
		return
	}
	entry := stsd[8:] // skip version/flags + entry count
	size := int(binary.BigEndian.Uint32(entry))
	if size < 86 || size > len(entry) {
		return
	}
	tag := string(entry[4:8])
	info.Tag = tag
	info.Width = int(binary.BigEndian.Uint16(entry[32:]))
	info.Height = int(binary.BigEndian.Uint16(entry[34:]))
	ext := entry[86:size]
	switch tag {
	case "hvc1", "hev1":
		info.Codec = "hevc"
		if c := find(ext, "hvcC"); len(c) >= 2 {
			info.Profile = hevcProfiles[c[1]&0x1F]
		}
	case "avc1", "avc3":
		info.Codec = "h264"
		if c := find(ext, "avcC"); len(c) >= 2 {
			info.Profile = avcProfiles[c[1]]
		}
	case "av01":
		info.Codec = "av1"
	case "mp4v":
		info.Codec = "mpeg4"
	default:
		info.Codec = tag
	}
	// Average frame rate from the sample table: samples / track duration.
	scale, _ := timing(find(trak, "mdia", "mdhd"))
	if stts := find(stbl, "stts"); len(stts) >= 8 && scale > 0 {
		n := int(binary.BigEndian.Uint32(stts[4:]))
		var samples, delta uint64
		for i := 0; i < n && 8+i*8+8 <= len(stts); i++ {
			c := uint64(binary.BigEndian.Uint32(stts[8+i*8:]))
			samples += c
			delta += c * uint64(binary.BigEndian.Uint32(stts[12+i*8:]))
		}
		if delta > 0 {
			info.FPS = math.Round(float64(samples)*float64(scale)/float64(delta)*1000) / 1000
		}
	}
}
