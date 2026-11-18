// Command build is the single build pipeline for Dashcam Studio: it builds the
// web UI with npm and the Go app, in parallel where possible, and packages one
// self-contained release file per OS.
//
//	go run ./tools/build             release: UI ‖ Go tests → every target → dist/
//	go run ./tools/build build       UI → binary for this machine (fast) → dist/
//	go run ./tools/build dev         Vite (hot reload) + Go API (restarts on .go changes)
//	go run ./tools/build test        Go tests ‖ UI type-check + vitest
//	go run ./tools/build ui          only the UI bundle
//
// Needs Go and Node.js (npm). Everything is cross-compiled with CGO disabled,
// so a Windows, macOS or Linux machine can build all release files.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	appName = "dashcam-studio"
	pkgMain = "./cmd/dashcam-studio"
)

var defaultTargets = "windows/amd64,darwin/universal,linux/amd64"

type config struct {
	root      string
	out       string
	version   string
	targets   []string
	skipTests bool
	window    bool
	noOpen    bool
	port      int
}

func main() {
	cmd := "release"
	args := os.Args[1:]
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cmd, args = args[0], args[1:]
	}
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	var c config
	targets := fs.String("targets", defaultTargets, "release targets: GOOS/GOARCH list; darwin/universal makes one macOS binary for Intel + Apple silicon")
	fs.StringVar(&c.out, "out", "dist", "output folder")
	fs.StringVar(&c.version, "version", "", "version stamped into the app (default: web/package.json version)")
	fs.BoolVar(&c.skipTests, "skip-tests", false, "skip go vet / go test / vitest in release")
	fs.BoolVar(&c.window, "window", false, "dev: show the UI in the app window instead of your browser")
	fs.IntVar(&c.port, "port", 8765, "dev: Go API port")
	fs.BoolVar(&c.noOpen, "no-open", false, "dev: don't open the UI automatically")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: go run ./tools/build [release|build|dev|test|ui] [flags]")
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)
	c.targets = strings.Split(*targets, ",")

	root, err := repoRoot()
	if err != nil {
		fail(err)
	}
	c.root = root
	if c.version == "" {
		c.version = packageVersion(root)
	}
	if !filepath.IsAbs(c.out) {
		c.out = filepath.Join(root, c.out)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	start := time.Now()
	switch cmd {
	case "release":
		err = release(ctx, c, c.targets)
	case "build":
		err = release(ctx, config{root: c.root, out: c.out, version: c.version, skipTests: true}, []string{runtime.GOOS + "/" + runtime.GOARCH})
	case "dev":
		err = dev(ctx, c)
	case "test":
		err = parallel(ctx,
			step{"go", func(ctx context.Context, w io.Writer) error {
				if err := goChecks(ctx, c, w); err != nil {
					return err
				}
				return run(ctx, w, c.root, nil, "go", "test", "./cmd/...")
			}},
			step{"ui", func(ctx context.Context, w io.Writer) error { return uiChecks(ctx, c, w) }},
		)
	case "ui":
		err = buildUI(ctx, c, os.Stdout)
	default:
		fs.Usage()
		os.Exit(2)
	}
	if err != nil {
		fail(err)
	}
	if cmd != "dev" {
		fmt.Printf("\n✓ %s done in %s\n", cmd, time.Since(start).Round(100*time.Millisecond))
	}
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "\n✗ %v\n", err)
	os.Exit(1)
}

// ---------------------------------------------------------------- pipeline

