package renderer

import (
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/dreikanter/notesview/internal/index"
)

var noteProtoRe = regexp.MustCompile(`href="note://(\d{8}_\d+)"`)
var relativeMdRe = regexp.MustCompile(`href="([^"]+\.md)"`)
var uidInTextRe = regexp.MustCompile(`\b(\d{8}_\d{4,})\b`)

// processNoteLinks rewrites three kinds of in-note links into /view/
// URLs: `note://<uid>` protocol links, relative `.md` links, and bare
// UIDs in text. linkQuery is appended verbatim to every generated
// href so the caller can thread the current index-panel state
// through; pass "" when there is no state to preserve.
func processNoteLinks(html string, idx *index.Index, currentDir, linkQuery string) string {
	// 1. Resolve note:// protocol links
	html = noteProtoRe.ReplaceAllStringFunc(html, func(match string) string {
		sub := noteProtoRe.FindStringSubmatch(match)
		if sub == nil {
			return match
		}
		uid := sub[1]
		if relPath, ok := idx.Lookup(uid); ok {
			return fmt.Sprintf(`href="/view/%s%s"`, relPath, linkQuery)
		}
		return fmt.Sprintf(`href="#" class="broken-link" title="Note %s not found"`, uid)
	})

	// 2. Rewrite relative .md links to /view/ routes
	html = relativeMdRe.ReplaceAllStringFunc(html, func(match string) string {
		sub := relativeMdRe.FindStringSubmatch(match)
		if sub == nil {
			return match
		}
		relLink := sub[1]
		// Skip absolute URLs (http://, https://, etc.) and already-rooted paths
		if strings.Contains(relLink, "://") || strings.HasPrefix(relLink, "/") {
			return match
		}
		resolved := path.Clean(path.Join(currentDir, relLink))
		resolved = strings.TrimPrefix(resolved, "/")
		return fmt.Sprintf(`href="/view/%s%s"`, resolved, linkQuery)
	})

	// 3. Auto-link bare UIDs in text content (not inside HTML tags)
	parts := splitByTags(html)
	for i, part := range parts {
		if !strings.HasPrefix(part, "<") {
			parts[i] = uidInTextRe.ReplaceAllStringFunc(part, func(match string) string {
				if relPath, ok := idx.Lookup(match); ok {
					return fmt.Sprintf(`<a href="/view/%s%s" class="uid-link">%s</a>`, relPath, linkQuery, match)
				}
				return match
			})
		}
	}
	return strings.Join(parts, "")
}

func splitByTags(html string) []string {
	var parts []string
	for len(html) > 0 {
		tagStart := strings.Index(html, "<")
		if tagStart == -1 {
			parts = append(parts, html)
			break
		}
		if tagStart > 0 {
			parts = append(parts, html[:tagStart])
		}
		tagEnd := strings.Index(html[tagStart:], ">")
		if tagEnd == -1 {
			parts = append(parts, html[tagStart:])
			break
		}
		parts = append(parts, html[tagStart:tagStart+tagEnd+1])
		html = html[tagStart+tagEnd+1:]
	}
	return parts
}
