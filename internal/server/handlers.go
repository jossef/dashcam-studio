package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jossef/dashcam-studio/internal/card"
	"github.com/jossef/dashcam-studio/internal/gps"
	"github.com/jossef/dashcam-studio/internal/media"
	"github.com/jossef/dashcam-studio/internal/volume"
)

func clipPayload(c *card.Clip, g *gps.Index) map[string]any {
	d := c.JSON()
	if s := g.Summary(c.Stem); s != nil {
		d["gps"] = s
	}
	return d
}

func (s *Server) library(w http.ResponseWriter, r *http.Request) {
	lib, g := s.State.snapshot()
	if lib == nil {
		writeJSON(w, 200, map[string]any{"root": nil, "clips": []any{}, "categories": []any{}})
		return
	}
	sorted := lib.Sorted()
	clips := make([]map[string]any, 0, len(sorted))
	for _, c := range sorted {
		clips = append(clips, clipPayload(c, g))
	}
	writeJSON(w, 200, map[string]any{"root": lib.Root, "categories": lib.Categories(), "clips": clips, "hasGps": g.Len() > 0})
}

// clipAndCam resolves ?id=&cam= or writes a 404.
func (s *Server) clipAndCam(w http.ResponseWriter, r *http.Request) (*card.Clip, string, bool) {
	lib, _ := s.State.snapshot()
	q := r.URL.Query()
	var clip *card.Clip
	if lib != nil {
		clip = lib.Get(q.Get("id"))
	}
	cam := strings.ToUpper(q.Get("cam"))
	if cam == "" {
		cam = "F"
	}
	if clip == nil {
		writeError(w, 404, "Unknown clip")
		return nil, "", false
	}
	if _, ok := clip.Files[cam]; !ok {
		name := card.CameraNames[cam]
		if name == "" {
			name = cam
		}
		writeError(w, 404, fmt.Sprintf("Clip has no %s video", name))
		return nil, "", false
	}
	return clip, cam, true
}

func (s *Server) clip(w http.ResponseWriter, r *http.Request, _ *Lease) error {
	lib, g := s.State.snapshot()
	var clip *card.Clip
	if lib != nil {
		clip = lib.Get(r.URL.Query().Get("id"))
	}
	if clip == nil {
		writeError(w, 404, "Unknown clip")
		return nil
	}
	cams := map[string]any{}
	for _, cam := range card.Cameras {
		path, ok := clip.Files[cam]
		if !ok {
			continue
		}
		info := media.Probe(path)
		entry := map[string]any{}
		raw, _ := json.Marshal(info)
		_ = json.Unmarshal(raw, &entry)
		entry["name"] = card.CameraNames[cam]
		entry["file"] = filepath.Base(path)
		entry["size"] = clip.Sizes[cam]
		if mime := media.MimeCodec(info); mime != "" {
			entry["mime"] = mime
		} else {
			entry["mime"] = nil
		}
		cams[cam] = entry
	}
	writeJSON(w, 200, map[string]any{"clip": clipPayload(clip, g), "cams": cams, "track": g.Track(clip.Stem)})
	return nil
}

func (s *Server) tracks(w http.ResponseWriter, r *http.Request) {
	lib, g := s.State.snapshot()
	out := map[string]any{}
	ids := r.URL.Query()["id"]
	if len(ids) > 500 {
		ids = ids[:500]
	}
	for _, id := range ids {
		if lib == nil {
			break
		}
		if c := lib.Get(id); c != nil {
			out[id] = g.Track(c.Stem)
		}
	}
	writeJSON(w, 200, map[string]any{"tracks": out})
}

func (s *Server) thumb(w http.ResponseWriter, r *http.Request, l *Lease) error {
	clip, cam, ok := s.clipAndCam(w, r)
	if !ok {
		return nil
	}
	path := media.Thumbnail(l.Context(), clip.Files[cam])
	if path == "" {
		writeError(w, 404, "Thumbnail unavailable")
		return nil
	}
	body, err := os.ReadFile(path)
	if err != nil {
		writeError(w, 404, "Thumbnail unavailable")
		return nil
	}
	h := w.Header()
	h.Set("Content-Type", "image/jpeg")
	h.Set("Content-Length", strconv.Itoa(len(body)))
	h.Set("Cache-Control", "public, max-age=604800, immutable")
	_, err = w.Write(body)
	return err
}