func release(ctx context.Context, c config, targets []string) error {
	fmt.Printf("Dashcam Studio %s → %s\n\n", c.version, rel(c.root, c.out))
	// Stage 1: the UI bundle and the Go checks don't depend on each other.
	steps := []step{{"ui", func(ctx context.Context, w io.Writer) error {
		if !c.skipTests {
			if err := uiChecks(ctx, c, w); err != nil {
				return err
			}
		}
		return buildUI(ctx, c, w)
	}}}
	if !c.skipTests {
		steps = append(steps, step{"go", func(ctx context.Context, w io.Writer) error { return goChecks(ctx, c, w) }})
	}
	if err := parallel(ctx, steps...); err != nil {
		return err
	}
	if !c.skipTests { // the packages that embed the UI, now that it is built
		if err := parallel(ctx, step{"go", func(ctx context.Context, w io.Writer) error {
			if err := run(ctx, w, c.root, nil, "go", "vet", "./cmd/...", "./internal/webui"); err != nil {
				return err
			}
			return run(ctx, w, c.root, nil, "go", "test", "./cmd/...")
		}}); err != nil {
			return err
		}
	}
	// Stage 2: compile every target (they embed the fresh UI bundle) in parallel.
	if err := os.MkdirAll(c.out, 0o755); err != nil {
		return err
	}
	var builds []step
	var mu sync.Mutex
	var artifacts []string
	for _, t := range targets {
		t := strings.TrimSpace(t)
		builds = append(builds, step{t, func(ctx context.Context, w io.Writer) error {
			path, err := buildTarget(ctx, c, t, w)
			if err == nil {
				mu.Lock()
				artifacts = append(artifacts, path)
				mu.Unlock()
			}
			return err
		}})
	}
	err := parallel(ctx, builds...)
	_ = os.Remove(filepath.Join(c.out, ".build")) // only if empty (dev keeps its binary there)
	if err != nil {
		return err
	}
	if err := writeChecksums(c.out, artifacts); err != nil {
		return err
	}
	fmt.Println("\nRelease files:")
	for _, a := range artifacts {
		st, _ := os.Stat(a)
		fmt.Printf("  %-52s %6.1f MB\n", rel(c.root, a), float64(st.Size())/1e6)
	}
	return nil
}

func goChecks(ctx context.Context, c config, w io.Writer) error {
	// Leave out the packages that embed the UI bundle: it is being rebuilt in
	// parallel, and the final `go build` compiles them anyway.
	list := exec.CommandContext(ctx, "go", "list", "./internal/...", "./tools/...")
	list.Dir = c.root
	out, err := list.Output()
	if err != nil {
		return fmt.Errorf("go list: %w", err)
	}
	var pkgs []string
	for _, p := range strings.Fields(string(out)) {
		if !strings.HasSuffix(p, "/internal/webui") {
			pkgs = append(pkgs, p)
		}
	}
	if err := run(ctx, w, c.root, nil, "go", append([]string{"vet"}, pkgs...)...); err != nil {
		return err
	}
	return run(ctx, w, c.root, nil, "go", append([]string{"test"}, pkgs...)...)
}

func npm() string {
	if runtime.GOOS == "windows" {
		return "npm.cmd"
	}
	return "npm"
}

// npmInstall runs `npm ci` when node_modules is missing or older than the lock file.
func npmInstall(ctx context.Context, c config, w io.Writer) error {
	web := filepath.Join(c.root, "web")
	lock, err := os.Stat(filepath.Join(web, "package-lock.json"))
	if err != nil {
		return run(ctx, w, web, nil, npm(), "install")
	}
	marker, err := os.Stat(filepath.Join(web, "node_modules", ".package-lock.json"))
	if err == nil && !marker.ModTime().Before(lock.ModTime()) {
		return nil
	}
	return run(ctx, w, web, nil, npm(), "ci", "--no-audit", "--no-fund")
}

var npmOnce sync.Mutex

func uiChecks(ctx context.Context, c config, w io.Writer) error {
	npmOnce.Lock()
	err := npmInstall(ctx, c, w)
	npmOnce.Unlock()
	if err != nil {
		return err
	}
	if err := run(ctx, w, filepath.Join(c.root, "web"), nil, npm(), "run", "typecheck"); err != nil {
		return err
	}
	return run(ctx, w, filepath.Join(c.root, "web"), nil, npm(), "test")
}

func buildUI(ctx context.Context, c config, w io.Writer) error {
	npmOnce.Lock()
	err := npmInstall(ctx, c, w)
	npmOnce.Unlock()
	if err != nil {
		return err
	}
	return run(ctx, w, filepath.Join(c.root, "web"), nil, npm(), "exec", "--", "vite", "build", "--logLevel", "warn")
}

