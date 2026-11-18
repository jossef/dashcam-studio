// Package zipstream writes an uncompressed ("stored") ZIP archive with an
// exact, precomputed size.
//
// Videos are already compressed, so storing them is as small as deflating them.
// It also lets the server start sending at once and announce Content-Length,
// which gives the browser a real progress bar and ETA. The archive switches to
// ZIP64 when it would exceed 4 GiB (e.g. whole-trip downloads).
package zipstream

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"time"
)

const (
	chunk = 1024 * 1024
	limit = 0xFFFFFFFF
)

// Entry is one archive member: either a file on disk or bytes in memory.
type Entry struct {
	Name  string
	Size  int64
	Mtime time.Time
	Path  string
	Data  []byte
}

// FromFile describes a file on disk, stored under name.
func FromFile(path, name string) (Entry, error) {
	st, err := os.Stat(path)
	if err != nil {
		return Entry{}, err
	}
	return Entry{Name: name, Size: st.Size(), Mtime: st.ModTime(), Path: path}, nil
}

// FromBytes describes an in-memory file.
func FromBytes(name string, data []byte, mtime time.Time) Entry {
	return Entry{Name: name, Size: int64(len(data)), Mtime: mtime, Data: data}
}

// Stream is a ZIP archive that can report its size before it is written.
type Stream struct {
	Entries []Entry
	Zip64   bool
}

// New prepares an archive; ZIP64 is chosen automatically.
func New(entries []Entry) *Stream {
	raw := int64(22)
	for _, e := range entries {
		raw += 30 + 46 + 2*int64(len(e.Name)) + e.Size + 16
	}
	return &Stream{Entries: entries, Zip64: raw >= limit-1024 || len(entries) >= 0xFFFF}
}

func (z *Stream) localExtraLen() int64 {
	if z.Zip64 {
		return 20
	}
	return 0
}

func (z *Stream) centralExtraLen() int64 {
	if z.Zip64 {
		return 28
	}
	return 0
}

func (z *Stream) descriptorLen() int64 {
	if z.Zip64 {
		return 24
	}
	return 16
}

// Size is the exact number of bytes WriteTo will write.
func (z *Stream) Size() int64 {
	var total int64
	for _, e := range z.Entries {
		n := int64(len(e.Name))
		total += 30 + n + z.localExtraLen() + e.Size + z.descriptorLen()
		total += 46 + n + z.centralExtraLen()
	}
	if z.Zip64 {
		total += 56 + 20
	}
	return total + 22
}

func dosTime(t time.Time) (uint16, uint16) {
	t = t.Local()
	year := max(1980, t.Year())
	return uint16(t.Hour()<<11 | t.Minute()<<5 | t.Second()/2),
		uint16((year-1980)<<9 | int(t.Month())<<5 | t.Day())
}

type le []byte

func (b le) u16(v uint16) le { return binary.LittleEndian.AppendUint16(b, v) }
func (b le) u32(v uint32) le { return binary.LittleEndian.AppendUint32(b, v) }
func (b le) u64(v uint64) le { return binary.LittleEndian.AppendUint64(b, v) }

type counter struct {
	w io.Writer
	n int64
}

func (c *counter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)
	return n, err
}

type central struct {
	e            Entry
	dtime, ddate uint16
	crc          uint32
	offset       int64
}

