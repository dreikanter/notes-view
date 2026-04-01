package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/dreikanter/notesview/internal/server"
)

func main() {
	port := flag.Int("port", 0, "port to listen on (default: auto)")
	flag.IntVar(port, "p", 0, "shorthand for --port")
	open := flag.Bool("open", false, "open browser on start")
	flag.BoolVar(open, "o", false, "shorthand for --open")
	editor := flag.String("editor", "", "editor command (default: $NOTESVIEW_EDITOR, $VISUAL, $EDITOR)")
	path := flag.String("path", "", "notes root path or file (default: $NOTESVIEW_PATH, $NOTES_PATH, .)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: notesview [options]\n\n")
		fmt.Fprintf(os.Stderr, "Serve markdown files with live preview.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  --port, -p int    port to listen on (default: auto)\n")
		fmt.Fprintf(os.Stderr, "  --open, -o        open browser on start\n")
		fmt.Fprintf(os.Stderr, "  --editor string   editor command (default: $NOTESVIEW_EDITOR, $VISUAL, $EDITOR)\n")
		fmt.Fprintf(os.Stderr, "  --path string     notes root path or file (default: $NOTESVIEW_PATH, $NOTES_PATH, .)\n")
	}
	flag.Parse()

	if *editor == "" {
		for _, env := range []string{"NOTESVIEW_EDITOR", "VISUAL", "EDITOR"} {
			if v := os.Getenv(env); v != "" {
				*editor = v
				break
			}
		}
	}

	if *path == "" {
		for _, env := range []string{"NOTESVIEW_PATH", "NOTES_PATH"} {
			if v := os.Getenv(env); v != "" {
				*path = v
				break
			}
		}
	}
	if *path == "" {
		*path = "."
	}
	*path = expandTilde(*path)

	root, initialFile, err := resolvePath(*path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	srv := server.NewServer(root, *editor)
	if err := srv.StartWatcher(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: file watcher failed to start: %v\n", err)
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *port))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	addr := listener.Addr().String()
	baseURL := "http://" + addr
	fmt.Printf("notesview serving %s at %s\n", root, baseURL)

	if *open {
		target := baseURL
		if initialFile != "" {
			target = baseURL + "/view/" + initialFile
		}
		openBrowser(target)
	}

	if err := http.Serve(listener, srv.Routes()); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// resolvePath returns the root directory and, if p points to a file,
// the relative file name within that directory.
func resolvePath(p string) (root, initialFile string, err error) {
	info, err := os.Stat(p)
	if err != nil {
		return "", "", err
	}
	if info.IsDir() {
		return p, "", nil
	}
	return filepath.Dir(p), filepath.Base(p), nil
}

func expandTilde(p string) string {
	if len(p) > 0 && p[0] == '~' {
		home, _ := os.UserHomeDir()
		return home + p[1:]
	}
	return p
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return
	}
	cmd.Start()
}
