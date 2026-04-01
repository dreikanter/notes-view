package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"

	"github.com/dreikanter/notesview/internal/server"
)

func main() {
	port := flag.Int("port", 0, "port to listen on (default: auto)")
	flag.IntVar(port, "p", 0, "port to listen on (shorthand)")
	open := flag.Bool("open", true, "open browser on start")
	flag.BoolVar(open, "o", true, "open browser on start (shorthand)")
	editor := flag.String("editor", "", "editor command (overrides $EDITOR)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: notesview [options] [path]\n\n")
		fmt.Fprintf(os.Stderr, "Serve markdown files with live preview.\n\n")
		fmt.Fprintf(os.Stderr, "Path resolution: argument > $NOTES_PATH > current directory\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	root := resolveRoot(flag.Arg(0))

	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: %s is not a directory\n", root)
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
	url := "http://" + addr
	fmt.Printf("notesview serving %s at %s\n", root, url)

	if *open {
		openBrowser(url)
	}

	if err := http.Serve(listener, srv.Routes()); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func resolveRoot(arg string) string {
	if arg != "" {
		if arg[0] == '~' {
			home, _ := os.UserHomeDir()
			arg = home + arg[1:]
		}
		return arg
	}
	if p := os.Getenv("NOTES_PATH"); p != "" {
		if p[0] == '~' {
			home, _ := os.UserHomeDir()
			p = home + p[1:]
		}
		return p
	}
	dir, _ := os.Getwd()
	return dir
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
