package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

// indexState parses the panel state from the request's query string.
// Only the "dir" mode is understood today; unknown values collapse to
// closed. The returned explicitPath is the raw ?path= value (empty
// string is valid and means "root"); hasPath distinguishes "no path
// param" from "path param set to empty" so the caller can pick a
// sensible default for the former (e.g. the note's parent directory).
//
// This function is the seam future index modes extend: adding
// "search" or "tag" means returning a mode discriminator plus the
// mode-specific extras the query encodes.
func indexState(r *http.Request) (open bool, explicitPath string, hasPath bool) {
	q := r.URL.Query()
	if q.Get("index") != "dir" {
		return false, "", false
	}
	raw, ok := q["path"]
	if !ok {
		return true, "", false
	}
	return true, strings.Trim(raw[0], "/"), true
}

// toggleHref returns the URL the hamburger should point at. When the
// panel is open, the link strips the query and leaves just the note
// path so a click closes the panel. When closed, the link adds the
// panel with an explicit default path (the note's parent) so a click
// opens the panel to the right directory.
func toggleHref(notePath string, open bool, defaultPath string) string {
	if open {
		return "/view/" + notePath
	}
	return "/view/" + notePath + "?index=dir&path=" + url.QueryEscape(defaultPath)
}

// buildLayoutFields assembles the common chrome every page needs. The
// effectivePath is the directory the panel is showing — already
// resolved from either ?path= or a handler-specific default — so the
// returned IndexQuery always renders with an explicit path and sticky
// navigation works regardless of how the panel was opened.
func (s *Server) buildLayoutFields(title, editPath string, open bool, effectivePath string) layoutFields {
	lf := layoutFields{
		Title:      title,
		EditPath:   editPath,
		IndexOpen:  open,
		IndexQuery: indexQuery(open, effectivePath),
		ShowToggle: true,
	}
	if editPath != "" {
		lf.EditHref = "/api/edit/" + editPath
	}
	return lf
}

// viewSSEWatch is the value for the sse-connect attribute on view.html.
// The SSE URL needs the note path percent-encoded because file names may
// contain spaces, slashes, question marks, etc.
func viewSSEWatch(filePath string) string {
	return "/events?watch=" + url.QueryEscape(filePath)
}

// handleRoot is the entry point for both the legacy redirect behavior
// and the standalone index panel (the replacement for /browse/). When
// the request has no query params it preserves the old "redirect to
// README if present, else show the root index" behavior. When
// ?index=dir is set, it renders the standalone panel at ?path=.
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if r.URL.Query().Get("index") == "dir" {
		s.handleStandaloneIndex(w, r)
		return
	}
	readme := filepath.Join(s.root, "README.md")
	if _, err := os.Stat(readme); err == nil {
		http.Redirect(w, r, "/view/README.md", http.StatusFound)
		return
	}
	s.handleStandaloneIndex(w, r)
}

