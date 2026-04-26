//go:build integration

package main_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dreikanter/nview/internal/server"
)

func TestIntegrationSmoke(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "2026", "03"), 0o755)
	os.MkdirAll(filepath.Join(dir, "2026", "01"), 0o755)
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Welcome\n\nHello world.\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "2026", "01", "20260101_1_readme.md"), []byte("# Welcome\n\nHello world.\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "2026", "03", "20260331_9201_todo.todo.md"),
		[]byte("---\ntitle: Daily Todo\ntype: todo\ntags: [todo]\n---\n# Daily Todo\n\n- [x] Done task\n- [ ] Pending task\n- [daily] Routine\n\nSee [readme](../../README.md) and note://9201.\n"), 0o644)

	srv, err := server.NewServer(dir, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv.Routes())
	defer ts.Close()

	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	rootResp, err := client.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	rootResp.Body.Close()
	if rootResp.StatusCode != http.StatusFound {
		t.Errorf("root: status = %d, want 302", rootResp.StatusCode)
	}

	viewResp, err := http.Get(ts.URL + "/n/9201")
	if err != nil {
		t.Fatal(err)
	}
	defer viewResp.Body.Close()
	if viewResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(viewResp.Body)
		t.Fatalf("view: status = %d, body: %s", viewResp.StatusCode, body)
	}
	if ct := viewResp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("view: content-type = %q, want text/html", ct)
	}
	body, _ := io.ReadAll(viewResp.Body)
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "Daily Todo") {
		t.Errorf("view: expected title 'Daily Todo' in body")
	}
	if !strings.Contains(bodyStr, `class="markdown-body`) {
		t.Errorf("view: expected markdown-body wrapper in HTML")
	}

	dirResp, err := http.Get(ts.URL + "/n/1")
	if err != nil {
		t.Fatal(err)
	}
	viewBody, err := io.ReadAll(dirResp.Body)
	dirResp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if dirResp.StatusCode != http.StatusOK {
		t.Errorf("/n/1: status = %d, body: %s", dirResp.StatusCode, viewBody)
	}
	if !strings.Contains(string(viewBody), `id="sidebar"`) {
		t.Errorf("expected #sidebar in response")
	}
	if !strings.Contains(string(viewBody), `id="recent-section"`) || !strings.Contains(string(viewBody), `href="/types"`) || !strings.Contains(string(viewBody), `href="/dates"`) {
		t.Errorf("expected metadata sidebar sections in response")
	}
	if !strings.Contains(string(viewBody), `"selectedPath":"2026/01/20260101_1_readme.md"`) {
		t.Errorf("expected selectedPath=readme rel-path in initial JSON")
	}

	listResp, err := http.Get(ts.URL + "/api/tree/list?path=")
	if err != nil {
		t.Fatal(err)
	}
	listBody, _ := io.ReadAll(listResp.Body)
	listResp.Body.Close()
	if !strings.Contains(string(listBody), `"path":"2026"`) || !strings.Contains(string(listBody), `"isDir":true`) {
		t.Errorf("expected 2026 dir entry in tree list, got: %s", listBody)
	}

	rawResp, err := http.Get(ts.URL + "/api/raw/1")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(rawResp.Body)
	rawResp.Body.Close()
	if string(raw) != "# Welcome\n\nHello world.\n" {
		t.Errorf("raw = %q", raw)
	}

	notFoundResp, err := http.Get(ts.URL + "/n/nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	notFoundResp.Body.Close()
	if notFoundResp.StatusCode != http.StatusNotFound {
		t.Errorf("404: status = %d", notFoundResp.StatusCode)
	}
}
