package renderer

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// findAnchor tokenizes rendered HTML and returns the first <a> tag whose
// attribute attrKey equals attrVal. It fails the test if no match is found.
func findAnchor(t *testing.T, rendered, attrKey, attrVal string) html.Token {
	t.Helper()
	z := html.NewTokenizer(strings.NewReader(rendered))
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			break
		}
		if tt != html.StartTagToken {
			continue
		}
		tok := z.Token()
		if tok.DataAtom == 0 && tok.Data != "a" || tok.DataAtom != 0 && tok.Data != "a" {
			continue
		}
		for _, a := range tok.Attr {
			if a.Key == attrKey && a.Val == attrVal {
				return tok
			}
		}
	}
	t.Fatalf("no <a %s=%q> found in:\n%s", attrKey, attrVal, rendered)
	return html.Token{} // unreachable
}

// assertAttr asserts that tok carries attribute key with the given value.
func assertAttr(t *testing.T, tok html.Token, key, val string) {
	t.Helper()
	for _, a := range tok.Attr {
		if a.Key == key {
			if a.Val != val {
				t.Errorf("attr %s=%q, want %q (tag: %s)", key, a.Val, val, tok)
			}
			return
		}
	}
	t.Errorf("attr %q not found on tag: %s", key, tok)
}

// assertNoAttr asserts that tok does NOT carry the named attribute.
func assertNoAttr(t *testing.T, tok html.Token, key string) {
	t.Helper()
	for _, a := range tok.Attr {
		if a.Key == key {
			t.Errorf("attr %q should be absent, got %s=%q (tag: %s)", key, key, a.Val, tok)
			return
		}
	}
}
