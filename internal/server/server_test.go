package server

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jossef/dashcam-studio/internal/testcard"
)

const (
	stem  = "NO20261113-143658-004641"
	stem2 = "NO20261113-143758-004642"
)

var cid = url.QueryEscape("Normal/" + stem)

type env struct {
	t    *testing.T
	root string
	home string
	st   *State
	srv  *httptest.Server
}

func setup(t *testing.T, size int) *env {
	t.Helper()
	home := t.TempDir()
	t.Setenv("DASHCAM_STUDIO_HOME", home)
	var lines []string
	for i := 0; i < 60; i += 3 {
		lines = append(lines, fmt.Sprintf("%d,A,32.07%03d,34.82,0,500,0,0,0,%sF.MP4,0,0,0", 1794537418+i, i, stem))
	}
	root := testcard.Make(t, "", "Normal", []testcard.Clip{testcard.C(stem, "FRC", size), testcard.C(stem2, "FR", 1000)}, lines)
	st := NewState()
	if err := st.Open(root); err != nil {
		t.Fatal(err)
	}
	static := fstest.MapFS{
		"index.html":     {Data: []byte(`<html dir="rtl"><title>Dashcam Studio</title><script src="./assets/app.js"></script></html>`)},
		"assets/app.js":  {Data: []byte("console.log(1)")},
		"assets/app.css": {Data: []byte("body{}")},
	}
	srv := httptest.NewServer(New(st, static))
	t.Cleanup(func() {
		st.Release(2 * time.Second)
		srv.Close()
	})
	return &env{t, root, home, st, srv}
}

type resp struct {
	*http.Response
	body []byte
}

func (e *env) req(method, path string, headers map[string]string, body any) resp {
	e.t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	r, err := http.NewRequest(method, e.srv.URL+path, rd)
	if err != nil {
		e.t.Fatal(err)
	}
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	res, err := e.srv.Client().Do(r)
	if err != nil {
		e.t.Fatal(err)
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)
	return resp{res, data}
}

func (e *env) get(path string) resp { return e.req("GET", path, nil, nil) }

func (e *env) json(method, path string, body any) map[string]any {
	e.t.Helper()
	r := e.req(method, path, nil, body)
	out := map[string]any{}
	if err := json.Unmarshal(r.body, &out); err != nil {
		e.t.Fatalf("%s %s: %v (%d %q)", method, path, err, r.StatusCode, r.body)
	}
	return out
}

func (e *env) file(cam, s string) []byte {
	data, err := os.ReadFile(filepath.Join(e.root, "Normal", testcard.CamDir[rune(cam[0])], s+cam+".MP4"))
	if err != nil {
		e.t.Fatal(err)
	}
	return data
}

func unzip(t *testing.T, data []byte) map[string][]byte {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string][]byte{}
	for _, f := range zr.File {
		rc, _ := f.Open()
		b, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("%s: %v", f.Name, err)
		}
		out[f.Name] = b
	}
	return out
}

func keys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// ---------------------------------------------------------------------- tests

func TestIndexAndAssets(t *testing.T) {
	e := setup(t, 50_000)
	r := e.get("/")
	if r.StatusCode != 200 || !bytes.Contains(r.body, []byte(`dir="rtl"`)) {
		t.Fatalf("%d %s", r.StatusCode, r.body)
	}
	if cc := r.Header.Get("Cache-Control"); cc != "no-cache" {
		t.Fatal(cc)
	}
	a := e.req("HEAD", "/assets/app.js", nil, nil)
	if a.StatusCode != 200 || !strings.Contains(a.Header.Get("Cache-Control"), "immutable") ||
		!strings.HasPrefix(a.Header.Get("Content-Type"), "text/javascript") {
		t.Fatalf("%d %v", a.StatusCode, a.Header)
	}
}

func TestStaticTraversalBlocked(t *testing.T) {
	e := setup(t, 1000)
	for _, p := range []string{"/../server.go", "/..%2Fserver.go", "/assets/../../go.mod"} {
		if r := e.get(p); r.StatusCode == 200 {
			t.Errorf("%s served", p)
		}
	}
}

func TestLibrary(t *testing.T) {
	e := setup(t, 50_000)
	lib := e.json("GET", "/api/library", nil)
	clips := lib["clips"].([]any)
	if len(clips) != 2 {
		t.Fatal(len(clips))
	}
	c := clips[0].(map[string]any)
	if c["id"] != "Normal/"+stem || c["start"] != "2026-11-13T14:36:58" || c["kind"] != "Normal" {
		t.Fatal(c)
	}
	if got := keys(c["cams"].(map[string]any)); !reflect.DeepEqual(got, []string{"C", "F", "R"}) {
		t.Fatal(got)
	}
	if _, ok := c["gps"]; !ok {
		t.Fatal("gps summary missing")
	}
	if got := keys(clips[1].(map[string]any)["cams"].(map[string]any)); !reflect.DeepEqual(got, []string{"F", "R"}) {
		t.Fatal(got)
	}
	if lib["hasGps"] != true {
		t.Fatal(lib["hasGps"])
	}
}

