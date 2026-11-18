package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"debug/macho"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// makeUniversal merges single-architecture Mach-O binaries into one "fat"
// binary (what `lipo -create` does), so macOS needs a single download.
func makeUniversal(out string, slices []string) error {
	const align = 14 // 2^14 = 16 KiB, the arm64 page size
	type arch struct {
		cpu, sub uint32
		data     []byte
	}
	var archs []arch
	for _, path := range slices {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		f, err := macho.Open(path)
		if err != nil {
			return err
		}
		archs = append(archs, arch{uint32(f.Cpu), f.SubCpu, data})
		f.Close()
	}
	var buf []byte
	buf = binary.BigEndian.AppendUint32(buf, macho.MagicFat)
	buf = binary.BigEndian.AppendUint32(buf, uint32(len(archs)))
	offset := uint32(1 << align)
	var offsets []uint32
	for _, a := range archs {
		offsets = append(offsets, offset)
		buf = binary.BigEndian.AppendUint32(buf, a.cpu)
		buf = binary.BigEndian.AppendUint32(buf, a.sub)
		buf = binary.BigEndian.AppendUint32(buf, offset)
		buf = binary.BigEndian.AppendUint32(buf, uint32(len(a.data)))
		buf = binary.BigEndian.AppendUint32(buf, align)
		offset += uint32(len(a.data))
		offset = (offset + (1<<align - 1)) &^ (1<<align - 1)
	}
	for i, a := range archs {
		buf = append(buf, make([]byte, int(offsets[i])-len(buf))...)
		buf = append(buf, a.data...)
	}
	if err := os.WriteFile(out, buf, 0o755); err != nil {
		return err
	}
	ff, err := macho.OpenFat(out) // sanity check
	if err != nil {
		return fmt.Errorf("universal binary is invalid: %w", err)
	}
	return ff.Close()
}

const infoPlist = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleName</key><string>Dashcam Studio</string>
  <key>CFBundleDisplayName</key><string>Dashcam Studio</string>
  <key>CFBundleIdentifier</key><string>app.dashcam-studio</string>
  <key>CFBundleExecutable</key><string>dashcam-studio</string>
  <key>CFBundleIconFile</key><string>icon.icns</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>CFBundleShortVersionString</key><string>%[1]s</string>
  <key>CFBundleVersion</key><string>%[1]s</string>
  <key>LSMinimumSystemVersion</key><string>11.0</string>
  <key>NSHighResolutionCapable</key><true/>
  <!-- The window is a Chromium app window; this process only serves it. -->
  <key>LSUIElement</key><true/>
</dict>
</plist>
`

// packageMac zips "Dashcam Studio.app" (double-click to run) into one file.
func packageMac(c config, bin, arch string) (string, error) {
	out := filepath.Join(c.out, fmt.Sprintf("%s-%s-macos-%s.zip", appName, c.version, arch))
	icon, err := icns()
	if err != nil {
		return "", err
	}
	exe, err := os.ReadFile(bin)
	if err != nil {
		return "", err
	}
	const app = "Dashcam Studio.app/Contents/"
	files := []archiveFile{
		{app + "Info.plist", []byte(fmt.Sprintf(infoPlist, c.version)), 0o644},
		{app + "MacOS/" + appName, exe, 0o755},
		{app + "Resources/icon.icns", icon, 0o644},
	}
	f, err := os.Create(out)
	if err != nil {
		return "", err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	for _, af := range files {
		h := &zip.FileHeader{Name: af.name, Method: zip.Deflate, Modified: time.Now()}
		h.SetMode(af.mode) // keeps the executable bit
		w, err := zw.CreateHeader(h)
		if err != nil {
			return "", err
		}
		if _, err := w.Write(af.data); err != nil {
			return "", err
		}
	}
	return out, zw.Close()
}

type archiveFile struct {
	name string
	data []byte
	mode os.FileMode
}

// packageTarGz wraps the Linux binary so it keeps its executable bit.
func packageTarGz(bin, out string) (string, error) {
	data, err := os.ReadFile(bin)
	if err != nil {
		return "", err
	}
	f, err := os.Create(out)
	if err != nil {
		return "", err
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: appName, Mode: 0o755, Size: int64(len(data)), ModTime: time.Now(), Typeflag: tar.TypeReg}); err != nil {
		return "", err
	}
	if _, err := tw.Write(data); err != nil {
		return "", err
	}
	if err := tw.Close(); err != nil {
		return "", err
	}
	return out, gz.Close()
}

func copyArtifact(src, dst string) (string, error) {
	in, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return "", err
	}
	return dst, out.Close()
}

func writeChecksums(dir string, files []string) error {
	sort.Strings(files)
	var lines []string
	for _, p := range files {
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		lines = append(lines, hex.EncodeToString(sum[:])+"  "+filepath.Base(p))
	}
	return os.WriteFile(filepath.Join(dir, "SHA256SUMS.txt"), []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}
