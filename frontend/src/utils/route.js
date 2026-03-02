export function parseRouteSlug(pathname = window.location.pathname || '/') {
  try {
    const decoded = decodeURIComponent(pathname)
    const trimmed = decoded.replace(/^\/+|\/+$/g, '')
    if (!trimmed) return ''

    const parts = trimmed.split('/').filter(Boolean)
    if (parts.length !== 1) return ''
    return parts[0]
  } catch {
    return ''
  }
}
