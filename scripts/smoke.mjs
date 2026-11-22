// Smoke test: start the demo stack, check the page and the API respond, stop.
import { spawn } from 'node:child_process'
import net from 'node:net'
const free = (port) => new Promise((ok) => {
  const s = net.createServer().once('error', () => ok(false)).once('listening', () => s.close(() => ok(true)))
  s.listen(port)
})
for (const port of [5173, Number(process.env.PORT || 8080)]) {
  if (!(await free(port))) { console.error(`smoke failed: port ${port} is already in use`); process.exit(1) }
}
const stack = spawn('node', ['scripts/dev-demo.mjs'], { stdio: ['ignore', 'ignore', 'inherit'], detached: true })
const wait = (ms) => new Promise((r) => setTimeout(r, ms))
async function until(url, check) {
  for (let i = 0; i < 40; i++) {
    try { const r = await fetch(url); if (await check(r)) return } catch {}
    await wait(500)
  }
  throw new Error(`smoke failed: ${url}`)
}
const cleanup = () => { try { process.kill(-stack.pid) } catch {} }
process.on('SIGINT', () => { cleanup(); process.exit(130) })
process.on('SIGTERM', () => { cleanup(); process.exit(143) })
try {
  await until('http://localhost:5173/', async (r) => r.ok && (await r.text()).includes('<div id="app">'))
  await until('http://localhost:5173/api/projects', async (r) => r.ok && Array.isArray((await r.json()).projects))
  console.log('smoke ok')
} catch (e) {
  console.error(e.message); process.exitCode = 1
} finally {
  cleanup()
}
