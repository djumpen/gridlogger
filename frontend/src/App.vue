<script setup>
import { computed, onMounted, ref } from 'vue'
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

function formatForInput(date) {
  const p = (n) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${p(date.getMonth() + 1)}-${p(date.getDate())}`
}

const selectedDate = ref(formatForInput(new Date()))

const viewOptions = [
  { label: 'Day', value: 'day' },
  { label: 'Week', value: 'week' },
  { label: 'Month', value: 'month' }
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

const currentStatus = computed(() => {
  if (!windowFrom.value || !windowTo.value || !intervals.value.length) {
    return 'unknown'
  }

  const now = new Date()
  const from = new Date(windowFrom.value)
  const to = new Date(windowTo.value)
  if (now < from || now >= to) {
    return 'outside window'
  }

  for (const iv of intervals.value) {
    const start = new Date(iv.start)
    const end = new Date(iv.end)
    if (now >= start && now < end) {
      return iv.status
    }
  }

  return 'unknown'
})

function computeWindow() {
  const base = new Date(`${selectedDate.value}T00:00:00`)
  if (Number.isNaN(base.getTime())) {
    throw new Error('Invalid date')
  }
  let from = new Date(base)
  let to = new Date(base)

  if (view.value === 'day') {
    to.setDate(to.getDate() + 1)
  }
  if (view.value === 'week') {
    const day = from.getDay() || 7
    from.setDate(from.getDate() - (day - 1))
    to = new Date(from)
    to.setDate(to.getDate() + 7)
  }
  if (view.value === 'month') {
    from = new Date(base.getFullYear(), base.getMonth(), 1)
    to = new Date(base.getFullYear(), base.getMonth() + 1, 1)
  }

  return { from, to }
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
  } catch (e) {
    error.value = e.message || 'Failed to load availability'
  } finally {
    loading.value = false
  }
}

onMounted(loadAvailability)
</script>

<template>
  <main class="page">
    <header class="hero">
      <div>
        <p class="kicker">Svitlo.🏘️</p>
        <h1>Power Availability Calendar</h1>
        <p class="sub">Project {{ projectId }} · TZ {{ timezone }} · outage if no ping for 2 minutes</p>
      </div>
      <div class="controls">
        <input v-model="selectedDate" type="date" @change="loadAvailability" />
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
    </header>

    <section class="stats">
      <article>
        <h2 :class="`status-${currentStatus.replace(' ', '-')}`">{{ currentStatus }}</h2>
        <p>Current status</p>
      </article>
      <article>
        <h2>{{ stats.availabilityPercent.toFixed(1) }}%</h2>
        <p>Availability in visible window</p>
      </article>
      <article>
        <h2>{{ stats.totalAvailableHours.toFixed(1) }} h</h2>
        <p>Total available</p>
      </article>
      <article>
        <h2>{{ stats.totalOutageHours.toFixed(1) }} h</h2>
        <p>Total outage</p>
      </article>
    </section>

    <section class="calendar" v-if="!loading && !error">
      <header>
        <h3>Intervals</h3>
        <p>{{ new Date(windowFrom).toLocaleDateString() }} → {{ new Date(windowTo).toLocaleDateString() }}</p>
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
      <p v-else>No intervals yet. Send pings to start tracking.</p>
    </section>

    <p v-if="loading">Loading…</p>
    <p v-if="error" class="error">{{ error }}</p>
  </main>
</template>
