# notesview

A local web server for browsing and previewing markdown notes with live reload.

## Features

- Renders markdown files with syntax highlighting
- Live reload via SSE when files change
- Opens files in your preferred editor
- Auto-detects GUI vs terminal editors
- Supports [Ghostty](https://ghostty.org/) terminal

## Installation

```sh
go install github.com/dreikanter/notesview/cmd/notesview@latest
```

## Usage

```sh
notesview [options] [path]
```

Path resolution order: CLI argument → `$NOTES_PATH` env var → current directory.

### Options

| Flag | Default | Description |
|------|---------|-------------|
| `--path` | `$NOTESVIEW_PATH` → `$NOTES_PATH` → `.` | Notes root directory or file to open |
| `--port`, `-p` | auto | Port to listen on |
| `--open`, `-o` | false | Open browser on start |
| `--editor` | `$NOTESVIEW_EDITOR` → `$VISUAL` → `$EDITOR` | Editor command |

If `--path` points to a file, the server root is set to the file's parent directory and the file is opened directly in the browser (when `--open` is set).

### Examples

```sh
notesview --path ~/notes           # serve a specific directory
notesview --path ~/notes/todo.md   # open a specific file, serve its directory
notesview -p 8080                  # use a fixed port
notesview --open                   # open browser automatically
notesview --editor=code            # use VS Code to open files
```

## Development

```sh
go test ./...
go build ./cmd/notesview
```
