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

// Synthesize a short click sound using the Web Audio API.
// A brief burst of noise with sharp decay mimics a mechanical key click.
let audioCtx = null;
let clickBuffer = null;

function initClickBuffer() {
  const sampleRate = audioCtx.sampleRate;
  const length = Math.floor(sampleRate * 0.008);
  clickBuffer = audioCtx.createBuffer(1, length, sampleRate);
  const data = clickBuffer.getChannelData(0);
  for (let i = 0; i < length; i++) {
    const envelope = 1 - i / length;
    data[i] = (Math.random() * 2 - 1) * envelope * envelope;
  }
}

function playClick() {
  if (!audioCtx) {
    audioCtx = new AudioContext();
    initClickBuffer();
  }
  const src = audioCtx.createBufferSource();
  src.buffer = clickBuffer;
  const gain = audioCtx.createGain();
  gain.gain.value = 0.15;
  src.connect(gain);
  gain.connect(audioCtx.destination);
  src.start();
}

document.addEventListener('DOMContentLoaded', function () {
  highlightIn(document);
  wireSidebarToggle();

  const sidebar = document.getElementById('sidebar');
  if (sidebar) {
    sidebar.addEventListener('mouseenter', function (e) {
      const link = e.target.closest('#sidebar a');
      if (link) playClick();
    }, true);
  }
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
    // Refresh the sidebar for the current note: while hidden, the
    // sidebar's DOM froze at its last render, but the user may have
    // clicked wiki-links and moved to a different note.
    window.htmx && window.htmx.ajax('GET', currentSidebarUrl(), {
      target: '#sidebar',
      swap: 'innerHTML',
    });
  } else {
    // Closing strips ?dir= from the URL (intentional, per spec). No
    // pushState — this is a UI preference, not a navigation event.
    const url = new URL(window.location.href);
    url.searchParams.delete('dir');
    history.replaceState(null, '', url.toString());
  }
}

// currentSidebarUrl builds the URL for refreshing the sidebar for the
// current note. The note path is stashed on <body> by the layout
// template (data-note-path) and re-stashed on #note-card for resilience
// across note-pane swaps.
function currentSidebarUrl() {
  const notePath = (document.body.dataset.notePath || '').replace(/^\/+/, '');
  const parent = notePath ? notePath.replace(/[^/]*$/, '').replace(/\/$/, '') : '';
  const base = notePath ? `/view/${notePath}` : '/';
  return `${base}?dir=${encodeURIComponent(parent)}`;
}
