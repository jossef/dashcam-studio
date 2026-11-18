// Package desktop shows the UI in a native-feeling window.
//
// Preference order for --ui auto:
//  1. window  - an embedded WebView2 window (Windows; WebView2 ships with Windows 10/11)
//  2. app     - Chrome / Edge / Chromium / Brave in --app mode: its own window, no tabs
//     or address bar, separate profile
//  3. browser - a tab in the default browser
//
// Run blocks until the user closes the window. In browser mode there is no
// window to watch, so the app exits after the page has been gone for a while
// (unless it runs in a terminal, where Ctrl+C stops it).
package desktop

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"
)

// Modes accepted by --ui.
const (
	ModeAuto    = "auto"
	ModeWindow  = "window"
	ModeApp     = "app"
	ModeBrowser = "browser"
	ModeNone    = "none"
)

// IdleExit is how long the page may be gone (no requests) before a browser-mode
// session ends. Background tabs still poll at least once a minute.
const IdleExit = 3 * time.Minute

// Options configure the UI.
type Options struct {
	URL         string
	Title       string
	Mode        string
	DataDir     string               // per-user folder for the window's browser profile
	Debug       bool                 // enable devtools / context menu
	Idle        func() time.Duration // time since the last request (browser mode)
	Interactive bool                 // a terminal is attached: never exit on idle
}

// ErrUnavailable means a UI mode cannot be used on this machine.
var ErrUnavailable = errors.New("not available")

// Run shows the UI and blocks until it is closed or ctx is cancelled.
func Run(ctx context.Context, o Options) error {
	switch o.Mode {
	case ModeNone:
		<-ctx.Done()
		return nil
	case ModeWindow:
		return runWebview(ctx, o)
	case ModeApp:
		return runAppMode(ctx, o)
	case ModeBrowser:
		return runBrowser(ctx, o)
	case ModeAuto, "":
		if err := runWebview(ctx, o); !errors.Is(err, ErrUnavailable) {
			return err
		}
		if err := runAppMode(ctx, o); !errors.Is(err, ErrUnavailable) {
			return err
		}
		return runBrowser(ctx, o)
	}
	return fmt.Errorf("unknown --ui mode %q (use auto, window, app, browser or none)", o.Mode)
}

func runBrowser(ctx context.Context, o Options) error {
	if err := openBrowser(o.URL); err != nil {
		log.Printf("could not open a browser (%v); open %s yourself", err, o.URL)
	}
	waitForPage(ctx, o)
	return nil
}

// waitForPage blocks until ctx ends or, without a terminal, the page has been
// gone for IdleExit.
func waitForPage(ctx context.Context, o Options) {
	if o.Interactive || o.Idle == nil {
		<-ctx.Done()
		return
	}
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if o.Idle() > IdleExit {
				log.Printf("no page has talked to the server for %s; exiting", IdleExit)
				return
			}
		}
	}
}
