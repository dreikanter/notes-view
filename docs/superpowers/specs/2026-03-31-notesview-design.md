# notesview — Local Markdown Viewer

A single Go binary that serves a local HTTP server to browse and render markdown files with GitHub-style presentation, live-reload, and a collapsible file browser.

## CLI

```
notesview [options] [path]
```

**Path resolution:** CLI argument > `$NOTES_PATH` > current working directory.

**Flags:**
- `--port` / `-p` — port number (default: auto-pick available port)
- `--open` / `-o` — open browser on start (default: true)
- `--editor` — override `$EDITOR` for the Edit button

Prints URL on startup. Opens browser automatically unless `--open=false`.

## Architecture

Single binary, all frontend assets embedded via Go's `embed.FS`. No external build step.

### Components

1. **HTTP server** (`net/http`) — serves SPA shell, API endpoints, embedded static assets
2. **Markdown renderer** (`goldmark` + extensions) — GFM-compatible markdown to HTML
3. **File watcher** (`fsnotify`) — watches currently viewed file, notifies via SSE
4. **Link resolver** — custom goldmark extension for note-specific link formats

### Dependencies

- `github.com/yuin/goldmark` — markdown parsing and rendering
- `github.com/yuin/goldmark-meta` — YAML frontmatter extraction
- `github.com/alecthomas/chroma/v2` — syntax highlighting for code blocks
- `github.com/fsnotify/fsnotify` — filesystem event notifications

## URL Scheme

| Route | Purpose |
|---|---|
| `/` | Redirects to README.md if present, otherwise shows file browser |
| `/view/{filepath...}` | Renders a markdown file |
| `/browse/{dirpath...}` | JSON directory listing |
| `/api/edit/{filepath...}` | Spawns `$EDITOR` on the file in the server's terminal |
| `/api/raw/{filepath...}` | Serves raw markdown content |
| `/events` | SSE endpoint for live-reload |

All file paths are relative to the root directory. Path traversal outside root is rejected.

## Frontend

Single-page app with client-side navigation. All HTML, CSS, and JS embedded in the binary.

### Layout (Collapsible Sidebar + Breadcrumbs)

- **Top bar:** hamburger toggle (left), breadcrumb path with clickable segments (center-left), Edit button (right)
- **Sidebar:** hidden by default, slides in from left on toggle. Contains a lazy-loaded directory tree.
- **Content area:** centered, max-width ~900px, GitHub-style rendered markdown

### Navigation

- Clicking breadcrumb segments navigates to that directory's listing
- Clicking sidebar files/folders navigates without full page reload (fetch + history.pushState)
- Browser back/forward works via popstate handling

### Visual Style

GitHub-flavored rendering:
- System font stack (`-apple-system, BlinkMacSystemFont, "Segoe UI", ...`)
- Content area with comfortable reading width
- Code blocks with syntax highlighting and subtle background
- Tables with borders and alternating row shading
- Blockquotes with left border accent

### YAML Frontmatter Display

Frontmatter is parsed and rendered as a metadata bar above the content:
- `title` displayed as the page heading (replaces first `# heading` if redundant)
- `tags` rendered as colored pills/badges
- `description` shown as subtitle text below the title
- `slug` not displayed (internal identifier)
- Raw YAML is never shown

### Task Syntax Rendering

- `- [+]` rendered as a checked checkbox (green checkmark)
- `- [ ]` rendered as an unchecked checkbox
- `- [daily]` rendered with a "daily" badge/tag next to the checkbox

These are display-only; checkboxes are not interactive.

## Markdown Rendering

Based on `goldmark` with these extensions:

### Standard GFM
- Tables, strikethrough, autolinks, task lists
- Fenced code blocks with syntax highlighting via `chroma`

### Custom Extensions

**note:// links:** `[link text](note://20260331_9201)` resolves to the file matching UID `20260331_9201` by scanning the directory tree. Rendered as `/view/YYYY/MM/YYYYMMDD_ID_slug.md`. Broken links rendered with a visual indicator (strikethrough or red).

**Auto-linked UIDs:** Bare `YYYYMMDD_NNNN` patterns (8-digit date + underscore + numeric ID) in text are auto-linked if a matching file exists in the root directory tree. Non-matching patterns are left as plain text.

**Relative .md links:** Standard markdown links to `.md` files (e.g., `[text](../other.md)`) are rewritten to `/view/...` routes so they navigate within the SPA.

### UID Resolution

On startup, build an in-memory index mapping UIDs to file paths by scanning filenames matching `YYYYMMDD_NNNN*.md`. Refresh the index when the sidebar directory listing is fetched (piggyback on existing I/O). This index is used by both `note://` links and auto-linking.

## Live Reload

1. Browser opens SSE connection to `/events`
2. Server tracks which file each SSE client is viewing
3. Server watches that file with `fsnotify`
4. On file change, server sends SSE event with the file path
5. Browser fetches `/api/raw/{path}` (or a render endpoint), replaces content area HTML
6. When user navigates to a different file, browser sends the new path; server updates the watch target
7. Debounce file change events (100ms) to avoid rapid-fire updates during saves

## File Browser Sidebar

- **Lazy-loaded:** root directory listed on first open; subdirectories fetched on expand
- **Sort order:** directories first, then files, both alphabetical
- **Filter:** shows `.md` files and directories only
- **Highlight:** current file visually highlighted
- **Click behavior:** files navigate to `/view/...`; directories expand/collapse in-place

## Security

- All file paths validated to stay within the configured root directory
- `/api/edit` only available on localhost (server binds to `127.0.0.1` by default)
- No authentication (local-only tool)

## Error Handling

- Missing files: 404 page with breadcrumb navigation still functional
- Broken `note://` links: rendered with visual indicator, not silently swallowed
- `$EDITOR` not set: Edit button shows a tooltip/message explaining the issue
- Port in use: auto-increment to next available port
