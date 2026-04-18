import { describe, it, expect, beforeEach, vi } from 'vitest'
import { TreeView } from './tree-view.js'

function makeLoader(tree) {
  return vi.fn(async (path) => (tree[path] || []).slice())
}

const twoRoot = {
  '': [
    { name: 'a', path: 'a', isDir: true },
    { name: 'readme.md', path: 'readme.md', isDir: false },
  ],
}

describe('TreeView construction', () => {
  let container
  beforeEach(() => {
    document.body.innerHTML = '<div id="host"></div>'
    container = document.getElementById('host')
  })

  it('wipes the container and mounts a tv-root with role="tree"', async () => {
    container.innerHTML = '<p>stale</p>'
    const tv = new TreeView(container, { loader: makeLoader(twoRoot) })
    await tv.ready
    expect(container.querySelector('p')).toBeNull()
    expect(container.querySelector('.tv-root')).toBeTruthy()
    expect(container.querySelector('.tv-root').getAttribute('role')).toBe('tree')
  })

  it('calls loader with rootPath on construction and renders returned children', async () => {
    const loader = makeLoader(twoRoot)
    const tv = new TreeView(container, { loader })
    await tv.ready
    expect(loader).toHaveBeenCalledWith('')
    const items = container.querySelectorAll('[role="treeitem"]')
    expect(items.length).toBe(2)
    expect(items[0].getAttribute('data-path')).toBe('a')
    expect(items[0].classList.contains('tv-item--dir')).toBe(true)
    expect(items[1].getAttribute('data-path')).toBe('readme.md')
    expect(items[1].classList.contains('tv-item--file')).toBe(true)
  })

  it('uses rootPath option when provided', async () => {
    const loader = makeLoader({ 'sub': [{ name: 'x.md', path: 'sub/x.md', isDir: false }] })
    const tv = new TreeView(container, { loader, rootPath: 'sub' })
    await tv.ready
    expect(loader).toHaveBeenCalledWith('sub')
    expect(container.querySelector('[data-path="sub/x.md"]')).toBeTruthy()
  })

  it('honors classPrefix option', async () => {
    const tv = new TreeView(container, {
      loader: makeLoader(twoRoot),
      classPrefix: 'x-',
    })
    await tv.ready
    expect(container.querySelector('.x-root')).toBeTruthy()
    expect(container.querySelector('.tv-root')).toBeNull()
    const item = container.querySelector('[role="treeitem"]')
    expect(item.classList.contains('x-item')).toBe(true)
    expect(item.classList.contains('x-item--dir')).toBe(true)
  })
})

const nested = {
  '': [
    { name: 'a', path: 'a', isDir: true },
    { name: 'b', path: 'b', isDir: true },
    { name: 'readme.md', path: 'readme.md', isDir: false },
  ],
  'a': [
    { name: 'inner.md', path: 'a/inner.md', isDir: false },
  ],
  'b': [
    { name: 'deep', path: 'b/deep', isDir: true },
  ],
  'b/deep': [
    { name: 'x.md', path: 'b/deep/x.md', isDir: false },
  ],
}

describe('TreeView expand/collapse', () => {
  let container
  beforeEach(() => {
    document.body.innerHTML = '<div id="host"></div>'
    container = document.getElementById('host')
  })

  it('expand(path) loads children, inserts rows, flips aria-expanded', async () => {
    const loader = makeLoader(nested)
    const tv = new TreeView(container, { loader })
    await tv.ready
    await tv.expand('a')
    expect(loader).toHaveBeenCalledWith('a')
    const row = container.querySelector('[data-path="a"]')
    expect(row.getAttribute('aria-expanded')).toBe('true')
    expect(container.querySelector('[data-path="a/inner.md"]')).toBeTruthy()
  })

  it('expand(path) is idempotent — no duplicate loader call, no duplicate rows', async () => {
    const loader = makeLoader(nested)
    const tv = new TreeView(container, { loader })
    await tv.ready
    await tv.expand('a')
    await tv.expand('a')
    const callsForA = loader.mock.calls.filter((c) => c[0] === 'a').length
    expect(callsForA).toBe(1)
    expect(container.querySelectorAll('[data-path="a/inner.md"]').length).toBe(1)
  })

  it('concurrent expand(path) calls share one loader call', async () => {
    const loader = makeLoader(nested)
    const tv = new TreeView(container, { loader })
    await tv.ready
    await Promise.all([tv.expand('a'), tv.expand('a')])
    const callsForA = loader.mock.calls.filter((c) => c[0] === 'a').length
    expect(callsForA).toBe(1)
  })

  it('collapse(path) removes DOM subtree but preserves model', async () => {
    const loader = makeLoader(nested)
    const tv = new TreeView(container, { loader })
    await tv.ready
    await tv.expand('a')
    tv.collapse('a')
    expect(container.querySelector('[data-path="a/inner.md"]')).toBeNull()
    expect(container.querySelector('[data-path="a"]').getAttribute('aria-expanded')).toBe('false')
    // Model retained: re-expand does not re-fetch
    await tv.expand('a')
    const callsForA = loader.mock.calls.filter((c) => c[0] === 'a').length
    expect(callsForA).toBe(1)
    expect(container.querySelector('[data-path="a/inner.md"]')).toBeTruthy()
  })

  it('toggle(path) expands if collapsed, collapses if expanded', async () => {
    const loader = makeLoader(nested)
    const tv = new TreeView(container, { loader })
    await tv.ready
    await tv.toggle('a')
    expect(container.querySelector('[data-path="a/inner.md"]')).toBeTruthy()
    await tv.toggle('a')
    expect(container.querySelector('[data-path="a/inner.md"]')).toBeNull()
  })

  it('emits tree:toggle on expand and collapse', async () => {
    const loader = makeLoader(nested)
    const tv = new TreeView(container, { loader })
    await tv.ready
    const events = []
    container.addEventListener('tree:toggle', (e) => events.push(e.detail))
    await tv.expand('a')
    tv.collapse('a')
    expect(events).toEqual([
      { path: 'a', expanded: true },
      { path: 'a', expanded: false },
    ])
  })

  it('expand(path) emits tree:error when loader rejects and leaves state unchanged', async () => {
    const loader = vi.fn(async (path) => {
      if (path === '') return nested['']
      throw new Error('nope')
    })
    const tv = new TreeView(container, { loader })
    await tv.ready
    const errors = []
    container.addEventListener('tree:error', (e) => errors.push(e.detail))
    await tv.expand('a').catch(() => {})
    expect(errors.length).toBe(1)
    expect(errors[0].path).toBe('a')
    expect(errors[0].error.message).toBe('nope')
    expect(container.querySelector('[data-path="a"]').getAttribute('aria-expanded')).toBe('false')
  })
})
