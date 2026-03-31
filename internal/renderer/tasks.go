package renderer

import (
	"regexp"
	"strings"
)

// GFM converts [ ] and [x] to checkbox inputs; we replace those rendered forms.
// [+] is not a GFM task marker so it passes through as literal text.
var taskPatterns = []struct {
	marker  string
	replace string
}{
	// GFM unchecked checkbox → task-unchecked span
	{`<input disabled="" type="checkbox"> `, `<span class="task-unchecked"></span> `},
	// GFM checked checkbox → task-checked span (fallback, in case [x] is used)
	{`<input checked="" disabled="" type="checkbox"> `, `<span class="task-checked">&#10003;</span> `},
	// [+] passes through goldmark as literal text
	{"[+] ", `<span class="task-checked">&#10003;</span> `},
}

var dailyPattern = regexp.MustCompile(`\[daily\]\s*`)

func processTaskSyntax(html string) string {
	for _, p := range taskPatterns {
		html = strings.ReplaceAll(html, p.marker, p.replace)
	}
	html = dailyPattern.ReplaceAllString(html, `<span class="task-tag">daily</span> `)
	return html
}
