// notesview front-end bootstrap.
//
// Loads HTMX + SSE, runs syntax highlighting on every swap, and owns
// the sidebar toggle (client-side visibility with localStorage +
// on-open sidebar refresh).

import 'htmx.org';
import 'htmx-ext-sse';
import hljs from 'highlight.js/lib/common';

function highlightIn(root) {
  if (!root || !root.querySelectorAll) return;
  root.querySelectorAll('.markdown-body pre > code').forEach(function (el) {
    hljs.highlightElement(el);
  });
}

document.addEventListener('DOMContentLoaded', function () {
  highlightIn(document);
  wireSidebarToggle();
});

document.body.addEventListener('htmx:afterSwap', function (e) {
  highlightIn(e.target);
});

function wireSidebarToggle() {
  const btn = document.getElementById('sidebar-toggle');
  if (!btn) return;
  const initiallyOpen = document.documentElement.classList.contains('sidebar-open');
  btn.setAttribute('aria-expanded', initiallyOpen ? 'true' : 'false');
  btn.addEventListener('click', toggleSidebar);
}

function toggleSidebar() {
  const root = document.documentElement;
  const btn = document.getElementById('sidebar-toggle');
  const open = root.classList.toggle('sidebar-open');
  if (btn) btn.setAttribute('aria-expanded', open ? 'true' : 'false');
  try {
    localStorage.setItem('notesview.sidebarOpen', open ? '1' : '0');
  } catch (e) {}

  if (open) {
    refreshSidebar();
  }
}

// getSidebarMode returns the current sidebar mode from localStorage.
// Defaults to 'dir' when not set.
function getSidebarMode() {
  try {
    return localStorage.getItem('notesview.sidebarMode') || 'dir';
  } catch (e) {
    return 'dir';
  }
}

// getSidebarTag returns the current sidebar tag from localStorage.
function getSidebarTag() {
  try {
    return localStorage.getItem('notesview.sidebarTag') || '';
  } catch (e) {
    return '';
  }
}

// getSidebarDir returns the current sidebar directory from localStorage.
// Defaults to the note's parent directory when not set.
function getSidebarDir() {
  try {
    const dir = localStorage.getItem('notesview.sidebarDir');
    if (dir !== null) return dir;
  } catch (e) {}
  // Fall back to the note's parent directory.
  const notePath = (document.body.dataset.notePath || '').replace(/^\/+/, '');
  return notePath ? notePath.replace(/[^/]*$/, '').replace(/\/$/, '') : '';
}

// refreshSidebar fetches the sidebar content based on the current
// localStorage mode and updates the #sidebar element via HTMX.
function refreshSidebar() {
  const mode = getSidebarMode();
  let url;
  if (mode === 'tags') {
    url = '/tags';
  } else if (mode === 'tag') {
    const tag = getSidebarTag();
    url = tag ? `/tags/${encodeURIComponent(tag)}` : '/tags';
  } else {
    const dir = getSidebarDir();
    url = `/dir/${dir}`;
  }
  window.htmx && window.htmx.ajax('GET', url, {
    target: '#sidebar',
    swap: 'innerHTML',
  });
}