func TestClipDetail(t *testing.T) {
	e := setup(t, 50_000)
	d := e.json("GET", "/api/clip?id="+cid, nil)
	cams := d["cams"].(map[string]any)
	if got := keys(cams); !reflect.DeepEqual(got, []string{"C", "F", "R"}) {
		t.Fatal(got)
	}
	f := cams["F"].(map[string]any)
	if f["name"] != "Front" || f["file"] != stem+"F.MP4" || f["size"] != 50_000.0 {
		t.Fatal(f)
	}
	if tr := d["track"].([]any); tr[0].([]any)[0] != 0.0 {
		t.Fatal(tr[0])
	}
	if r := e.get("/api/clip?id=nope"); r.StatusCode != 404 {
		t.Fatal(r.StatusCode)
	}
}

func TestMediaFullAndRanges(t *testing.T) {
	e := setup(t, 50_000)
	raw := e.file("R", stem)
	path := "/media?id=" + cid + "&cam=R"
	r := e.get(path)
	if r.StatusCode != 200 || !bytes.Equal(r.body, raw) || r.Header.Get("Accept-Ranges") != "bytes" {
		t.Fatalf("full: %d", r.StatusCode)
	}
	r = e.req("GET", path, map[string]string{"Range": "bytes=100-199"}, nil)
	if r.StatusCode != 206 || !bytes.Equal(r.body, raw[100:200]) || r.Header.Get("Content-Range") != fmt.Sprintf("bytes 100-199/%d", len(raw)) {
		t.Fatalf("range: %d %v", r.StatusCode, r.Header)
	}
	if r = e.req("GET", path, map[string]string{"Range": "bytes=49990-"}, nil); !bytes.Equal(r.body, raw[49990:]) {
		t.Fatal("open range")
	}
	if r = e.req("GET", path, map[string]string{"Range": "bytes=-10"}, nil); !bytes.Equal(r.body, raw[len(raw)-10:]) {
		t.Fatal("suffix range")
	}
	if r = e.req("GET", path, map[string]string{"Range": "bytes=999999-"}, nil); r.StatusCode != 416 {
		t.Fatal(r.StatusCode)
	}
	if r = e.get("/media?id=" + url.QueryEscape("Normal/"+stem2) + "&cam=C"); r.StatusCode != 404 {
		t.Fatal(r.StatusCode)
	}
}

func TestMediaDownloadHeader(t *testing.T) {
	e := setup(t, 1000)
	r := e.req("HEAD", "/media?id="+cid+"&cam=F&download=1", nil, nil)
	if !strings.Contains(r.Header.Get("Content-Disposition"), "attachment") {
		t.Fatal(r.Header)
	}
}

func TestZipSingleClip(t *testing.T) {
	e := setup(t, 50_000)
	r := e.get("/api/zip?id=" + cid)
	if r.StatusCode != 200 || r.Header.Get("Content-Length") != fmt.Sprint(len(r.body)) {
		t.Fatalf("%d %v", r.StatusCode, r.Header)
	}
	if !strings.Contains(r.Header.Get("Content-Disposition"), "dashcam_2026-11-13_14-36-58_004641.zip") {
		t.Fatal(r.Header.Get("Content-Disposition"))
	}
	files := unzip(t, r.body)
	want := []string{stem + ".gpx", stem + "C.MP4", stem + "F.MP4", stem + "R.MP4"}
	if got := keys(files); !reflect.DeepEqual(got, want) {
		t.Fatal(got)
	}
	if !bytes.Equal(files[stem+"F.MP4"], e.file("F", stem)) || !bytes.Contains(files[stem+".gpx"], []byte("<trkpt")) {
		t.Fatal("content")
	}
}

func TestZipWithoutGPXAndTrip(t *testing.T) {
	e := setup(t, 50_000)
	ids := "id=" + cid + "&id=" + url.QueryEscape("Normal/"+stem2)
	files := unzip(t, e.get("/api/zip?"+ids+"&gpx=0").body)
	if len(files) != 5 {
		t.Fatal(keys(files))
	}
	for name := range files {
		if !strings.Contains(name, "/") || strings.HasSuffix(name, ".gpx") {
			t.Fatal(name)
		}
	}
	if r := e.req("HEAD", "/api/zip?"+ids, nil, nil); !strings.Contains(r.Header.Get("Content-Disposition"), "2clips") {
		t.Fatal(r.Header)
	}
}

