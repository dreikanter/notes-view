// nview front-end bootstrap.
//
// Loads HTMX + SSE support, runs syntax highlighting on swaps, owns sidebar
// toggle/navigation, and keeps note updates flowing through the unified
// /events endpoint.

import htmx from 'htmx.org'
import 'htmx-ext-sse'
import hljs from 'highlight.js/lib/common'

function highlightIn(root) {
  if (!root || !root.querySelectorAll) return
  root.querySelectorAll('.markdown-body pre > code').forEach(function (el) {
    hljs.highlightElement(el)
  })
}

let es = null
let watchedNote = ''

function encodePath(p) {
  if (!p) return ''
  return p.split('/').map(encodeURIComponent).join('/')
}

function decodeHref(href) {
  const card = document.getElementById('note-card')
  if (card) return card.getAttribute('data-note-path') || ''
  return ''
}

function openEventSource(notePath) {
  if (es) es.close()
  watchedNote = notePath || ''
  window.__tvWatchedNote = watchedNote
  es = new EventSource('/events' + (watchedNote ? '?watch=' + encodeURIComponent(watchedNote) : ''))
  es.addEventListener('change', () => {
    if (!watchedNote) return
    const card = document.getElementById('note-card')
    const href = card?.getAttribute('data-view-href') || location.pathname
    htmx.ajax('GET', href, {
      target: '#note-pane',
      swap: 'innerHTML',
      headers: { 'HX-Target': 'note-pane' },
    })
  })
  es.addEventListener('dir-changed', () => {
    const selected = watchedNote ? '?selected=' + encodeURIComponent(watchedNote) : ''
    htmx.ajax('GET', '/sidebar' + selected, {
      target: '#sidebar',
      swap: 'innerHTML',
    })
  })
}

document.addEventListener('DOMContentLoaded', function () {
  highlightIn(document)
  wireSidebarToggle()
  wireThemeToggle()
  openEventSource(document.body.getAttribute('data-note-path') || '')
})

function wireSidebarToggle() {
  const btn = document.getElementById('sidebar-toggle')
  if (!btn) return
  const initiallyOpen = document.documentElement.classList.contains('sidebar-open')
  btn.setAttribute('aria-expanded', initiallyOpen ? 'true' : 'false')
  btn.addEventListener('click', toggleSidebar)
}

function toggleSidebar() {
  const root = document.documentElement
  const btn = document.getElementById('sidebar-toggle')
  const open = root.classList.toggle('sidebar-open')
  if (btn) btn.setAttribute('aria-expanded', open ? 'true' : 'false')
  try {
    localStorage.setItem('nview.sidebarOpen', open ? '1' : '0')
  } catch (e) {}
}

function wireThemeToggle() {
  const btn = document.getElementById('theme-toggle')
  if (!btn) return
  const root = document.documentElement
  btn.setAttribute('aria-pressed', root.classList.contains('dark') ? 'true' : 'false')
  btn.addEventListener('click', () => {
    const isDark = root.classList.toggle('dark')
    btn.setAttribute('aria-pressed', isDark ? 'true' : 'false')
    try {
      localStorage.setItem('nview.theme', isDark ? 'dark' : 'light')
    } catch (e) {}
    const light = document.getElementById('hljs-light')
    const dark = document.getElementById('hljs-dark')
    if (light) light.disabled = isDark
    if (dark) dark.disabled = !isDark
  })
}

function loadIntoPane(href, state) {
  htmx.ajax('GET', href, {
    target: '#note-pane',
    swap: 'innerHTML',
    headers: { 'HX-Target': 'note-pane' },
  })
  if (state) history.pushState(state, '', href)
}

window.selectTag = function(tag, skipPush) {
  const href = '/tags/' + encodeURIComponent(tag)
  loadIntoPane(href, skipPush ? null : { type: 'tag', tag: tag, href: href })
  openEventSource('')
}

let pendingNoteScrollReset = false

document.addEventListener('click', function(e) {
  const uidLink = e.target.closest('.uid-link')
  if (uidLink) {
    e.preventDefault()
    const uid = (uidLink.dataset.noteUid || uidLink.textContent || '').replace(/^#/, '').trim()
    if (uid) scrollSidebarToUID(uid)
    return
  }

  const link = e.target.closest('[data-action]')
  if (!link) return
  const action = link.dataset.action
  if (action === 'selectTag') {
    e.preventDefault()
    selectTag(link.dataset.entryName, false)
  } else if (action === 'selectDir' || action === 'selectNote' || action === 'selectIndex') {
    e.preventDefault()
    const href = link.dataset.entryHref || link.getAttribute('href')
    pendingNoteScrollReset = true
    const stateType = action === 'selectNote' ? 'note' : 'index'
    loadIntoPane(href, { type: stateType, href })
    openEventSource(action === 'selectNote' ? decodeHref(href) : '')
  }
})

function scrollSidebarToUID(uid) {
  const link = document.querySelector('#sidebar [data-note-uid="' + cssEscape(uid) + '"]')
  if (!link) return
  link.scrollIntoView({ block: 'center', inline: 'nearest' })
  link.classList.add('bg-blue-50', 'dark:bg-blue-950')
  setTimeout(() => link.classList.remove('bg-blue-50', 'dark:bg-blue-950'), 1200)
}

function cssEscape(s) {
  if (typeof CSS !== 'undefined' && CSS.escape) return CSS.escape(s)
  return String(s).replace(/["\\[\]]/g, '\\$&')
}

window.addEventListener('popstate', function(e) {
  const state = e.state
  const href = state?.href || location.pathname
  loadIntoPane(href, null)
  openEventSource(href.startsWith('/n/') ? decodeHref(href) : '')
})

document.body.addEventListener('htmx:afterSwap', function(e) {
  highlightIn(e.target)
  if (e.target && e.target.id === 'note-pane') {
    const noteCard = e.target.querySelector('[data-note-path]')
    const notePath = noteCard ? noteCard.getAttribute('data-note-path') : ''
    if (notePath) openEventSource(notePath)
  }
  if (pendingNoteScrollReset && e.target && e.target.id === 'note-pane') {
    e.target.scrollTop = 0
    pendingNoteScrollReset = false
  }
})
