package media

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jossef/dashcam-studio/internal/paths"
	"github.com/jossef/dashcam-studio/internal/proc"
)

// FFmpeg and FFprobe are the tool paths ("" when not installed). A copy placed
// next to the app's executable wins over PATH.
var (
	FFmpeg  = proc.Find("ffmpeg")
	FFprobe = proc.Find("ffprobe")
)

func ffprobe(path string) (Info, bool) {
	if FFprobe == "" {
		return Info{}, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	out, err := proc.Command(ctx, FFprobe, "-v", "error", "-print_format", "json", "-show_format", "-show_streams", path).Output()
	if err != nil {
		return Info{}, false
	}
	var data struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
		Streams []struct {
			Type      string `json:"codec_type"`
			Codec     string `json:"codec_name"`
			Tag       string `json:"codec_tag_string"`
			Profile   string `json:"profile"`
			Width     int    `json:"width"`
			Height    int    `json:"height"`
			FrameRate string `json:"r_frame_rate"`
		} `json:"streams"`
	}
	if json.Unmarshal(out, &data) != nil {
		return Info{}, false
	}
	var info Info
	info.Duration, _ = strconv.ParseFloat(data.Format.Duration, 64)
	for _, s := range data.Streams {
		switch {
		case s.Type == "video" && info.Codec == "":
			info.Codec, info.Tag, info.Profile, info.Width, info.Height = s.Codec, s.Tag, s.Profile, s.Width, s.Height
			if num, den, ok := strings.Cut(s.FrameRate, "/"); ok {
				n, _ := strconv.ParseFloat(num, 64)
				d, _ := strconv.ParseFloat(den, 64)
				if d > 0 {
					info.FPS = float64(int(n/d*1000+0.5)) / 1000
				}
			}
		case s.Type == "audio":
			info.Audio = true
		}
	}
	return info, true
}

// ------------------------------------------------------------------ thumbnails

var (
	thumbSlots = make(chan struct{}, max(2, min(4, runtime.NumCPU()/2)))
	thumbMu    sync.Mutex
	thumbCalls = map[string]*thumbCall{}
)

type thumbCall struct {
	done chan struct{}
	path string
}

func thumbPath(src string) (string, error) {
	st, err := os.Stat(src)
	if err != nil {
		return "", err
	}
	abs, _ := filepath.Abs(src)
	sum := sha1.Sum([]byte(fmt.Sprintf("%s|%d|%d|v2", abs, st.Size(), st.ModTime().Unix())))
	return filepath.Join(paths.CacheDir("thumbs"), hex.EncodeToString(sum[:])[:20]+".jpg"), nil
}

// Thumbnail returns the path of a cached JPEG thumbnail, generating it on the
// first request. Concurrent requests for the same file share one ffmpeg run.
// ctx cancels the ffmpeg run (e.g. when the card is being disconnected).
func Thumbnail(ctx context.Context, src string) string {
	if FFmpeg == "" {
		return ""
	}
	dst, err := thumbPath(src)
	if err != nil {
		return ""
	}
	if _, err := os.Stat(dst); err == nil {
		return dst
	}
	thumbMu.Lock()
	call := thumbCalls[dst]
	if call == nil {
		call = &thumbCall{done: make(chan struct{})}
		thumbCalls[dst] = call
		go func() {
			call.path = makeThumb(ctx, src, dst, 3.0)
			thumbMu.Lock()
			delete(thumbCalls, dst)
			thumbMu.Unlock()
			close(call.done)
		}()
	}
	thumbMu.Unlock()
	select {
	case <-call.done:
		return call.path
	case <-time.After(40 * time.Second):
		return ""
	case <-ctx.Done():
		return ""
	}
}

func makeThumb(ctx context.Context, src, dst string, at float64) string {
	select {
	case thumbSlots <- struct{}{}:
		defer func() { <-thumbSlots }()
	case <-ctx.Done():
		return ""
	}
	tmp := fmt.Sprintf("%s.%d.tmp.jpg", dst, time.Now().UnixNano())
	defer os.Remove(tmp)
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := proc.Command(ctx, FFmpeg, "-v", "error", "-y", "-ss", strconv.FormatFloat(at, 'f', -1, 64), "-i", src,
		"-frames:v", "1", "-vf", "scale=400:-2", "-q:v", "5", tmp)
	if cmd.Run() != nil {
		return ""
	}
	if os.Rename(tmp, dst) != nil {
		return ""
	}
	return dst
}

// ------------------------------------------------------------------- transcode

// TranscodeArgs is the ffmpeg command line that streams a browser-safe,
// fragmented H.264 MP4 to stdout.
func TranscodeArgs(src string, start float64, height int) []string {
	return []string{
		"-v", "error", "-nostdin",
		"-ss", fmt.Sprintf("%.3f", max(0, start)), "-i", src,
		"-map", "0:v:0", "-map", "0:a:0?",
		"-vf", fmt.Sprintf("scale=-2:'min(%d,ih)'", height),
		"-c:v", "libx264", "-preset", "veryfast", "-tune", "zerolatency", "-crf", "23",
		"-pix_fmt", "yuv420p", "-g", "48",
		"-c:a", "aac", "-b:a", "128k",
		"-movflags", "frag_keyframe+empty_moov+default_base_moof",
		"-f", "mp4", "pipe:1",
	}
}

// Transcode starts ffmpeg; read the video from the returned stream and Close
// it to stop ffmpeg. Cancelling ctx kills ffmpeg too.
func Transcode(ctx context.Context, src string, start float64) (io.ReadCloser, error) {
	cmd := proc.Command(ctx, FFmpeg, TranscodeArgs(src, start, 720)...)
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &transcodeStream{ReadCloser: out, cmd: cmd}, nil
}

type transcodeStream struct {
	io.ReadCloser
	cmd *exec.Cmd
}

func (t *transcodeStream) Close() error {
	_ = t.cmd.Process.Kill()
	_ = t.ReadCloser.Close()
	return t.cmd.Wait()
}
