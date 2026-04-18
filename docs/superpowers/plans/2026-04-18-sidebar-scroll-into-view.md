# Sidebar Scroll-Into-View Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** When a note is selected (from any source), scroll the sidebar so the highlighted entry is visible inside the sidebar's viewport.

**Architecture:** Extend the existing `markSelected` helper in `web/src/app.js` so that, after applying highlight classes to the matching sidebar entry, it also calls `scrollIntoView({ block: 'center', inline: 'nearest' })`. All other behavior (timing, re-swaps, element lookup) stays the same. No server, template, CSS, or test changes.

**Tech Stack:** Vanilla JS + HTMX, Vite build, Playwright E2E (regression only).

**Spec:** `docs/superpowers/specs/2026-04-18-sidebar-scroll-into-view-design.md`

---

## File Map

| File | Action | Responsibility |
|------|--------|----------------|
| `web/src/app.js` | Modify (function `markSelected`, lines 99-103) | Apply highlight classes AND scroll the selected entry into the sidebar's viewport. |

No other files are touched.

---

### Task 1: Add `scrollIntoView` call inside `markSelected`

**Files:**
- Modify: `web/src/app.js:99-103`

- [ ] **Step 1: Read the current `markSelected` function**

Confirm the current shape before editing. Run:

```
Read web/src/app.js lines 99-103
```

Expected existing content (exactly):

```js
function markSelected(selector) {
  clearSelected();
  var el = document.querySelector(selector);
  if (el) el.classList.add('selected', 'bg-blue-100', 'border-blue-300', 'text-blue-700');
}
```

If the file does not match exactly (e.g. someone else refactored `markSelected`), STOP and re-check the spec/plan before editing.

- [ ] **Step 2: Edit `markSelected` to also scroll the selected element into view**

Replace the single-line `if (el)` body with a block that adds the highlight classes AND scrolls the element into the nearest scrollable ancestor's viewport.

Use Edit on `web/src/app.js`:

`old_string`:
```js
function markSelected(selector) {
  clearSelected();
  var el = document.querySelector(selector);
  if (el) el.classList.add('selected', 'bg-blue-100', 'border-blue-300', 'text-blue-700');
}
```

`new_string`:
```js
function markSelected(selector) {
  clearSelected();
  var el = document.querySelector(selector);
  if (el) {
    el.classList.add('selected', 'bg-blue-100', 'border-blue-300', 'text-blue-700');
    el.scrollIntoView({ block: 'center', inline: 'nearest' });
  }
}
```

Why `block: 'center'`:
- Centers the entry in the sidebar's scroll viewport, surfacing context above and below — better readability than `'nearest'` on long lists.
- Once an element is centered, the browser's computed scroll delta for a follow-up `scrollIntoView({ block: 'center' })` call is zero, so back-to-back calls during the two `htmx:afterSwap` events per navigation produce no visible movement.
- `inline: 'nearest'` prevents any accidental horizontal scroll in edge cases.

Why this seam:
- `markSelected` is the single funnel for every selection change (sidebar click, main-pane link, `popstate`, HTMX re-swap).
- The existing `htmx:afterSwap` listener (`web/src/app.js:266-276`) already defers `markSelected` via `setTimeout(..., 0)` so the new swapped DOM is in place — timing from issue #82's note is already covered.

- [ ] **Step 3: Build the front-end assets**

Run from the worktree root:

```
make assets
```

Expected: `npx vite build` completes without errors. The built `app.js` in `web/static/` is updated with the new call.

If the build fails: read the Vite error, fix the syntax in `web/src/app.js`, and re-run.

- [ ] **Step 4: Start the dev server and run the Go server manually against fixtures**

From the worktree root, build and run the server against the Playwright fixtures so you can click around in a real browser.

```
make build
./notesview --notes tests/fixtures/notes --addr 127.0.0.1:5173
```

Leave this running in one terminal; open http://127.0.0.1:5173 in a browser.

Manual verification steps:

1. Open the sidebar (click the sidebar toggle in the header).
2. Click the `journal` directory in the sidebar — confirm it expands to show `day-one.md` and `day-two.md`.
3. Click `day-two.md` from the main-pane directory listing.
4. **Expected:** the sidebar entry for `day-two.md` is highlighted AND visible inside the sidebar (not requiring manual scroll).
5. Resize the browser window so the sidebar viewport is short enough that not all sidebar entries fit. Reload, navigate to a note whose sidebar entry was previously below the fold, and confirm the sidebar auto-scrolls to reveal it.
6. Confirm that main-pane scroll position and the page itself are NOT affected by the sidebar scroll.

If the behavior is wrong, stop and debug — do NOT proceed to commit.

Stop the dev server (`Ctrl+C`) once manual verification passes.

- [ ] **Step 5: Run the existing Playwright suite for regressions**

From the worktree root:

```
npx playwright test tests/sidebar-tree.spec.ts
```

Expected: all existing tests still pass. The scroll change cannot hide elements, so `toBeVisible()` assertions in the suite are only made more likely to succeed, not less.

If any test fails, investigate — the failure is almost certainly unrelated to this change, but confirm before moving on.

- [ ] **Step 6: Commit**

This repo tracks the Vite build output (`web/static/app.js`, `web/static/index.html`, `web/static/style.css` are all under version control — verified via `git ls-files web/static/`). So the commit must include both the source change and the rebuilt static asset so the Go binary serves up-to-date JS.

```
git add web/src/app.js web/static/app.js
git commit -m "Scroll sidebar to reveal the selected note's list item (#82)"
```

If `git status` shows additional changes under `web/static/` (e.g. `index.html`, `style.css`) that were NOT expected from this change, stop and investigate — do not silently include unrelated rebuild artifacts.

- [ ] **Step 7: Verify the working tree is clean**

```
git status
```

Expected: `nothing to commit, working tree clean`.

---

## Self-Review Summary

- **Spec coverage:** Single requirement from the spec ("after markSelected, call scrollIntoView on the element") → implemented in Task 1 Step 2.
- **Placeholders:** none — all steps contain exact file paths, exact code, and exact commands.
- **Type consistency:** only one function (`markSelected`) is touched; no cross-task signature drift possible.
- **Scope:** one file changed; one behavior added; no new tests or fixtures per the spec's explicit non-goals.
