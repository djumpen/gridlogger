<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import IntervalBar from './IntervalBar.vue'

const props = defineProps({
  project: {
    type: Object,
    required: true
  }
})

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
const notificationsAvailable = ref(false)
const notificationsLoading = ref(false)
const notificationsSaving = ref(false)
const notificationsError = ref('')
const notificationsSubscribed = ref(false)

const selectedDate = ref(formatForInput(new Date()))
const viewOptions = [
  { label: 'День', value: 'day' },
  { label: 'Тиждень', value: 'week' },
  { label: 'Місяць', value: 'month' }
]
const xAxisTicks = [0, 2, 4, 6, 8, 10, 12]

const projectSubtitle = computed(() => {
  const parts = []
  if (props.project.city) {
    parts.push(`м. ${props.project.city}`)
  }
  if (props.project.description) {
    parts.push(props.project.description)
  }
  return parts.join('. ')
})

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

    const noon = new Date(Math.min(dayStart.getTime() + 12 * 60 * 60 * 1000, dayEnd.getTime()))
    out.push({
      day: new Intl.DateTimeFormat('en-CA', {
        timeZone: timezone,
        year: 'numeric',
        month: '2-digit',
        day: '2-digit'
      }).format(dayStart),
      amFrom: dayStart.toISOString(),
      amTo: noon.toISOString(),
      pmFrom: noon.toISOString(),
      pmTo: dayEnd.toISOString(),
      amIntervals: clipIntervalsToRange(dayIntervals, dayStart, noon),
      pmIntervals: clipIntervalsToRange(dayIntervals, noon, dayEnd)
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

const windowLabel = computed(() => {
  if (!windowFrom.value || !windowTo.value) return ''
  return `${windowFrom.value.slice(0, 10)} → ${windowTo.value.slice(0, 10)}`
})

const isCurrentIntervalSelected = computed(() => {
  const selected = computeWindowForDate(new Date(`${selectedDate.value}T00:00:00`), view.value)
  const now = computeWindowForDate(new Date(), view.value)
  return selected.from.getTime() === now.from.getTime() && selected.to.getTime() === now.to.getTime()
})

function formatForInput(date) {
  const p = (n) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${p(date.getMonth() + 1)}-${p(date.getDate())}`
}

function xTickStyle(tick) {
  const left = `${(tick / 12) * 100}%`
  if (tick === 0) return { left, transform: 'translateX(0)' }
  if (tick === 12) return { left, transform: 'translateX(-100%)' }
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

function handleAuthChanged() {
  loadNotificationSubscription()
}

function computeWindow() {
  const seed = new Date(`${selectedDate.value}T00:00:00`)
  if (Number.isNaN(seed.getTime())) {
    throw new Error('Некоректна дата')
  }
  return computeWindowForDate(seed, view.value)
}

async function loadCurrentStatus() {
  if (!props.project?.id) return

  const now = new Date()
  const from = new Date(now.getTime() - (3 * 60 * 60 * 1000))
  const to = new Date(now.getTime() + 60 * 1000)
  const url = `/api/projects/${props.project.id}/availability?from=${encodeURIComponent(from.toISOString())}&to=${encodeURIComponent(to.toISOString())}`
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

async function loadAvailability() {
  if (!props.project?.id) return

  try {
    loading.value = true
    error.value = ''

    const { from, to } = computeWindow()
    windowFrom.value = from.toISOString()
    windowTo.value = to.toISOString()

    const url = `/api/projects/${props.project.id}/availability?from=${encodeURIComponent(windowFrom.value)}&to=${encodeURIComponent(windowTo.value)}`
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

async function loadNotificationSubscription() {
  if (!props.project?.id) return

  notificationsLoading.value = true
  notificationsError.value = ''
  notificationsAvailable.value = false
  notificationsSubscribed.value = false

  try {
    const meResp = await fetch('/api/me', {
      credentials: 'include'
    })
    if (meResp.status === 401 || meResp.status === 403 || meResp.status === 503) {
      return
    }
    if (!meResp.ok) {
      throw new Error(await meResp.text())
    }

    notificationsAvailable.value = true
    const subResp = await fetch(`/api/projects/${props.project.id}/notifications/subscription`, {
      credentials: 'include'
    })
    if (!subResp.ok) {
      throw new Error(await subResp.text())
    }
    const data = await subResp.json()
    notificationsSubscribed.value = !!data.subscribed
  } catch (e) {
    notificationsError.value = e.message || 'Не вдалося завантажити налаштування сповіщень.'
  } finally {
    notificationsLoading.value = false
  }
}

async function toggleNotificationSubscription() {
  if (!props.project?.id || !notificationsAvailable.value) return

  const nextValue = !notificationsSubscribed.value
  notificationsSaving.value = true
  notificationsError.value = ''

  try {
    const resp = await fetch(`/api/projects/${props.project.id}/notifications/subscription`, {
      method: 'POST',
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        subscribed: nextValue
      })
    })
    if (!resp.ok) {
      throw new Error(await resp.text())
    }
    const data = await resp.json()
    notificationsSubscribed.value = !!data.subscribed
  } catch (e) {
    notificationsError.value = e.message || 'Не вдалося оновити підписку на сповіщення.'
  } finally {
    notificationsSaving.value = false
  }
}

watch(() => props.project?.id, (nextId, prevId) => {
  if (nextId && nextId !== prevId) {
    loadAvailability()
    loadNotificationSubscription()
  }
})

onMounted(() => {
  loadAvailability()
  loadNotificationSubscription()
  document.addEventListener('pointerdown', handleOutsidePointerDown)
  window.addEventListener('auth-changed', handleAuthChanged)
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', handleOutsidePointerDown)
  window.removeEventListener('auth-changed', handleAuthChanged)
})
</script>

<template>
  <header class="hero">
    <div>
      <h1>{{ project.name }}</h1>
      <p class="sub">{{ projectSubtitle }}</p>
    </div>
    <div class="project-notification-controls">
      <button
        v-if="notificationsAvailable"
        :class="['secondary-btn', notificationsSubscribed ? 'notify-unsubscribe' : 'notify-subscribe']"
        type="button"
        :disabled="notificationsLoading || notificationsSaving"
        @click="toggleNotificationSubscription"
      >
        {{
          notificationsSaving
            ? 'Оновлення…'
            : (notificationsSubscribed ? '🔕 Відписатись' : '🔔 Підписатись на оновлення')
        }}
      </button>
      <button
        v-else
        class="secondary-btn"
        type="button"
        disabled
        title="Потрібно увійти через Telegram"
      >
        🔔 Підписатись на оновлення
      </button>
      <p v-if="!notificationsAvailable && !notificationsLoading" class="sub">
        Увійдіть через Telegram, щоб керувати сповіщеннями.
      </p>
      <p v-if="notificationsError" class="error notification-error">{{ notificationsError }}</p>
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
</template>
