package media

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jossef/dashcam-studio/internal/proc"
)

func TestMimeCodec(t *testing.T) {
	cases := []struct {
		info Info
		want string
	}{
		{Info{Codec: "hevc", Tag: "hvc1", Profile: "Main"}, "hvc1.1.6.L150.B0"},
		{Info{Codec: "hevc", Tag: "hev1", Profile: "Main 10"}, "hev1.2.6.L150.B0"},
		{Info{Codec: "hevc", Tag: "[0][0][0][0]"}, "hvc1.1.6.L150.B0"},
		{Info{Codec: "h264"}, "avc1.640033"},
		{Info{Codec: "mpeg4"}, ""},
	}
	for _, c := range cases {
		if got := MimeCodec(c.info); got != c.want {
			t.Errorf("%+v: got %q want %q", c.info, got, c.want)
		}
	}
}

func runFFmpeg(ctx context.Context, args ...string) error {
	out, err := proc.Command(ctx, FFmpeg, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, out)
	}
	return nil
}

func TestProbeRejectsNonMP4(t *testing.T) {
	p := filepath.Join(t.TempDir(), "x.MP4")
	_ = os.WriteFile(p, []byte("definitely not a video"), 0o644)
	if _, err := probeMP4(p); err == nil {
		t.Fatal("expected an error")
	}
}

// TestProbeAgainstFFprobe encodes a short clip with ffmpeg and checks that the
// pure-Go probe agrees with ffprobe. Skipped when ffmpeg is not installed.
func TestProbeAgainstFFprobe(t *testing.T) {
	if FFmpeg == "" || FFprobe == "" {
		t.Skip("ffmpeg not installed")
	}
	for _, codec := range []string{"libx264", "libx265"} {
		t.Run(codec, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "clip.mp4")
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			args := []string{"-v", "error", "-f", "lavfi", "-i", "testsrc=size=320x240:rate=25:duration=2",
				"-f", "lavfi", "-i", "sine=duration=2", "-shortest", "-c:v", codec, "-c:a", "aac", "-movflags", "+faststart"}
			if codec == "libx265" {
				args = append(args, "-tag:v", "hvc1", "-x265-params", "log-level=none")
			}
			if err := runFFmpeg(ctx, append(args, out)...); err != nil {
				t.Skipf("encoder unavailable: %v", err)
			}
			got, err := probeMP4(out)
			if err != nil {
				t.Fatal(err)
			}
			want, ok := ffprobe(out)
			if !ok {
				t.Fatal("ffprobe failed")
			}
			if got.Codec != want.Codec || got.Tag != want.Tag || got.Width != want.Width || got.Height != want.Height ||
				got.FPS != want.FPS || got.Audio != want.Audio || got.Profile != want.Profile {
				t.Fatalf("pure Go %+v\nffprobe  %+v", got, want)
			}
			if d := got.Duration - want.Duration; d > 0.1 || d < -0.1 {
				t.Fatalf("duration %v vs %v", got.Duration, want.Duration)
			}
			if MimeCodec(got) == "" {
				t.Fatal("no mime")
			}
		})
	}
}

// TestLiveCard probes real files when DASHCAM_CARD points at a card (read-only).
func TestLiveCard(t *testing.T) {
	root := os.Getenv("DASHCAM_CARD")
	if root == "" {
		t.Skip("set DASHCAM_CARD to probe a real card")
	}
	matches, _ := filepath.Glob(filepath.Join(root, "Normal", "Front", "*.MP4"))
	if len(matches) == 0 {
		t.Skip("no clips")
	}
	info := Probe(matches[len(matches)-1])
	t.Logf("%s: %+v mime=%s", filepath.Base(matches[len(matches)-1]), info, MimeCodec(info))
	if info.Codec == "" || info.Width == 0 || info.Duration == 0 {
		t.Fatalf("%+v", info)
	}
	if alt, ok := ffprobe(matches[len(matches)-1]); ok && (alt.Codec != info.Codec || alt.Width != info.Width || alt.Tag != info.Tag) {
		t.Fatalf("ffprobe disagrees: %+v", alt)
	}
}