func (s *Server) media(w http.ResponseWriter, r *http.Request, l *Lease) error {
	clip, cam, ok := s.clipAndCam(w, r)
	if !ok {
		return nil
	}
	path := clip.Files[cam]
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := w.Header()
	h.Set("Content-Type", "video/mp4")
	h.Set("Cache-Control", "private, max-age=3600")
	if r.URL.Query().Get("download") != "" {
		h.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, asciiName(filepath.Base(path))))
	}
	// ServeContent handles Range (206/416), HEAD and Accept-Ranges.
	http.ServeContent(w, r, "", time.Time{}, leaseReader{f, l})
	return nil
}

func (s *Server) transcode(w http.ResponseWriter, r *http.Request, l *Lease) error {
	if media.FFmpeg == "" {
		writeError(w, 501, "ffmpeg is not installed")
		return nil
	}
	clip, cam, ok := s.clipAndCam(w, r)
	if !ok {
		return nil
	}
	start, _ := strconv.ParseFloat(r.URL.Query().Get("t"), 64)
	stream, err := media.Transcode(l.Context(), clip.Files[cam], start)
	if err != nil {
		writeError(w, 500, "ffmpeg failed to start: "+err.Error())
		return nil
	}
	defer stream.Close()
	h := w.Header()
	h.Set("Content-Type", "video/mp4")
	h.Set("Cache-Control", "no-store")
	h.Set("Connection", "close")
	w.WriteHeader(200)
	buf := make([]byte, 64*1024)
	out := leaseWriter{w, l}
	rc := http.NewResponseController(w)
	for {
		n, rerr := stream.Read(buf)
		if n > 0 {
			if _, err := out.Write(buf[:n]); err != nil {
				return nil
			}
			_ = rc.Flush()
		}
		if rerr != nil {
			return nil
		}
	}
}

func (s *Server) zip(w http.ResponseWriter, r *http.Request, l *Lease) error {
	q := r.URL.Query()
	var ids []string
	var cams string
	var withGPX bool
	if token := q.Get("token"); token != "" {
		job, ok := s.State.export(token)
		if !ok {
			writeError(w, 404, "Export link expired - start the export again")
			return nil
		}
		ids, cams, withGPX = job.ids, job.cams, job.gpx
	} else {
		for _, id := range q["id"] {
			if id != "" {
				ids = append(ids, id)
			}
		}
		cams = q.Get("cams")
		if cams == "" {
			cams = "FRC"
		}
		withGPX = q.Get("gpx") != "0"
	}
	lib, g := s.State.snapshot()
	z, err := buildZip(lib, g, ids, cams, withGPX)
	var ue *UserError
	if errors.As(err, &ue) {
		writeError(w, 404, ue.Msg)
		return nil
	}
	if err != nil {
		return err
	}
	h := w.Header()
	h.Set("Content-Type", "application/zip")
	h.Set("Content-Length", strconv.FormatInt(z.stream.Size(), 10))
	h.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, asciiName(z.name), url.PathEscape(z.name)))
	h.Set("Cache-Control", "no-store")
	w.WriteHeader(200)
	if r.Method == http.MethodHead {
		return nil
	}
	_, _ = z.stream.WriteTo(leaseWriter{w, l})
	return nil
}

func (s *Server) favorites(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"ids": s.State.Favorites().IDs()})
}

func (s *Server) drives(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"drives": volume.Detect()})
}

type dirEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func (s *Server) browse(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		roots := []dirEntry{}
		for _, p := range volume.Roots() {
			roots = append(roots, dirEntry{p, p})
		}
		writeJSON(w, 200, map[string]any{"path": "", "parent": nil, "dirs": roots})
		return
	}
	path, _ = filepath.Abs(path)
	var parent any = filepath.Dir(path)
	if parent == path {
		parent = nil
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		writeJSON(w, 200, map[string]any{"path": path, "parent": filepath.Dir(path), "dirs": []any{}, "error": err.Error()})
		return
	}
	dirs := []dirEntry{}
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") && !strings.HasPrefix(e.Name(), "$") {
			dirs = append(dirs, dirEntry{e.Name(), filepath.Join(path, e.Name())})
		}
	}
	sort.Slice(dirs, func(i, j int) bool { return strings.ToLower(dirs[i].Name) < strings.ToLower(dirs[j].Name) })
	if len(dirs) > 500 {
		dirs = dirs[:500]
	}
	writeJSON(w, 200, map[string]any{"path": path, "parent": parent, "dirs": dirs, "isDashcam": card.LooksLikeDashcamRoot(path)})
}

