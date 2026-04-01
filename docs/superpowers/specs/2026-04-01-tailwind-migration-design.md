# Tailwind CSS Migration

**Date:** 2026-04-01
**Status:** Approved

## Goal

Replace the 712-line hand-written `web/static/style.css` with a Tailwind-based stylesheet. Use `@tailwindcss/typography` (`prose`) for markdown content. All component styling expressed via `@apply` in a source CSS file — no inline utility classes in HTML or JS.

## File Changes

| File | Action |
|------|--------|
| `web/src/input.css` | New — Tailwind source with directives and all component definitions |
| `web/static/style.css` | Generated output — rebuilt by `make css`, committed so `go build` works without Tailwind |
| `tailwind.config.js` | New — content scanning, typography plugin |
| `web/static/index.html` | Minimal — add `prose max-w-none` class to markdown wrapper div (JS-generated) |
| `web/static/app.js` | Update markdown wrapper to include `prose max-w-none`; class names otherwise unchanged |
| `Makefile` | Add `css`, `css-watch`, update `all` target |

## CSS Architecture

`web/src/input.css`:

```css
@tailwind base;
@tailwind components;
@tailwind utilities;

@layer components {
  /* App chrome */
  /* Sidebar */
  /* Breadcrumbs */
  /* Frontmatter bar */
  /* Directory listing */
  /* Loading / error states */
  /* Go-renderer classes */
}
```

All component classes implemented with `@apply` using Tailwind's default design tokens (gray, blue, green, red scales). No custom CSS variables — Tailwind's palette replaces them.

## Component Classes

### App Chrome
- `#topbar` — fixed top bar, border-bottom, flex row
- `#sidebar-toggle` — ghost button
- `#edit-btn` — outlined button
- `#sidebar` — fixed left panel, border-right, slide transition
- `#sidebar.hidden` — `@apply -translate-x-full` (Tailwind's `hidden` utility kept for subtree nodes)
- `#content` — top margin offset, max-width centered, responsive padding

### Sidebar Tree
- `.sidebar-item` — flex row, hover background
- `.sidebar-item.active` — highlighted background, semibold
- `.sidebar-arrow` — small muted expand icon
- `.sidebar-label` — truncated text
- `.sidebar-link` — unstyled anchor, hover accent color
- `.sidebar-subtree` — indented nested list
- `.sidebar-subtree.hidden` — `display: none`

### Breadcrumbs
- `.breadcrumb` — accent link
- `.breadcrumb-sep` — muted separator
- `.breadcrumb-current` — default color, truncated

### Frontmatter Bar
- `.frontmatter-bar` — bottom border, bottom padding
- `.fm-title` — large semibold heading
- `.fm-description` — muted subtitle
- `.fm-tags` — flex wrap row
- `.fm-tag` — pill badge, accent background

### Directory Listing
- `.dir-listing` — bordered rounded container
- `.dir-title` — subtle background header
- `.dir-entries` — unstyled list
- `.entry` — border-bottom row
- `.entry a` — flex row with icon, hover background
- `.entry-dir a` — semibold

### States
- `.loading` — centered muted text
- `.error` — centered danger text
- `.error-page` — full error layout

### Go-Renderer Classes (cannot use inline utilities)
These classes are emitted by the Go renderer; defined with `@apply`:

- `.broken-link` — red strikethrough link
- `.uid-link` — accent monospace link
- `.task-checked` — green filled checkbox icon
- `.task-unchecked` — bordered empty checkbox
- `.task-tag` — subtle pill label
- `.chroma` — code block wrapper (font, background, border, overflow)
- `.chroma pre` — resets inside chroma wrapper

### Markdown Body
The `<div class="markdown-body">` wrapper in `app.js` gets `prose max-w-none` added. The typography plugin handles all rendered markdown element styling (headings, paragraphs, lists, tables, blockquotes, code). Custom overrides for `.broken-link`, `.uid-link`, etc. are scoped inside the prose context where needed.

## Tailwind Config

```js
module.exports = {
  content: [
    './web/static/index.html',
    './web/static/app.js',
  ],
  theme: { extend: {} },
  plugins: [require('@tailwindcss/typography')],
}
```

## Build Targets

```makefile
css:
    tailwindcss -i web/src/input.css -o web/static/style.css --minify

css-watch:
    tailwindcss -i web/src/input.css -o web/static/style.css --watch

all: css build
```

`make css` must be run after modifying `input.css` or any scanned content file. The generated `web/static/style.css` is committed so contributors can `go build` without installing Tailwind.

## Responsive Behaviour

Preserved from current CSS:
- Sidebar push on wide screens (≥900px): content left-margin adjusts when sidebar is open
- Mobile padding adjustments at 768px and 480px breakpoints — expressed with Tailwind responsive prefixes (`md:`, `sm:`) in `@apply` blocks

## What Does Not Change

- All class names used in `app.js` HTML strings remain the same
- `index.html` structure unchanged
- Server-side Go code unchanged
- All existing tests pass without modification