// handleStandaloneIndex renders the index panel as a page of its own,
// with no note card. This is the Option B replacement for /browse/ —
// the directory the panel shows comes from ?path= rather than from the
// URL path component.
func (s *Server) handleStandaloneIndex(w http.ResponseWriter, r *http.Request) {
	panelPath := strings.Trim(r.URL.Query().Get("path"), "/")
	absPath, err := SafePath(s.root, panelPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	stat, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !stat.IsDir() {
		http.Error(w, "not a directory", http.StatusBadRequest)
		return
	}

	lf := s.buildLayoutFields(dirTitle(panelPath), "", true, panelPath)
	// No note is in context on the standalone page, so the hamburger
	// has nothing to reveal/hide — hide it rather than render a
	// link that would take the user to an empty screen.
	lf.ShowToggle = false

	card, err := s.buildDirIndex(panelPath, "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	lf.IndexCard = card

	go s.index.Build()

	browse := BrowseData{
		layoutFields: lf,
		DirPath:      panelPath,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.renderBrowse(w, browse); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleView(w http.ResponseWriter, r *http.Request) {
	reqPath := r.PathValue("filepath")
	absPath, err := SafePath(s.root, reqPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	currentDir := filepath.Dir(reqPath)
	if currentDir == "." {
		currentDir = ""
	}
	html, fm, err := s.renderer.Render(data, currentDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	title := filepath.Base(reqPath)
	if fm != nil && fm.Title != "" {
		title = fm.Title
	}

	// Resolve the panel's effective directory. When the panel is open
	// and ?path= is absent we default to the note's own parent — this
	// is the only case where the default matters because every link
	// the server itself renders emits ?path= explicitly.
	open, explicitPath, hasPath := indexState(r)
	panelPath := currentDir
	if open && hasPath {
		panelPath = explicitPath
	}

	lf := s.buildLayoutFields(title, reqPath, open, panelPath)
	lf.ToggleHref = toggleHref(reqPath, open, currentDir)

	if open {
		// A read failure (invalid ?path=, permissions, vanished dir)
		// shouldn't 500 the note view — log and render without a card
		// so the user still sees the note.
		card, err := s.buildDirIndex(panelPath, reqPath)
		if err != nil {
			s.logger.Warn("index card build failed", "path", panelPath, "err", err)
		} else {
			lf.IndexCard = card
		}
	}

	view := ViewData{
		layoutFields: lf,
		FilePath:     reqPath,
		Frontmatter:  fm,
		HTML:         template.HTML(html),
		SSEWatch:     viewSSEWatch(reqPath),
		ViewHref:     "/view/" + reqPath + lf.IndexQuery,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.renderView(w, view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleBrowse is a permanent redirect to the canonical
// /?index=dir&path= form. The positional /browse/{dir} route is
// retained purely as a compatibility shim for external bookmarks; the
// server itself never generates /browse/ links anymore.
func (s *Server) handleBrowse(w http.ResponseWriter, r *http.Request) {
	reqPath := strings.Trim(r.PathValue("dirpath"), "/")
	target := "/?index=dir&path=" + url.QueryEscape(reqPath)
	http.Redirect(w, r, target, http.StatusMovedPermanently)
}

// buildDirIndex assembles an IndexCard in directory mode for a path
// relative to the notes root. notePath is the note currently in view
// (if any) — directory links in the resulting card will target that
// note with an updated ?path= so the note stays visible when the user
// navigates the panel. Pass "" for the standalone page.
func (s *Server) buildDirIndex(panelPath, notePath string) (*IndexCard, error) {
	absPath, err := SafePath(s.root, panelPath)
	if err != nil {
		return nil, err
	}
	entries, err := readDirEntries(absPath, panelPath, notePath)
	if err != nil {
		return nil, err
	}
	return &IndexCard{
		Mode:        "dir",
		Breadcrumbs: buildBreadcrumbs(panelPath, notePath),
		Entries:     entries,
		Empty:       "No files here.",
	}, nil
}

func dirTitle(reqPath string) string {
	if reqPath == "" {
		return ""
	}
	return reqPath
}

func (s *Server) handleRaw(w http.ResponseWriter, r *http.Request) {
	reqPath := r.PathValue("filepath")
	absPath, err := SafePath(s.root, reqPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(data)
}

// terminalEditors are editors that need an interactive terminal to run.
var terminalEditors = map[string]bool{
	"vim": true, "nvim": true, "vi": true, "nano": true,
	"emacs": true, "micro": true, "helix": true, "hx": true,
	"joe": true, "ne": true, "mcedit": true, "ed": true,
}

func (s *Server) handleEdit(w http.ResponseWriter, r *http.Request) {
	reqPath := r.PathValue("filepath")
	absPath, err := SafePath(s.root, reqPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if s.editor == "" {
		http.Error(w, "no editor configured (set NOTESVIEW_EDITOR, VISUAL, or EDITOR)", http.StatusBadRequest)
		return
	}

	// Parse the editor env var the way shells treat $EDITOR: the first
	// token is the binary, the rest are leading arguments (e.g.
	// `code --wait`, `subl -w`, `nvim -R`). Without this split, exec
	// looks for a literal binary named `"code --wait"` and 500s. A
	// whitespace-only value slips past the `== ""` guard above but
	// yields zero fields, so recheck after Fields to avoid indexing a
	// nil slice and panicking the handler.
	fields := strings.Fields(s.editor)
	if len(fields) == 0 {
		http.Error(w, "no editor configured (set NOTESVIEW_EDITOR, VISUAL, or EDITOR)", http.StatusBadRequest)
		return
	}
	editorBin, editorArgs := fields[0], fields[1:]
	args := append(append([]string{}, editorArgs...), absPath)

	if err := openEditor(editorBin, args); err != nil {
		http.Error(w, fmt.Sprintf("failed to start editor: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// openEditor launches the editor for the given file. GUI editors are spawned
// directly. Terminal editors are opened in a new terminal window.
func openEditor(editorBin string, args []string) error {
	if terminalEditors[filepath.Base(editorBin)] {
		return openInTerminal(editorBin, args)
	}
	cmd := exec.Command(editorBin, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd.Start()
}

// openInTerminal opens a terminal editor in a new terminal window.
func openInTerminal(editorBin string, args []string) error {
	switch runtime.GOOS {
	case "darwin":
		return openInTerminalDarwin(editorBin, args)
	case "linux":
		return openInTerminalLinux(editorBin, args)
	default:
		// Fallback: try to run directly (will likely fail for TUI editors
		// but there's no portable way to open a terminal on this OS)
		cmd := exec.Command(editorBin, args...)
		return cmd.Start()
	}
}

func openInTerminalDarwin(editorBin string, args []string) error {
	// Prefer Ghostty via its bundled binary. Launching via
	// `open -na Ghostty.app --args …` is unreliable for .app bundles
	// because AppKit doesn't always forward `--args` to the inner
	// executable, so we invoke the binary directly when it's present.
	ghosttyBin := "/Applications/Ghostty.app/Contents/MacOS/ghostty"
	if _, err := os.Stat(ghosttyBin); err == nil {
		ghosttyArgs := append([]string{"-e", editorBin}, args...)
		cmd := exec.Command(ghosttyBin, ghosttyArgs...)
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		return cmd.Start()
	}

	// Fall back to AppleScript: prefer iTerm2 if running, else Terminal.app.
	// Both execute a single shell command string, so we POSIX-quote every
	// argument and concatenate.
	shellCmd := shellJoin(append([]string{editorBin}, args...))
	scriptCmd := appleEscape(shellCmd)
	script := fmt.Sprintf(`
		tell application "System Events"
			set iterm_running to (name of processes) contains "iTerm2"
		end tell
		if iterm_running then
			tell application "iTerm2"
				activate
				tell current window
					create tab with default profile
					tell current session
						write text "%s"
					end tell
				end tell
			end tell
		else
			tell application "Terminal"
				activate
				do script "%s"
			end tell
		end if
	`, scriptCmd, scriptCmd)
	cmd := exec.Command("osascript", "-e", script)
	return cmd.Start()
}

func openInTerminalLinux(editorBin string, args []string) error {
	// Per-terminal invocation style. Most accept `-e cmd [args…]` with
	// separate argv tokens, but xfce4-terminal's `-e` expects a single
	// shell-string, so we POSIX-quote into one arg for it.
	editorArgv := append([]string{editorBin}, args...)
	shellCmd := shellJoin(editorArgv)
	terminals := []struct {
		cmd  string
		args []string
	}{
		{"ghostty", append([]string{"-e"}, editorArgv...)},
		{"x-terminal-emulator", append([]string{"-e"}, editorArgv...)},
		{"gnome-terminal", append([]string{"--"}, editorArgv...)},
		{"konsole", append([]string{"-e"}, editorArgv...)},
		{"xfce4-terminal", []string{"-e", shellCmd}},
		{"xterm", append([]string{"-e"}, editorArgv...)},
	}
	for _, t := range terminals {
		if path, err := exec.LookPath(t.cmd); err == nil {
			cmd := exec.Command(path, t.args...)
			cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			return cmd.Start()
		}
	}
	return fmt.Errorf("no terminal emulator found; install one or use a GUI editor")
}

// shellJoin POSIX-quotes each arg and joins with spaces, producing a string
// safe to pass to `sh -c` or a terminal that expects a single command line.
func shellJoin(argv []string) string {
	quoted := make([]string, len(argv))
	for i, a := range argv {
		quoted[i] = shellQuote(a)
	}
	return strings.Join(quoted, " ")
}

// shellQuote wraps s in single quotes, escaping any embedded single quotes
// via the classic `'\''` dance. This is POSIX-safe for any string.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// appleEscape escapes a string for inclusion inside an AppleScript
// double-quoted string literal.
func appleEscape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}