// buildTarget compiles one GOOS/GOARCH and packages it; it returns the release file.
func buildTarget(ctx context.Context, c config, target string, w io.Writer) (string, error) {
	goos, goarch, ok := strings.Cut(target, "/")
	if !ok {
		return "", fmt.Errorf("bad target %q (want GOOS/GOARCH)", target)
	}
	tmp := filepath.Join(c.out, ".build", goos+"-"+goarch)
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp) // only the packaged release file stays in dist/
	if goos == "darwin" && goarch == "universal" {
		var slices []string
		for _, arch := range []string{"amd64", "arm64"} {
			bin := filepath.Join(tmp, appName+"-"+arch)
			if err := goBuild(ctx, c, "darwin", arch, bin, w); err != nil {
				return "", err
			}
			slices = append(slices, bin)
		}
		bin := filepath.Join(tmp, appName)
		if err := makeUniversal(bin, slices); err != nil {
			return "", err
		}
		return packageMac(c, bin, "universal")
	}
	bin := filepath.Join(tmp, appName)
	if goos == "windows" {
		bin += ".exe"
		if err := writeWindowsResources(c, goarch); err != nil {
			return "", err
		}
	}
	if err := goBuild(ctx, c, goos, goarch, bin, w); err != nil {
		return "", err
	}
	switch goos {
	case "windows":
		return copyArtifact(bin, filepath.Join(c.out, fmt.Sprintf("%s-%s-windows-%s.exe", appName, c.version, archName(goarch))))
	case "darwin":
		return packageMac(c, bin, archName(goarch))
	default:
		return packageTarGz(bin, filepath.Join(c.out, fmt.Sprintf("%s-%s-%s-%s.tar.gz", appName, c.version, goos, archName(goarch))))
	}
}

func goBuild(ctx context.Context, c config, goos, goarch, out string, w io.Writer) error {
	ldflags := "-s -w -X main.version=" + c.version
	if goos == "windows" {
		ldflags += " -H windowsgui" // no console window; the app attaches to a terminal when there is one
	}
	env := []string{"GOOS=" + goos, "GOARCH=" + goarch, "CGO_ENABLED=0"}
	return run(ctx, w, c.root, env, "go", "build", "-trimpath", "-ldflags", ldflags, "-o", out, pkgMain)
}

func archName(goarch string) string {
	if goarch == "amd64" {
		return "x64"
	}
	return goarch
}

// ------------------------------------------------------------------- dev

// dev runs the Vite dev server (hot reload) next to the Go API, restarting the
// Go side whenever a .go file changes. The UI is opened on Vite's URL, which
// proxies /api and /media to Go.
func dev(ctx context.Context, c config) error {
	// The Vite proxy must reach this exact port, so don't let the app fall
	// through to the next free one (a stale server would get the requests).
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", c.port))
	if err != nil {
		return fmt.Errorf("port %d is in use (an old server still running?); stop it or pass -port N", c.port)
	}
	ln.Close()
	if err := npmInstall(ctx, c, os.Stdout); err != nil {
		return err
	}
	const viteURL = "http://localhost:5173/"
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	wg.Add(2)
	go func() {
		defer wg.Done()
		env := []string{fmt.Sprintf("DASHCAM_API=http://127.0.0.1:%d", c.port)}
		err := run(ctx, prefixed("vite", os.Stdout), filepath.Join(c.root, "web"), env, npm(), "run", "dev", "--", "--strictPort")
		if ctx.Err() == nil {
			errs <- fmt.Errorf("vite stopped: %v", err)
		}
	}()
	go func() {
		defer wg.Done()
		args := []string{"--port", fmt.Sprint(c.port), "--debug", "--dev-url", viteURL}
		if !c.window {
			args = append(args, "--ui", "none")
		}
		if err := watchGo(ctx, c, args); err != nil && ctx.Err() == nil {
			errs <- err
		}
	}()
	if !c.window && !c.noOpen {
		go func() {
			time.Sleep(2 * time.Second)
			if ctx.Err() == nil {
				fmt.Printf("\n  Opening %s  (Ctrl+C stops everything)\n\n", viteURL)
				_ = openURL(viteURL)
			}
		}()
	}
	select {
	case <-ctx.Done():
	case err := <-errs:
		return err
	}
	wg.Wait()
	return nil
}

