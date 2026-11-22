// Demo API for local UI work and browser tests. No database, no dependencies.
// Serves the read endpoints the frontend needs, with fixed sample data.
// MOCK_EMPTY=1 returns no public projects. PORT sets the port (default 8080).
import http from 'node:http'

const port = Number(process.env.PORT || 8080)
const empty = process.env.MOCK_EMPTY === '1'

const projects = empty ? [] : [
  { id: 1, name: 'Вул. Сонячна, 12', slug: 'sonyachna-12', city: 'Київ', description: 'Демо-адреса', isPublic: true, notificationsEnabled: true, hasOutageSchedule: false },
  { id: 2, name: 'Вул. Лугова, 5', slug: 'lugova-5', city: 'Львів', description: 'Демо-адреса', isPublic: true, notificationsEnabled: true, hasOutageSchedule: false }
]

// Pattern is anchored to absolute time (5h on, 2h off from the epoch), so every
// window agrees about what the status is at a given moment.
function availability(from, to) {
  const hour = 3600 * 1000
  const maxSpan = 400 * 24 * hour
  const start = new Date(from).getTime()
  const end = new Date(to).getTime()
  if (!Number.isFinite(start) || !Number.isFinite(end) || end <= start || end - start > maxSpan) return null
  const cycle = 7 * hour
  const intervals = []
  let t = start
  while (t < end) {
    const pos = ((t % cycle) + cycle) % cycle
    const isAvailable = pos < 5 * hour
    const boundary = t - pos + (isAvailable ? 5 * hour : cycle)
    const next = Math.min(end, boundary)
    intervals.push({ start: new Date(t).toISOString(), end: new Date(next).toISOString(), status: isAvailable ? 'available' : 'outage' })
    t = next
  }
  const sum = (s) => intervals.filter((i) => i.status === s).reduce((a, i) => a + (new Date(i.end) - new Date(i.start)), 0) / hour
  const avail = sum('available')
  const out = sum('outage')
  const total = avail + out || 1
  return {
    intervals,
    stats: {
      availabilityPercent: Math.round((avail / total) * 1000) / 10,
      totalAvailableHours: Math.round(avail * 10) / 10,
      totalOutageHours: Math.round(out * 10) / 10
    }
  }
}

const json = (res, code, body) => {
  res.writeHead(code, { 'Content-Type': 'application/json' })
  res.end(JSON.stringify(body))
}

http.createServer((req, res) => {
  const url = new URL(req.url, 'http://x')
  const p = url.pathname
  let m
  if (p === '/healthz' || p === '/readyz') return json(res, 200, { status: 'ok' })
  if (p === '/api/projects') return json(res, 200, { projects })
  if ((m = p.match(/^\/api\/project-slugs\/(.+)$/))) {
    let slug = m[1]
    try { slug = decodeURIComponent(slug) } catch { return json(res, 400, { error: 'bad slug' }) }
    const project = projects.find((x) => x.slug === slug)
    return project ? json(res, 200, { project }) : json(res, 404, { error: 'project not found' })
  }
  if ((m = p.match(/^\/api\/projects\/(\d+)\/availability$/))) {
    const data = availability(url.searchParams.get('from'), url.searchParams.get('to'))
    return data ? json(res, 200, data) : json(res, 400, { error: 'invalid or too long time window' })
  }
  if (p === '/api/auth/telegram/config') return json(res, 200, { enabled: false })
  if (p === '/api/me') return json(res, 401, { error: 'unauthorized' })
  if (p.endsWith('/notifications/subscription')) return json(res, 401, { error: 'unauthorized' })
  if (p.endsWith('/yasno')) return json(res, 200, { schedule: null })
  json(res, 404, { error: 'not found in demo api' })
}).listen(port, '127.0.0.1', () => console.log(`demo api on :${port}${empty ? ' (empty)' : ''}`))