// WriteTo streams the archive. Files are opened one at a time and closed as
// soon as they are copied, or when an error stops the stream.
func (z *Stream) WriteTo(w io.Writer) (int64, error) {
	out := &counter{w: w}
	version := uint16(20)
	if z.Zip64 {
		version = 45
	}
	const flags = 0x0008 | 0x0800 // data descriptor + UTF-8 names
	var records []central
	for _, e := range z.Entries {
		dt, dd := dosTime(e.Mtime)
		hdr := le{}.u32(0x04034B50).u16(version).u16(flags).u16(0).u16(dt).u16(dd).u32(0)
		if z.Zip64 {
			hdr = hdr.u32(limit).u32(limit).u16(uint16(len(e.Name))).u16(20)
			hdr = append(hdr, e.Name...)
			hdr = hdr.u16(0x0001).u16(16).u64(0).u64(0)
		} else {
			hdr = hdr.u32(0).u32(0).u16(uint16(len(e.Name))).u16(0)
			hdr = append(hdr, e.Name...)
		}
		offset := out.n
		if _, err := out.Write(hdr); err != nil {
			return out.n, err
		}
		crc, written, err := z.copyData(out, e)
		if err != nil {
			return out.n, err
		}
		if written != e.Size {
			return out.n, fmt.Errorf("%s: expected %d bytes, read %d", e.Name, e.Size, written)
		}
		desc := le{}.u32(0x08074B50).u32(crc)
		if z.Zip64 {
			desc = desc.u64(uint64(written)).u64(uint64(written))
		} else {
			desc = desc.u32(uint32(written)).u32(uint32(written))
		}
		if _, err := out.Write(desc); err != nil {
			return out.n, err
		}
		records = append(records, central{e, dt, dd, crc, offset})
	}

	cdStart := out.n
	for _, r := range records {
		rec := le{}.u32(0x02014B50).u16(version).u16(version).u16(flags).u16(0).u16(r.dtime).u16(r.ddate).u32(r.crc)
		if z.Zip64 {
			rec = rec.u32(limit).u32(limit).u16(uint16(len(r.e.Name))).u16(28).u16(0).u16(0).u16(0).u32(0).u32(limit)
			rec = append(rec, r.e.Name...)
			rec = rec.u16(0x0001).u16(24).u64(uint64(r.e.Size)).u64(uint64(r.e.Size)).u64(uint64(r.offset))
		} else {
			rec = rec.u32(uint32(r.e.Size)).u32(uint32(r.e.Size)).u16(uint16(len(r.e.Name))).u16(0).u16(0).u16(0).u16(0).u32(0).u32(uint32(r.offset))
			rec = append(rec, r.e.Name...)
		}
		if _, err := out.Write(rec); err != nil {
			return out.n, err
		}
	}
	cdSize := out.n - cdStart
	n := uint64(len(records))
	var tail le
	if z.Zip64 {
		end64 := out.n
		tail = tail.u32(0x06064B50).u64(44).u16(version).u16(version).u32(0).u32(0).u64(n).u64(n).u64(uint64(cdSize)).u64(uint64(cdStart))
		tail = tail.u32(0x07064B50).u32(0).u64(uint64(end64)).u32(1)
		m := uint16(min(n, 0xFFFF))
		tail = tail.u32(0x06054B50).u16(0).u16(0).u16(m).u16(m).u32(limit).u32(limit).u16(0)
	} else {
		tail = tail.u32(0x06054B50).u16(0).u16(0).u16(uint16(n)).u16(uint16(n)).u32(uint32(cdSize)).u32(uint32(cdStart)).u16(0)
	}
	_, err := out.Write(tail)
	return out.n, err
}

func (z *Stream) copyData(w io.Writer, e Entry) (uint32, int64, error) {
	if e.Data != nil {
		_, err := w.Write(e.Data)
		return crc32.ChecksumIEEE(e.Data), int64(len(e.Data)), err
	}
	f, err := os.Open(e.Path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	h := crc32.NewIEEE()
	buf := make([]byte, chunk)
	var written int64
	for written < e.Size {
		n, rerr := f.Read(buf[:min(int64(len(buf)), e.Size-written)])
		if n > 0 {
			h.Write(buf[:n])
			if _, err := w.Write(buf[:n]); err != nil {
				return 0, written, err
			}
			written += int64(n)
		}
		if rerr == io.EOF {
			return 0, written, fmt.Errorf("%s shrank while zipping", e.Path)
		}
		if rerr != nil {
			return 0, written, rerr
		}
	}
	return h.Sum32(), written, nil
}
