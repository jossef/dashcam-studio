// Command dashcam-studio plays Front / Rear / Cabin dashcam clips side by side.
//
//	dashcam-studio [ROOT] [--port N] [--host H] [--ui auto|window|app|browser|none]
//
// It serves the embedded web UI and its API on 127.0.0.1 and shows it in an
// app window. Closing the window quits the app.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/jossef/dashcam-studio/internal/desktop"
	"github.com/jossef/dashcam-studio/internal/media"
	"github.com/jossef/dashcam-studio/internal/paths"
	"github.com/jossef/dashcam-studio/internal/server"
	"github.com/jossef/dashcam-studio/internal/webui"
)

// version is set by the release build (-ldflags "-X main.version=...").
var version = "dev"

// The UI window and its message loop must stay on the main OS thread.
func init() { runtime.LockOSThread() }

type options struct {
	root    string
	port    int
	host    string
	ui      string
	devURL  string
	debug   bool
	version bool
}

func parseArgs(args []string, stderr io.Writer) (options, error) {
	var o options
	fs := flag.NewFlagSet("dashcam-studio", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.IntVar(&o.port, "port", 8765, "port to listen on (the next free one is used if taken)")
	fs.StringVar(&o.host, "host", "127.0.0.1", "interface to bind (default: local only)")
	fs.StringVar(&o.ui, "ui", desktop.ModeAuto, "how to show the UI: auto, window, app, browser or none")
	noBrowser := fs.Bool("no-browser", false, "don't open any window (same as --ui none)")
	fs.StringVar(&o.devURL, "dev-url", "", "open this URL instead of the embedded UI (Vite dev server)")
	fs.BoolVar(&o.debug, "debug", false, "log requests and enable the window's developer tools")
	fs.BoolVar(&o.version, "version", false, "print the version and exit")
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Dashcam Studio %s - play Front / Rear / Cabin dashcam clips side by side.\n\n", version)
		fmt.Fprintf(stderr, "Usage: dashcam-studio [ROOT] [flags]\n\n  ROOT  dashcam folder, e.g. E:\\ or /Volumes/DASHCAM (optional: pick it in the app)\n\nFlags:\n")
		fs.PrintDefaults()
	}
	// Accept the folder before or after the flags.
	rest := args
	for {
		if err := fs.Parse(rest); err != nil {
			return o, err
		}
		if fs.NArg() == 0 {
			break
		}
		if o.root != "" {
			return o, fmt.Errorf("unexpected argument %q", fs.Arg(0))
		}
		o.root, rest = fs.Arg(0), fs.Args()[1:]
	}
	if *noBrowser {
		o.ui = desktop.ModeNone
	}
	switch o.ui {
	case desktop.ModeAuto, desktop.ModeWindow, desktop.ModeApp, desktop.ModeBrowser, desktop.ModeNone:
	default:
		err := fmt.Errorf("unknown --ui mode %q (use auto, window, app, browser or none)", o.ui)
		fmt.Fprintln(stderr, err)
		return o, err
	}
	return o, nil
}

func main() {
	interactive := attachConsole()
	o, err := parseArgs(os.Args[1:], os.Stderr)
	if err == flag.ErrHelp {
		return
	}
	if err != nil {
		os.Exit(2)
	}
	if o.version {
		fmt.Println(version)
		return
	}
	if err := run(o, interactive); err != nil {
		log.Printf("error: %v", err)
		if !interactive {
			fatalDialog(err.Error()) // double-clicked: there is no terminal to show the error in
		}
		os.Exit(1)
	}
}

func run(o options, interactive bool) error {
	log.SetFlags(0)
	log.SetPrefix("  ")
	state := server.NewState()
	if o.root != "" {
		if err := state.Open(o.root); err != nil {
			log.Printf("! %v\n  You can choose another folder in the app.", err)
		} else {
			st := state.Status()
			gpsNote := ""
			if len(st.GPSFiles) > 0 {
				gpsNote = "  (+ GPS: " + strings.Join(st.GPSFiles, ", ") + ")"
			}
			log.Printf("Loaded %d clips from %s%s", st.ClipCount, *st.Root, gpsNote)
		}
	}

	ln, err := server.Listen(o.host, o.port, 20)
	if err != nil {
		return err
	}
	srv := server.New(state, webui.FS())
	srv.Debug = o.debug

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go state.Watch(ctx, server.WatchInterval) // notices when the card is pulled out / plugged back in
	served := make(chan error, 1)
	go func() { served <- srv.Serve(ctx, ln) }()

	port := ln.Addr().(*net.TCPAddr).Port
	host := o.host
	if host == "0.0.0.0" || host == "::" || host == "" {
		host = "127.0.0.1"
	}
	url := "http://" + net.JoinHostPort(host, strconv.Itoa(port)) + "/"
	log.Printf("Dashcam Studio %s  ->  %s", version, url)
	if !webui.Built() && o.devURL == "" {
		log.Printf("(this build has no UI bundle; run `go run ./tools/build`)")
	}
	if media.FFmpeg == "" {
		log.Printf("(ffmpeg not found: thumbnails and the compatibility player are disabled)")
	}
	if o.ui == desktop.ModeNone || interactive {
		log.Printf("Press Ctrl+C to stop.")
	}
	uiURL := url
	if o.devURL != "" {
		uiURL = o.devURL
	}
	err = desktop.Run(ctx, desktop.Options{
		URL: uiURL, Title: "Dashcam Studio", Mode: o.ui, DataDir: paths.CacheDir(), Debug: o.debug,
		Idle: srv.IdleSince, Interactive: interactive,
	})
	stop() // window closed: shut the server down
	if serr := <-served; err == nil {
		err = serr
	}
	return err
}