// watchGo builds and runs the app, rebuilding it when Go sources change.
func watchGo(ctx context.Context, c config, args []string) error {
	bin := filepath.Join(c.out, ".build", "dev", appName)
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	w := prefixed("go", os.Stdout)
	for ctx.Err() == nil {
		stamp := goSourcesStamp(c.root)
		if err := run(ctx, w, c.root, nil, "go", "build", "-o", bin, pkgMain); err != nil {
			fmt.Fprintln(w, "build failed; waiting for changes…")
			waitForChange(ctx, c.root, stamp)
			continue
		}
		runCtx, cancel := context.WithCancel(ctx)
		done := make(chan struct{})
		go func() {
			_ = run(runCtx, w, c.root, nil, bin, args...)
			close(done)
		}()
		select {
		case <-done:
			cancel()
			if ctx.Err() == nil {
				waitForChange(ctx, c.root, stamp) // app exited (e.g. window closed): restart on the next change
			}
		case <-changed(ctx, c.root, stamp):
			fmt.Fprintln(w, "Go sources changed, restarting…")
			cancel()
			<-done
		}
	}
	return nil
}

func goSourcesStamp(root string) string {
	var b strings.Builder
	for _, dir := range []string{"cmd", "internal"} {
		_ = filepath.WalkDir(filepath.Join(root, dir), func(p string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() && strings.HasSuffix(p, ".go") {
				if info, err := d.Info(); err == nil {
					fmt.Fprintf(&b, "%s:%d:%d;", p, info.Size(), info.ModTime().UnixNano())
				}
			}
			return nil
		})
	}
	return b.String()
}

func changed(ctx context.Context, root, stamp string) <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		waitForChange(ctx, root, stamp)
		close(ch)
	}()
	return ch
}

func waitForChange(ctx context.Context, root, stamp string) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(700 * time.Millisecond):
			if goSourcesStamp(root) != stamp {
				return
			}
		}
	}
}

func openURL(url string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}

// ---------------------------------------------------------------- plumbing

type step struct {
	name string
	fn   func(ctx context.Context, w io.Writer) error
}

// parallel runs steps concurrently with prefixed output; the first failure
// cancels the others.
func parallel(ctx context.Context, steps ...step) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var wg sync.WaitGroup
	var once sync.Once
	var first error
	for _, s := range steps {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := prefixed(s.name, os.Stdout)
			t := time.Now()
			if err := s.fn(ctx, w); err != nil {
				once.Do(func() { first = fmt.Errorf("%s: %w", s.name, err); cancel() })
				return
			}
			fmt.Fprintf(w, "ok (%s)\n", time.Since(t).Round(100*time.Millisecond))
		}()
	}
	wg.Wait()
	return first
}

var outMu sync.Mutex

type prefixWriter struct {
	prefix string
	out    io.Writer
	buf    []byte
}

func prefixed(name string, out io.Writer) io.Writer {
	return &prefixWriter{prefix: fmt.Sprintf("[%s] ", name), out: out}
}

func (p *prefixWriter) Write(b []byte) (int, error) {
	outMu.Lock()
	defer outMu.Unlock()
	p.buf = append(p.buf, b...)
	for {
		i := strings.IndexByte(string(p.buf), '\n')
		if i < 0 {
			break
		}
		line := strings.TrimRight(string(p.buf[:i]), "\r")
		p.buf = p.buf[i+1:]
		if strings.TrimSpace(line) != "" {
			fmt.Fprintln(p.out, p.prefix+line)
		}
	}
	return len(b), nil
}

func run(ctx context.Context, w io.Writer, dir string, env []string, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdout, cmd.Stderr = w, w
	cmd.WaitDelay = 3 * time.Second
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}

func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "web", "package.json")); err == nil {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("run this from inside the dashcam-studio repository")
		}
		dir = parent
	}
}

func packageVersion(root string) string {
	data, err := os.ReadFile(filepath.Join(root, "web", "package.json"))
	if err != nil {
		return "dev"
	}
	var pkg struct {
		Version string `json:"version"`
	}
	if json.Unmarshal(data, &pkg) != nil || pkg.Version == "" {
		return "dev"
	}
	return pkg.Version
}

func rel(root, path string) string {
	if r, err := filepath.Rel(root, path); err == nil {
		return r
	}
	return path
}
