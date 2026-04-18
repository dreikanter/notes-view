# Sidebar Scroll-Into-View on Selection — Design Spec

## Overview

When a note is opened from the main pane (index view or directory listing), the sidebar highlights the matching entry but does not scroll the sidebar's viewport to reveal it. For deeply nested notes or long listings, the highlight can sit off-screen, forcing the user to scroll the sidebar manually to confirm location.

This spec covers a minimal, single-file front-end change that scrolls the selected sidebar entry into view after every relevant HTMX swap.

## Goals

- After the sidebar swap completes and the selected entry gains its highlight classes, the entry is guaranteed to be visible inside the sidebar viewport.
- No visible jitter when multiple `htmx:afterSwap` events fire for a single navigation (e.g., main pane + sidebar reload).
- No regressions to main-pane scroll position or page-level scroll.

## Non-goals

- Animating the scroll (native behavior is sufficient).
- Restoring sidebar scroll position across page loads.
- Manual visibility/jitter guards. `scrollIntoView({ block: 'center' })` computes a zero scroll delta when the element is already centered, so back-to-back calls produce no visible movement in practice.

## Current Behavior (as of commit `d98dbe2`)

- `markSelected(selector)` in `web/src/app.js:99-103` adds highlight classes to the matching sidebar entry.
- The `htmx:afterSwap` listener at `web/src/app.js:266-276` re-applies selection after every swap. It already wraps `markSelected` in `setTimeout(..., 0)` so the call runs after the swapped DOM is in place.
- `selectNote` (`web/src/app.js:163-197`) triggers a parent-directory reload when opening a nested note. The post-swap re-application above means the highlight lands on the freshly swapped element, not the pre-swap one.
- `#sidebar` (`web/templates/layout.html:32`) is declared with `overflow-y-auto` and owns an independent scroll context. A `scrollIntoView` call on a sidebar descendant scrolls only the sidebar.

## Design

### Change summary

Extend `markSelected` in `web/src/app.js` so that, immediately after adding the highlight classes, it calls:

```js
el.scrollIntoView({ block: 'center', inline: 'nearest' });
```

That is the entire behavioral change. No other files are touched.

### Why this seam

- `markSelected` is the single point of contact between "selection changed" and the DOM. Every navigation path — sidebar click, main-pane link, `popstate`, or HTMX re-swap — funnels through it.
- The existing `htmx:afterSwap` handler already owns the timing: it defers `markSelected` via `setTimeout(0)` until the new subtree has been inserted. Adding the scroll call inside `markSelected` inherits that timing for free.
- Putting the scroll in any other location (e.g., directly in `selectNote`) would miss the cases where the element is only created after a subsequent swap.

### Why `block: 'center'`

Centering the entry in the sidebar viewport surfaces useful context above and below the selection when the list is long. `block: 'nearest'` would only reveal the entry at the sidebar's edge — functionally sufficient, but less readable.

In practice `'center'` is effectively idempotent: once an element is at the center, the browser's computed scroll delta for a follow-up `scrollIntoView({ block: 'center' })` call is zero, so back-to-back calls during the two `htmx:afterSwap` events per navigation produce no visible movement.

`inline: 'nearest'` prevents any accidental horizontal scroll in edge cases.

### Timing and re-swaps

Each note navigation can fire two `htmx:afterSwap` events — one for `#note-pane` and one for `#files-content`. The existing listener defers `markSelected` via `setTimeout(..., 0)` so the swapped DOM is in place. If the first call lands on the same element position the second call will recompute a zero delta. If the sidebar re-swap changes the target element's position, the second call shifts the viewport to the new center — the desired behavior.

If the matching sidebar entry does not yet exist when `markSelected` runs, `querySelector` returns `null` and the existing `if (el)` guard short-circuits — no highlight, no scroll.

### Scroll scope

`scrollIntoView` walks up looking for the nearest scrollable ancestor and scrolls it. The sidebar's `aside#sidebar` has `overflow-y-auto`; the main pane is a sibling, not an ancestor, so it is unaffected. Page-level scroll is not triggered because the sidebar is positioned `fixed`.

### No new markup or CSS

The selected entry is already discoverable via `[data-entry-href="..."]`. No additional identifiers, data attributes, or styling are required.

## Risks

- **Element not found.** If `markSelected` runs before the matching DOM exists, `querySelector` returns `null` and the existing `if (el)` guard prevents both the highlight and the new scroll call from executing.
- **Cross-browser support.** `Element.scrollIntoView` with an options object is supported across all modern browsers (Chrome, Firefox, Safari, Edge). Notes-view already targets evergreen browsers, so no fallback is needed.
- **Test fragility.** The existing Playwright suite asserts visibility of entries after navigation. `scrollIntoView` can only improve visibility, not hide elements, so existing assertions remain valid. No new tests are required by this change.

## Alternatives Considered

1. **`block: 'nearest'`.** Reveals the entry at whichever edge of the sidebar is closer — no context above or below. Worse readability for long lists than centering.
2. **`block: 'center'` with an explicit `getBoundingClientRect` visibility guard.** Belt-and-braces against jitter. Rejected as preemptive complexity: since the follow-up scroll delta is zero once the element is centered, there is nothing to guard against.
3. **`Element.scrollIntoViewIfNeeded({ centerIfNeeded: true })`.** Non-standard; implemented in WebKit/Chromium but not Firefox. Not portable.
4. **Server-side signal (out-of-band swap with `hx-swap-oob` that carries a "focus me" marker).** Heavier than needed for a UI-only concern and couples presentation to server templates.

The plain one-line `scrollIntoView({ block: 'center', inline: 'nearest' })` call is chosen because it gives the best-positioned scroll and uses only standard APIs. If observed jitter ever justifies the visibility guard, it can be added later.

## Testing

- Manual verification: navigate to a note whose sidebar entry would be below the fold (e.g., a note inside a long directory listing) and confirm the sidebar auto-scrolls to reveal it.
- Regression check: ensure `tests/sidebar-tree.spec.ts` still passes without modification.
- No new Playwright assertions are required; the fix is a visual convenience layered on top of existing, already-tested selection behavior.

## References

- Issue: https://github.com/dreikanter/notes-view/issues/82
- Relates to #61 (UI adjustments umbrella issue).
- Relevant files: `web/src/app.js`, `web/templates/layout.html`.
