//go:build !windows

package desktop

import "context"

// runWebview: an embedded webview on macOS/Linux needs cgo (WebKit), which
// would rule out single-command cross-compiling. Those platforms use the
// Chromium --app window instead.
func runWebview(context.Context, Options) error { return ErrUnavailable }
