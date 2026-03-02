function toInt(raw) {
  const n = Number.parseInt(raw, 10)
  if (!Number.isInteger(n) || n <= 0) return null
  return n
}

export function parseAppRoute(pathname = window.location.pathname || '/') {
  try {
    const decoded = decodeURIComponent(pathname)
    const trimmed = decoded.replace(/^\/+|\/+$/g, '')
    if (!trimmed) {
      return { name: 'landing' }
    }

    const parts = trimmed.split('/').filter(Boolean)

    if (parts[0] === 'a' && parts[1] === 'settings' && parts.length === 2) {
      return { name: 'settings' }
    }

    if (parts[0] === 'a' && parts[1] === 'settings' && parts[2] === 'project' && parts.length === 4) {
      const id = toInt(parts[3])
      if (id) {
        return { name: 'settings-project', projectId: id }
      }
      return { name: 'not-found' }
    }

    if (parts[0] === 'a' && parts[1] === 'projects' && parts.length === 3) {
      const id = toInt(parts[2])
      if (id) {
        return { name: 'settings-project', projectId: id }
      }
      return { name: 'not-found' }
    }

    if (parts.length === 1) {
      return { name: 'public-project', slug: parts[0] }
    }

    return { name: 'not-found' }
  } catch {
    return { name: 'not-found' }
  }
}
