package index

import (
	"bufio"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

// frontmatter is the typed per-file frontmatter. Fields not carried by a
// given file default to Go zero values. The struct is intentionally
// private: extending it with new fields is a local change.
type frontmatter struct {
	Title   string    `yaml:"title"`
	Tags    []string  `yaml:"tags"`
	Aliases []string  `yaml:"aliases"`
	Date    time.Time `yaml:"date"`
}

// parseFrontmatter reads the file at path, extracts the YAML block between
// the first two `---` fences on their own lines, and unmarshals it. Missing
// fences yield a zero-valued frontmatter and no error. Read errors and
// malformed YAML are returned.
func parseFrontmatter(path string) (frontmatter, error) {
	var fm frontmatter

	f, err := os.Open(path)
	if err != nil {
		return fm, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	// Allow very long frontmatter lines (defensive against defaults).
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	// First non-empty line must be `---`.
	if !scanner.Scan() {
		return fm, scanner.Err()
	}
	if strings.TrimSpace(scanner.Text()) != "---" {
		return fm, nil
	}

	var body strings.Builder
	closed := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			closed = true
			break
		}
		body.WriteString(line)
		body.WriteByte('\n')
	}
	if err := scanner.Err(); err != nil {
		return fm, err
	}
	if !closed {
		// No closing fence — treat as no frontmatter.
		return frontmatter{}, nil
	}

	if body.Len() == 0 {
		return fm, nil
	}

	if err := yaml.Unmarshal([]byte(body.String()), &fm); err != nil {
		return frontmatter{}, err
	}
	return fm, nil
}
