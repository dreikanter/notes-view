// notesview-desktop is a Wails v2 desktop wrapper around the notesview
// HTTP server. It reuses the same internal/server package the CLI uses;
// instead of binding a TCP listener, the Wails asset server dispatches
// webview requests straight to srv.Routes().
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/dreikanter/notes-view/internal/logging"
	"github.com/dreikanter/notes-view/internal/server"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

// Version is stamped at link time via -ldflags "-X main.Version=...".
var Version = "dev"

type desktopApp struct {
	srv *server.Server
}

// onStartup starts the file watcher once the webview is up. Failing to
// start the watcher is non-fatal: the UI still works, we just lose
// live reload.
func (a *desktopApp) onStartup(_ context.Context) {
	if err := a.srv.StartWatcher(); err != nil {
		a.srv.Logger().Warn("file watcher failed to start", "err", err)
	}
}

// onShutdown drains the SSE hub and closes the fsnotify watcher so the
// process exits cleanly when the user closes the window.
func (a *desktopApp) onShutdown(_ context.Context) {
	a.srv.Shutdown()
}

func main() {
	if Version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "(devel)" {
			Version = info.Main.Version
		}
	}

	var (
		pathFlag    = flag.String("path", "", "notes root path or file (default: $NOTESVIEW_PATH, $NOTES_PATH, .)")
		editorFlag  = flag.String("editor", "", "editor command (default: $NOTESVIEW_EDITOR, $VISUAL, $EDITOR)")
		versionFlag = flag.Bool("version", false, "print version and exit")
	)
	flag.Parse()

	if *versionFlag {
		fmt.Println(Version)
		return
	}

	editor := *editorFlag
	if editor == "" {
		editor = firstEnv("NOTESVIEW_EDITOR", "VISUAL", "EDITOR")
	}

	path := *pathFlag
	if path == "" {
		path = firstEnv("NOTESVIEW_PATH", "NOTES_PATH")
	}
	if path == "" {
		path = "."
	}
	path = expandTilde(path)

	logger, closer, err := logging.New(logging.Config{
		Level:  os.Getenv("NOTESVIEW_LOG_LEVEL"),
		Format: os.Getenv("NOTESVIEW_LOG_FORMAT"),
		File:   os.Getenv("NOTESVIEW_LOG_FILE"),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if closer != nil {
		defer func() { _ = closer.Close() }()
	}

	root, err := resolveRoot(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	srv, err := server.NewServer(root, editor, logger)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	app := &desktopApp{srv: srv}

	logger.Info("notesview-desktop starting", "root", root, "version", Version)

	err = wails.Run(&options.App{
		Title:     "notesview",
		Width:     1280,
		Height:    860,
		MinWidth:  640,
		MinHeight: 400,
		AssetServer: &assetserver.Options{
			Handler: srv.Routes(),
		},
		OnStartup:  app.onStartup,
		OnShutdown: app.onShutdown,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func firstEnv(keys ...string) string {
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}

func expandTilde(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return p
		}
		return home + p[1:]
	}
	return p
}

// resolveRoot returns the notes root directory. If p points to a file, its
// parent directory is used. The initial-file shortcut that the CLI honours
// via --open is intentionally dropped here: the desktop shell always boots
// at the server root (which redirects to README.md when present).
func resolveRoot(p string) (string, error) {
	info, err := os.Stat(p)
	if err != nil {
		return "", err
	}
	dir := p
	if !info.IsDir() {
		dir = filepath.Dir(p)
	}
	return filepath.Abs(dir)
}
