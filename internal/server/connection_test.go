package server

import (
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// Card disconnect handling: detect a pulled card, reconnect, user disconnect,
// and releasing every open file.

func (e *env) status() map[string]any { return e.json("GET", "/api/status", nil) }

func TestPulledCardIsDetectedAndReconnects(t *testing.T) {
	e := setup(t, 1000)
	if c := e.status()["connection"]; c != ConnConnected {
		t.Fatal(c)
	}
	away := e.root + "-away"
	if err := os.Rename(e.root, away); err != nil { // simulate pulling the card
		t.Fatal(err)
	}
	if got := e.st.CheckConnection(); got != "lost" {
		t.Fatal(got)
	}
	st := e.status()
	if st["connection"] != ConnLost || st["lastRoot"] != e.root {
		t.Fatal(st)
	}
	if r := e.req("HEAD", "/media?id="+cid+"&cam=R", nil, nil); r.StatusCode != 503 {
		t.Fatal(r.StatusCode)
	}
	if got := e.st.CheckConnection(); got != "" {
		t.Fatalf("nothing to do while the card is away, got %q", got)
	}
	if err := os.Rename(away, e.root); err != nil { // plug it back in
		t.Fatal(err)
	}
	if got := e.st.CheckConnection(); got != "reconnected" {
		t.Fatal(got)
	}
	if c := e.status()["connection"]; c != ConnConnected {
		t.Fatal(c)
	}
	if r := e.req("HEAD", "/media?id="+cid+"&cam=R", nil, nil); r.StatusCode != 200 {
		t.Fatal(r.StatusCode)
	}
}

func TestUserDisconnectReleasesCard(t *testing.T) {
	e := setup(t, 1000)
	d := e.json("POST", "/api/disconnect", map[string]any{})
	st := d["status"].(map[string]any)
	if d["root"] != e.root || d["released"] != true || st["connection"] != ConnClosed || st["root"] != nil {
		t.Fatal(d)
	}
	if r := e.get("/media?id=" + cid + "&cam=F"); r.StatusCode != 503 || !strings.Contains(string(r.body), "disconnected") {
		t.Fatal(r.StatusCode)
	}
	if clips := e.json("GET", "/api/library", nil)["clips"].([]any); len(clips) != 0 {
		t.Fatal(clips)
	}
	if got := e.st.CheckConnection(); got != "" {
		t.Fatalf("a closed card is not reopened automatically, got %q", got)
	}
	if r := e.req("POST", "/api/open", nil, map[string]string{"path": e.root}); r.StatusCode != 200 {
		t.Fatal(r.StatusCode)
	}
	if c := e.status()["connection"]; c != ConnConnected {
		t.Fatal(c)
	}
}

func TestDisconnectUnblocksAStalledStream(t *testing.T) {
	e := setup(t, 8*1024*1024)
	// A browser that buffered enough stops reading: our writer blocks with the file open.
	res, err := http.Get(e.srv.URL + "/media?id=" + cid + "&cam=F")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	buf := make([]byte, 1024)
	if _, err := res.Body.Read(buf); err != nil { // read a little, then stall
		t.Fatal(err)
	}
	time.Sleep(400 * time.Millisecond)
	if e.st.Busy() == 0 {
		t.Fatal("stream should hold the card")
	}
	start := time.Now()
	if _, released := e.st.Close(); !released {
		t.Fatal("not released")
	}
	if d := time.Since(start); d > 3*time.Second {
		t.Fatalf("release took %s", d)
	}
	if e.st.Busy() != 0 {
		t.Fatal("still busy")
	}
	// nothing holds the files open any more: the folder can be moved
	if err := os.Rename(e.root, e.root+"-away"); err != nil {
		t.Fatal(err)
	}
	_ = os.Rename(e.root+"-away", e.root)
}

func TestEjectRefusesNonRemovableFolder(t *testing.T) {
	e := setup(t, 1000)
	d := e.json("POST", "/api/disconnect", map[string]any{"eject": true})
	if d["ejected"] != false || !strings.Contains(d["message"].(string), "not a removable drive") {
		t.Fatal(d)
	}
	if _, err := os.Stat(e.root); err != nil {
		t.Fatal(err)
	}
}
