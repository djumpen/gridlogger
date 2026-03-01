<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import IntervalBar from './components/IntervalBar.vue'

const projectId = 1
const timezone = 'Europe/Kyiv'
const view = ref('week')
const loading = ref(false)
const error = ref('')
const intervals = ref([])
const stats = ref({
  availabilityPercent: 0,
  totalAvailableHours: 0,
  totalOutageHours: 0
})
const windowFrom = ref('')
const windowTo = ref('')
const dateInputRef = ref(null)
const calendarPopoverRef = ref(null)
const calendarOpen = ref(false)
const telegramWidgetRef = ref(null)
const telegramConfig = ref({
  enabled: false,
  botUsername: '',
  requestAccess: 'write'
})
const currentUser = ref(null)
const authError = ref('')

const telegramCallbackName = 'gridloggerTelegramAuth'

function formatForInput(date) {
  const p = (n) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${p(date.getMonth() + 1)}-${p(date.getDate())}`
}

const selectedDate = ref(formatForInput(new Date()))

const viewOptions = [
  { label: 'День', value: 'day' },
  { label: 'Тиждень', value: 'week' },
  { label: 'Місяць', value: 'month' }
]

const xAxisTicks = [0, 2, 4, 6, 8, 10, 12]

function xTickStyle(tick) {
  const left = `${(tick / 12) * 100}%`
  if (tick === 0) {
    return { left, transform: 'translateX(0)' }
  }
  if (tick === 12) {
    return { left, transform: 'translateX(-100%)' }
  }
  return { left, transform: 'translateX(-50%)' }
}

function clipIntervalsToRange(sourceIntervals, from, to) {
  const fromTs = from.getTime()
  const toTs = to.getTime()
  const out = []

  for (const iv of sourceIntervals) {
    const ivStart = new Date(iv.start).getTime()
    const ivEnd = new Date(iv.end).getTime()
    if (ivEnd <= fromTs || ivStart >= toTs) continue

    out.push({
      start: new Date(Math.max(ivStart, fromTs)).toISOString(),
      end: new Date(Math.min(ivEnd, toTs)).toISOString(),
      status: iv.status
    })
  }

  return out
}

const groupedByDay = computed(() => {
  if (!windowFrom.value || !windowTo.value) return []

  const out = []
  const now = new Date()
  const start = new Date(windowFrom.value)
  const end = new Date(windowTo.value)
  const cursor = new Date(start)

  while (cursor < end) {
    const dayStart = new Date(cursor)
    const dayEnd = new Date(dayStart)
    dayEnd.setDate(dayEnd.getDate() + 1)
    if (dayEnd > end) {
      dayEnd.setTime(end.getTime())
    }

    const dayIntervals = []
    for (const iv of intervals.value) {
      const ivStart = new Date(iv.start)
      const ivEnd = new Date(iv.end)
      if (ivEnd <= dayStart || ivStart >= dayEnd) continue
      const clippedStart = ivStart > dayStart ? ivStart : dayStart
      const clippedEnd = ivEnd < dayEnd ? ivEnd : dayEnd

      if (iv.status !== 'outage' || clippedEnd <= now) {
        dayIntervals.push({
          start: clippedStart.toISOString(),
          end: clippedEnd.toISOString(),
          status: iv.status
        })
        continue
      }

      if (clippedStart >= now) {
        dayIntervals.push({
          start: clippedStart.toISOString(),
          end: clippedEnd.toISOString(),
          status: 'future'
        })
        continue
      }

      dayIntervals.push({
        start: clippedStart.toISOString(),
        end: now.toISOString(),
        status: 'outage'
      })
      dayIntervals.push({
        start: now.toISOString(),
        end: clippedEnd.toISOString(),
        status: 'future'
      })
    }

    const dayLabel = new Intl.DateTimeFormat('en-CA', {
      timeZone: timezone,
      year: 'numeric',
      month: '2-digit',
      day: '2-digit'
    }).format(dayStart)

    out.push({
      day: dayLabel,
      from: dayStart.toISOString(),
      to: dayEnd.toISOString(),
      intervals: dayIntervals,
      amFrom: dayStart.toISOString(),
      amTo: new Date(Math.min(dayStart.getTime() + 12 * 60 * 60 * 1000, dayEnd.getTime())).toISOString(),
      pmFrom: new Date(Math.min(dayStart.getTime() + 12 * 60 * 60 * 1000, dayEnd.getTime())).toISOString(),
      pmTo: dayEnd.toISOString(),
      amIntervals: clipIntervalsToRange(
        dayIntervals,
        dayStart,
        new Date(Math.min(dayStart.getTime() + 12 * 60 * 60 * 1000, dayEnd.getTime()))
      ),
      pmIntervals: clipIntervalsToRange(
        dayIntervals,
        new Date(Math.min(dayStart.getTime() + 12 * 60 * 60 * 1000, dayEnd.getTime())),
        dayEnd
      )
    })
    cursor.setDate(cursor.getDate() + 1)
  }
  return out
})

const currentStatus = ref('unknown')
const currentStatusLabel = computed(() => {
  const labels = {
    available: 'Є світло',
    outage: 'Немає світла',
    future: 'Майбутній період',
    unknown: 'Невідомо'
  }
  return labels[currentStatus.value] || 'Невідомо'
})

const currentUserLabel = computed(() => {
  if (!currentUser.value) return ''
  if (currentUser.value.username) {
    return `@${currentUser.value.username}`
  }

  const firstName = currentUser.value.firstName || ''
  const lastName = currentUser.value.lastName || ''
  return `${firstName} ${lastName}`.trim() || 'Telegram'
})

const windowLabel = computed(() => {
  if (!windowFrom.value || !windowTo.value) return ''
  return `${windowFrom.value.slice(0, 10)} → ${windowTo.value.slice(0, 10)}`
})

function computeWindowForDate(date, mode) {
  const base = new Date(date)
  base.setHours(0, 0, 0, 0)
  let from = new Date(base)
  let to = new Date(base)

  if (mode === 'day') {
    to.setDate(to.getDate() + 1)
  }
  if (mode === 'week') {
    const day = from.getDay() || 7
    from.setDate(from.getDate() - (day - 1))
    to = new Date(from)
    to.setDate(to.getDate() + 7)
  }
  if (mode === 'month') {
    from = new Date(base.getFullYear(), base.getMonth(), 1)
    to = new Date(base.getFullYear(), base.getMonth() + 1, 1)
  }

  return { from, to }
}

const isCurrentIntervalSelected = computed(() => {
  const selected = computeWindowForDate(new Date(`${selectedDate.value}T00:00:00`), view.value)
  const now = computeWindowForDate(new Date(), view.value)
  return selected.from.getTime() === now.from.getTime() && selected.to.getTime() === now.to.getTime()
})

function openDatePicker() {
  calendarOpen.value = !calendarOpen.value
  if (!calendarOpen.value) return
  nextTick(() => {
    dateInputRef.value?.focus()
  })
}

function closeDatePicker() {
  calendarOpen.value = false
}

function handleDateChange() {
  loadAvailability()
  closeDatePicker()
}

function shiftWindow(step) {
  const current = new Date(`${selectedDate.value}T00:00:00`)
  if (Number.isNaN(current.getTime())) return

  if (view.value === 'day') {
    current.setDate(current.getDate() + step)
  }
  if (view.value === 'week') {
    current.setDate(current.getDate() + (step * 7))
  }
  if (view.value === 'month') {
    current.setMonth(current.getMonth() + step)
  }

  selectedDate.value = formatForInput(current)
  loadAvailability()
}

function goToCurrentWindow() {
  selectedDate.value = formatForInput(new Date())
  loadAvailability()
}

function handleOutsidePointerDown(event) {
  if (!calendarOpen.value) return
  if (calendarPopoverRef.value?.contains(event.target)) return
  closeDatePicker()
}

function computeWindow() {
  const seed = new Date(`${selectedDate.value}T00:00:00`)
  if (Number.isNaN(seed.getTime())) {
    throw new Error('Некоректна дата')
  }
  return computeWindowForDate(seed, view.value)
}

async function loadAvailability() {
  try {
    loading.value = true
    error.value = ''
    const { from, to } = computeWindow()
    windowFrom.value = from.toISOString()
    windowTo.value = to.toISOString()

    const url = `/api/projects/${projectId}/availability?from=${encodeURIComponent(windowFrom.value)}&to=${encodeURIComponent(windowTo.value)}`
    const resp = await fetch(url)
    if (!resp.ok) {
      throw new Error(await resp.text())
    }
    const data = await resp.json()
    intervals.value = data.intervals
    stats.value = data.stats
    await loadCurrentStatus()
  } catch (e) {
    error.value = e.message || 'Не вдалося завантажити дані'
    currentStatus.value = 'unknown'
  } finally {
    loading.value = false
  }
}

async function loadCurrentStatus() {
  const now = new Date()
  const from = new Date(now.getTime() - (3 * 60 * 60 * 1000))
  const to = new Date(now.getTime() + 60 * 1000)
  const url = `/api/projects/${projectId}/availability?from=${encodeURIComponent(from.toISOString())}&to=${encodeURIComponent(to.toISOString())}`
  const resp = await fetch(url)
  if (!resp.ok) {
    currentStatus.value = 'unknown'
    return
  }

  const data = await resp.json()
  const nowTs = now.getTime()
  const hit = (data.intervals || []).find((iv) => {
    const start = new Date(iv.start).getTime()
    const end = new Date(iv.end).getTime()
    return nowTs >= start && nowTs < end
  })
  currentStatus.value = hit?.status || 'unknown'
}

async function loadTelegramConfig() {
  try {
    const resp = await fetch('/auth/telegram/config', { credentials: 'include' })
    if (!resp.ok) {
      telegramConfig.value = { enabled: false, botUsername: '', requestAccess: 'write' }
      return
    }
    const data = await resp.json()
    telegramConfig.value = {
      enabled: !!data.enabled,
      botUsername: data.botUsername || '',
      requestAccess: data.requestAccess || 'write'
    }
  } catch {
    telegramConfig.value = { enabled: false, botUsername: '', requestAccess: 'write' }
  }
}

async function loadMe() {
  try {
    const resp = await fetch('/me', { credentials: 'include' })
    if (resp.status === 401) {
      currentUser.value = null
      return
    }
    if (!resp.ok) {
      currentUser.value = null
      return
    }
    const data = await resp.json()
    currentUser.value = data.user || null
  } catch {
    currentUser.value = null
  }
}

function clearTelegramWidget() {
  if (telegramWidgetRef.value) {
    telegramWidgetRef.value.innerHTML = ''
  }
}

function renderTelegramWidget() {
  clearTelegramWidget()
  if (!telegramConfig.value.enabled || !telegramConfig.value.botUsername || currentUser.value) {
    return
  }
  if (!telegramWidgetRef.value) {
    return
  }

  window[telegramCallbackName] = async (user) => {
    try {
      authError.value = ''
      const body = new URLSearchParams()
      for (const [key, value] of Object.entries(user || {})) {
        if (value === undefined || value === null) continue
        body.append(key, String(value))
      }

      const resp = await fetch('/auth/telegram/callback', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/x-www-form-urlencoded'
        },
        credentials: 'include',
        body
      })
      if (!resp.ok) {
        throw new Error(await resp.text())
      }
      const data = await resp.json()
      currentUser.value = data.user || null
      clearTelegramWidget()
    } catch (e) {
      authError.value = e.message || 'Не вдалося увійти через Telegram'
    }
  }

  const script = document.createElement('script')
  script.async = true
  script.src = 'https://telegram.org/js/telegram-widget.js?22'
  script.setAttribute('data-telegram-login', telegramConfig.value.botUsername)
  script.setAttribute('data-size', 'medium')
  script.setAttribute('data-userpic', 'false')
  script.setAttribute('data-request-access', telegramConfig.value.requestAccess)
  script.setAttribute('data-onauth', `${telegramCallbackName}(user)`)
  telegramWidgetRef.value.appendChild(script)
}

async function logout() {
  try {
    await fetch('/auth/logout', {
      method: 'POST',
      credentials: 'include'
    })
  } finally {
    currentUser.value = null
    authError.value = ''
    renderTelegramWidget()
  }
}

onMounted(() => {
  loadAvailability()
  loadTelegramConfig().then(() => {
    loadMe().then(() => {
      renderTelegramWidget()
    })
  })
  document.addEventListener('pointerdown', handleOutsidePointerDown)
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', handleOutsidePointerDown)
  delete window[telegramCallbackName]
})
</script>

<template>
  <main class="page">
    <div class="topbar">
      <p class="kicker">Svitlo.🏘️</p>
      <div class="topbar-actions">
        <template v-if="currentUser">
          <span class="user-chip">{{ currentUserLabel }}</span>
          <button class="login-link" type="button" @click="logout">Вийти</button>
        </template>
        <div v-else-if="telegramConfig.enabled" class="telegram-widget-wrap">
          <div ref="telegramWidgetRef"></div>
        </div>
        <span v-else class="auth-muted">Вхід через Telegram недоступний</span>
      </div>
    </div>
    <p v-if="authError" class="error auth-error">{{ authError }}</p>

    <header class="hero">
      <div>
        <h1>Коновальця 36Б</h1>
        <p class="sub">м. Київ. Ввод #1</p>
      </div>
    </header>

    <section class="stats">
      <article>
        <h2 class="current-status-row" :class="`status-${currentStatus.replace(' ', '-')}`">
          <span class="status-dot" aria-hidden="true"></span>
          <span>{{ currentStatusLabel }}</span>
        </h2>
        <p>Поточний стан</p>
      </article>
      <article>
        <h2>{{ stats.availabilityPercent.toFixed(1) }}%</h2>
        <p>Наявність у цьому інтервалі</p>
      </article>
      <article>
        <h2>{{ stats.totalAvailableHours.toFixed(1) }} год</h2>
        <p>Загалом зі світлом</p>
      </article>
      <article>
        <h2>{{ stats.totalOutageHours.toFixed(1) }} год</h2>
        <p>Загалом без світла</p>
      </article>
    </section>

    <section class="calendar" v-if="!loading && !error">
      <header>
        <div class="calendar-title-row">
          <h3>Інтервали</h3>
          <div class="tabs">
            <button
              v-for="item in viewOptions"
              :key="item.value"
              :class="{ active: view === item.value }"
              @click="view = item.value; loadAvailability()"
            >
              {{ item.label }}
            </button>
          </div>
        </div>
        <div class="window-row">
          <button class="nav-btn" type="button" @click="shiftWindow(-1)" aria-label="Попередній інтервал">←</button>
          <div class="calendar-popover-wrap" ref="calendarPopoverRef">
            <button
              class="window-btn"
              type="button"
              @click="openDatePicker"
              title="Обрати дату для цього інтервалу"
            >
              {{ windowLabel }}
            </button>
            <div v-if="calendarOpen" class="calendar-popover">
              <input
                ref="dateInputRef"
                v-model="selectedDate"
                type="date"
                @change="handleDateChange"
              />
            </div>
          </div>
          <button class="nav-btn" type="button" @click="shiftWindow(1)" aria-label="Наступний інтервал">→</button>
          <button
            v-if="!isCurrentIntervalSelected"
            class="current-btn"
            type="button"
            @click="goToCurrentWindow"
          >
            Поточний
          </button>
        </div>
      </header>
      <div class="rows" v-if="groupedByDay.length">
        <div class="row axis-row">
          <div></div>
          <div class="x-axis-legend">
            <span
              v-for="tick in xAxisTicks"
              :key="tick"
              class="x-axis-tick"
              :style="xTickStyle(tick)"
            >
              {{ tick }}
            </span>
          </div>
        </div>
        <div v-for="item in groupedByDay" :key="item.day" class="row">
          <div class="day">{{ item.day }}</div>
          <div class="day-bars">
            <div class="day-half">
              <span class="half-label">AM</span>
              <IntervalBar :intervals="item.amIntervals" :from="item.amFrom" :to="item.amTo" :show-labels="false" />
            </div>
            <div class="day-half">
              <span class="half-label">PM</span>
              <IntervalBar :intervals="item.pmIntervals" :from="item.pmFrom" :to="item.pmTo" :show-labels="false" />
            </div>
          </div>
        </div>
      </div>
      <p v-else>Інтервалів ще немає. Надішліть ping, щоб почати відстеження.</p>
    </section>

    <p v-if="loading">Завантаження…</p>
    <p v-if="error" class="error">{{ error }}</p>
  </main>
</template>
