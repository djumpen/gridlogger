// One command for the UI without Docker or a database: demo API + Vite dev server.
import { spawn } from 'node:child_process'
const apiPort = process.env.PORT || 8080
const spawnGroup = (cmd, args, opts = {}) => spawn(cmd, args, { stdio: 'inherit', detached: true, ...opts })
const procs = [
  spawnGroup('node', ['scripts/mock-api.mjs']),
  spawnGroup('npm', ['run', 'dev'], { cwd: 'frontend', env: { ...process.env, API_PROXY_TARGET: `http://127.0.0.1:${apiPort}` } })
]
let code = 0
let stopping = false
function stop() {
  if (stopping) return
  stopping = true
  // Kill whole process groups so Vite (a child of npm) stops too.
  for (const p of procs) { try { process.kill(-p.pid) } catch {} }
}
process.on('SIGINT', () => { stop() })
process.on('SIGTERM', () => { stop() })
process.on('SIGHUP', () => { stop() })
process.on('exit', () => { stop() })
let exited = 0
for (const p of procs) {
  p.on('error', (e) => { console.error(`dev-demo: failed to start a process: ${e.message}`); code = 1; stop() })
  p.on('exit', (c, sig) => {
    if (!stopping && (c ?? 1) !== 0) code = c ?? 1
    stop()
    if (++exited === procs.length) process.exit(code)
  })
}
