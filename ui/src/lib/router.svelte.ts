// Tiny history router: same-origin <a href="/…"> clicks become pushState navigations.
export const route = $state({ path: location.pathname })

export function navigate(to: string, replace = false) {
  if (to === location.pathname + location.search) return
  history[replace ? 'replaceState' : 'pushState'](null, '', to)
  route.path = location.pathname
  window.scrollTo(0, 0)
}

addEventListener('popstate', () => (route.path = location.pathname))
addEventListener('click', e => {
  if (e.defaultPrevented || e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return
  const a = (e.target as Element | null)?.closest?.('a')
  if (!a || a.target || a.hasAttribute('download') || a.origin !== location.origin) return
  if (a.pathname.startsWith('/api/') || a.pathname.startsWith('/_/')) return
  e.preventDefault()
  navigate(a.pathname + a.search)
})
