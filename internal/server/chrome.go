package server

import (
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// indexQuery formats the canonical query suffix that preserves the
// panel's state across links. Empty string means closed. When open the
// panel path is always explicit — callers must resolve any default
// (e.g. the note's parent directory) before constructing the query —
// so the rendered URL is unambiguous and sticky navigation works.
func indexQuery(open bool, path string) string {
	if !open {
		return ""
	}
	return "?index=dir&path=" + url.QueryEscape(path)
}

// dirLinkHref builds an href that repositions the index panel to a new
// directory while preserving the current note (sticky model). notePath
// is the note that should stay visible, or "" for the standalone index
// page where there's no note to keep.
func dirLinkHref(notePath, dirPath string) string {
	q := "?index=dir&path=" + url.QueryEscape(dirPath)
	if notePath == "" {
		return "/" + q
	}
	return "/view/" + notePath + q
}

// fileLinkHref builds an href that changes the note while keeping the
// panel on the same directory. This is the other half of the sticky
// model: clicking a sibling file swaps the note card; the panel stays.
func fileLinkHref(filePath, panelPath string) string {
	return "/view/" + filePath + "?index=dir&path=" + url.QueryEscape(panelPath)
}

// buildBreadcrumbs constructs the panel's header trail. Intermediate
// segments link back up the directory chain via dirLinkHref so a click
// only repositions the panel — the note card is untouched. The final
// segment is marked Current (no link) since it's the directory the
// panel is already showing.
func buildBreadcrumbs(panelPath, notePath string) BreadcrumbsData {
	data := BreadcrumbsData{
		HomeHref: dirLinkHref(notePath, ""),
	}
	panelPath = strings.Trim(panelPath, "/")
	if panelPath == "" {
		return data
	}
	segments := strings.Split(panelPath, "/")
	accumulated := ""
	for i, seg := range segments {
		if accumulated == "" {
			accumulated = seg
		} else {
			accumulated += "/" + seg
		}
		if i == len(segments)-1 {
			data.Crumbs = append(data.Crumbs, Crumb{Label: seg, Current: true})
			continue
		}
		data.Crumbs = append(data.Crumbs, Crumb{
			Label: seg,
			Href:  dirLinkHref(notePath, accumulated),
		})
	}
	return data
}

// readDirEntries returns the visible entries of a notes directory as
// IndexEntry values. Directory entries link through dirLinkHref so the
// note stays put on click; file entries link through fileLinkHref so
// the panel stays put on click.
func readDirEntries(absPath, relPath, notePath string) ([]IndexEntry, error) {
	dirEntries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, err
	}
	entries := make([]IndexEntry, 0, len(dirEntries))
	for _, de := range dirEntries {
		name := de.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if !de.IsDir() && !strings.HasSuffix(name, ".md") {
			continue
		}
		entryRel := name
		if relPath != "" {
			entryRel = filepath.ToSlash(filepath.Join(relPath, name))
		}
		var href string
		if de.IsDir() {
			href = dirLinkHref(notePath, entryRel)
		} else {
			href = fileLinkHref(entryRel, relPath)
		}
		entries = append(entries, IndexEntry{
			Name:  name,
			IsDir: de.IsDir(),
			Href:  href,
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return entries[i].Name < entries[j].Name
	})
	return entries, nil
}