func TestZipUnknown(t *testing.T) {
	if r := setup(t, 1000).get("/api/zip?id=x"); r.StatusCode != 404 {
		t.Fatal(r.StatusCode)
	}
}

func TestTracks(t *testing.T) {
	d := setup(t, 1000).json("GET", "/api/tracks?id="+cid, nil)
	if n := len(d["tracks"].(map[string]any)["Normal/"+stem].([]any)); n != 20 {
		t.Fatal(n)
	}
}

func TestOpenBadFolder(t *testing.T) {
	e := setup(t, 1000)
	r := e.req("POST", "/api/open", nil, map[string]string{"path": filepath.Join(e.root, "does-not-exist")})
	if r.StatusCode != 400 || !strings.Contains(string(r.body), "not found") {
		t.Fatalf("%d %s", r.StatusCode, r.body)
	}
	// still serving the previous library
	if n := e.json("GET", "/api/status", nil)["clipCount"]; n != 2.0 {
		t.Fatal(n)
	}
}

func TestFavoritesRoundtrip(t *testing.T) {
	e := setup(t, 1000)
	a, b := "Normal/"+stem, "Normal/"+stem2
	d := e.json("POST", "/api/favorites", map[string]any{"ids": []string{a, b}, "on": true})
	if !reflect.DeepEqual(d["ids"], []any{a, b}) {
		t.Fatal(d)
	}
	d = e.json("POST", "/api/favorites", map[string]any{"ids": []string{b}, "on": false})
	if !reflect.DeepEqual(d["ids"], []any{a}) {
		t.Fatal(d)
	}
	if d = e.json("GET", "/api/favorites", nil); !reflect.DeepEqual(d["ids"], []any{a}) {
		t.Fatal(d)
	}
	// persisted in the config dir, never on the card
	saved, err := os.ReadFile(filepath.Join(e.home, "favorites.json"))
	if err != nil || !bytes.Contains(saved, []byte(a)) {
		t.Fatal(err, string(saved))
	}
	_ = filepath.Walk(e.root, func(p string, _ os.FileInfo, _ error) error {
		if strings.Contains(filepath.Base(p), "favorites") {
			t.Errorf("wrote to the card: %s", p)
		}
		return nil
	})
	if r := e.req("POST", "/api/favorites", nil, map[string]any{"ids": "nope"}); r.StatusCode != 400 {
		t.Fatal(r.StatusCode)
	}
}

func TestExportMultipleClips(t *testing.T) {
	e := setup(t, 50_000)
	job := e.json("POST", "/api/export", map[string]any{"ids": []string{"Normal/" + stem, "Normal/" + stem2}, "cams": "FR", "gpx": false})
	if job["count"] != 2.0 || !strings.HasSuffix(job["name"].(string), "_2clips.zip") {
		t.Fatal(job)
	}
	r := e.get(job["url"].(string))
	if r.StatusCode != 200 || float64(len(r.body)) != job["size"] {
		t.Fatal(r.StatusCode, len(r.body), job["size"])
	}
	want := []string{stem + "/" + stem + "F.MP4", stem + "/" + stem + "R.MP4", stem2 + "/" + stem2 + "F.MP4", stem2 + "/" + stem2 + "R.MP4"}
	if got := keys(unzip(t, r.body)); !reflect.DeepEqual(got, want) {
		t.Fatal(got)
	}
	if r := e.get("/api/zip?token=nope"); r.StatusCode != 404 {
		t.Fatal(r.StatusCode)
	}
	if r := e.req("POST", "/api/export", nil, map[string]any{"ids": []string{"Normal/nope"}}); r.StatusCode != 404 {
		t.Fatal(r.StatusCode)
	}
}

func TestBrowse(t *testing.T) {
	e := setup(t, 1000)
	d := e.json("GET", "/api/browse?path="+url.QueryEscape(e.root), nil)
	if d["isDashcam"] != true {
		t.Fatal(d)
	}
	found := false
	for _, x := range d["dirs"].([]any) {
		found = found || x.(map[string]any)["name"] == "Normal"
	}
	if !found {
		t.Fatal(d["dirs"])
	}
	if roots := e.json("GET", "/api/browse", nil)["dirs"].([]any); len(roots) == 0 {
		t.Fatal("no roots")
	}
}

func TestInvalidJSONAndUnknownRoutes(t *testing.T) {
	e := setup(t, 1000)
	r, _ := http.Post(e.srv.URL+"/api/open", "application/json", strings.NewReader("{nope"))
	if r.StatusCode != 400 {
		t.Fatal(r.StatusCode)
	}
	if r := e.get("/api/nothing"); r.StatusCode != 404 {
		t.Fatal(r.StatusCode)
	}
	if r := e.req("POST", "/api/nothing", nil, map[string]any{}); r.StatusCode != 404 {
		t.Fatal(r.StatusCode)
	}
}