// ------------------------------------------------------------------- POST

func (s *Server) open(w http.ResponseWriter, r *http.Request) {
	data, err := readJSON(r)
	if err != nil {
		writeError(w, 400, "Invalid JSON")
		return
	}
	path, _ := data["path"].(string)
	if err := s.State.Open(path); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, s.State.Status())
}

func (s *Server) setFavorites(w http.ResponseWriter, r *http.Request) {
	data, err := readJSON(r)
	if err != nil {
		writeError(w, 400, "Invalid JSON")
		return
	}
	var ids []string
	switch v := data["ids"].(type) {
	case nil:
		if id, ok := data["id"].(string); ok && id != "" {
			ids = []string{id}
		}
	case []any:
		for _, x := range v {
			ids = append(ids, fmt.Sprint(x))
		}
	default:
		writeError(w, 400, "ids must be a list")
		return
	}
	on := true
	if v, ok := data["on"].(bool); ok {
		on = v
	}
	out, err := s.State.Favorites().Set(ids, on)
	if err != nil {
		writeError(w, 500, "Can't save favorites: "+err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"ids": out})
}

func (s *Server) export(w http.ResponseWriter, r *http.Request) {
	data, err := readJSON(r)
	if err != nil {
		writeError(w, 400, "Invalid JSON")
		return
	}
	var ids []string
	if list, ok := data["ids"].([]any); ok {
		for _, x := range list {
			if id := fmt.Sprint(x); x != nil && id != "" {
				ids = append(ids, id)
			}
		}
	}
	cams := normalizeCams(fmt.Sprint(valueOr(data["cams"], "FRC")))
	withGPX := true
	if v, ok := data["gpx"].(bool); ok {
		withGPX = v
	}
	lease, err := s.State.UseCard()
	if err != nil {
		writeDisconnected(w)
		return
	}
	lib, g := s.State.snapshot()
	z, err := buildZip(lib, g, ids, cams, withGPX)
	lease.Done()
	var ue *UserError
	if errors.As(err, &ue) {
		writeError(w, 404, ue.Msg)
		return
	}
	if err != nil {
		writeDisconnected(w)
		return
	}
	token := s.State.addExport(ids, cams, withGPX)
	writeJSON(w, 200, map[string]any{"url": "/api/zip?token=" + token, "name": z.name, "size": z.stream.Size(), "count": z.count})
}

func valueOr(v any, def string) any {
	if v == nil || v == "" {
		return def
	}
	return v
}

func normalizeCams(cams string) string {
	var b strings.Builder
	for _, c := range strings.ToUpper(cams) {
		if strings.ContainsRune("FRC", c) {
			b.WriteRune(c)
		}
	}
	if b.Len() == 0 {
		return "FRC"
	}
	return b.String()
}

func (s *Server) disconnect(w http.ResponseWriter, r *http.Request) {
	data, err := readJSON(r)
	if err != nil {
		writeError(w, 400, "Invalid JSON")
		return
	}
	root, released := s.State.Close()
	out := map[string]any{"root": strPtr(root), "released": released, "ejected": false, "message": ""}
	if eject, _ := data["eject"].(bool); eject && root != "" {
		if !released {
			out["message"] = "files on the card are still in use"
		} else {
			out["ejected"], out["message"] = volume.Eject(root)
		}
	}
	out["status"] = s.State.Status()
	writeJSON(w, 200, out)
}

func (s *Server) rescan(w http.ResponseWriter, r *http.Request) {
	root := s.State.Root()
	if root == "" {
		writeError(w, 400, "No folder open")
		return
	}
	if err := s.State.Open(root); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, s.State.Status())
}
