<script setup>
import { computed, ref, watch } from 'vue'

const props = defineProps({
  schedule: {
    type: Object,
    required: true
  },
  title: {
    type: String,
    default: 'Графік відключень'
  },
  subtitle: {
    type: String,
    default: ''
  }
})

const selectedDayKey = ref('')
const kyivTime = new Intl.DateTimeFormat('en-CA', {
  timeZone: 'Europe/Kyiv',
  year: 'numeric',
  month: '2-digit',
  day: '2-digit'
})
const kyivHour = new Intl.DateTimeFormat('en-GB', {
  timeZone: 'Europe/Kyiv',
  hour: '2-digit',
  hour12: false
})

const days = computed(() => Array.isArray(props.schedule?.days) ? props.schedule.days : [])
const activeDay = computed(() => {
  const current = days.value.find((item) => item.key === selectedDayKey.value)
  return current || days.value[0] || null
})
const hourlySlots = computed(() => {
  if (!activeDay.value) return []
  return Array.from({ length: 24 }, (_, hour) => {
    const startMinute = hour * 60
    const endMinute = startMinute + 60
    const slots = Array.isArray(activeDay.value.slots) ? activeDay.value.slots : []
    const definite = slots.some((slot) => slot.type === 'Definite' && overlaps(slot.startMinute, slot.endMinute, startMinute, endMinute))
    const available = slots.some((slot) => slot.type === 'NotPlanned' && overlaps(slot.startMinute, slot.endMinute, startMinute, endMinute))
    let status = 'unknown'
    if (definite) status = 'outage'
    if (!definite && available) status = 'available'

    return {
      hour,
      label: `${String(hour).padStart(2, '0')}:00`,
      status,
      isCurrentHour: isCurrentHour(activeDay.value.date, hour)
    }
  })
})
const updatedLabel = computed(() => {
  const raw = props.schedule?.updatedAt
  if (!raw) return ''
  const parsed = new Date(raw)
  if (Number.isNaN(parsed.getTime())) return ''
  return new Intl.DateTimeFormat('uk-UA', {
    timeZone: 'Europe/Kyiv',
    day: '2-digit',
    month: 'long',
    hour: '2-digit',
    minute: '2-digit'
  }).format(parsed)
})

watch(days, (nextDays) => {
  if (!nextDays.length) {
    selectedDayKey.value = ''
    return
  }
  if (nextDays.some((item) => item.key === selectedDayKey.value)) return
  selectedDayKey.value = nextDays[0].key
}, { immediate: true })

function overlaps(aStart, aEnd, bStart, bEnd) {
  return aStart < bEnd && aEnd > bStart
}

function isCurrentHour(dateString, hour) {
  if (!dateString) return false
  const today = kyivTime.format(new Date())
  if (today !== dateString) return false
  const currentHour = Number(kyivHour.format(new Date()))
  return currentHour === hour
}
</script>

<template>
  <article class="outage-card">
    <header class="outage-card-header">
      <div>
        <p class="outage-card-kicker">{{ title }}</p>
        <h3>{{ schedule.address || subtitle || 'Підібрана адреса' }}</h3>
        <p v-if="subtitle" class="sub outage-card-subtitle">{{ subtitle }}</p>
      </div>
      <div class="outage-card-meta">
        <span class="outage-group-badge">Група {{ schedule.group }}</span>
        <span v-if="updatedLabel" class="helper-text">Оновлено {{ updatedLabel }}</span>
      </div>
    </header>

    <div v-if="days.length" class="outage-day-tabs" role="tablist" aria-label="Дні графіка">
      <button
        v-for="day in days"
        :key="day.key"
        type="button"
        :class="['outage-day-tab', { active: day.key === activeDay?.key }]"
        @click="selectedDayKey = day.key"
      >
        <span>{{ day.weekdayShort }}</span>
        <small>{{ day.label }}</small>
      </button>
    </div>

    <div v-if="activeDay" class="outage-grid">
      <div
        v-for="slot in hourlySlots"
        :key="slot.hour"
        :class="['outage-hour-cell', `is-${slot.status}`, { 'is-current': slot.isCurrentHour }]"
      >
        <span class="outage-hour-icon" aria-hidden="true">
          {{ slot.status === 'outage' ? '✕' : '⚡' }}
        </span>
        <span>{{ slot.label }}</span>
      </div>
    </div>

    <div class="outage-legend">
      <span><i class="legend-dot is-current"></i> поточний час</span>
      <span><i class="legend-dot is-available"></i> є світло</span>
      <span><i class="legend-dot is-outage"></i> немає світла</span>
      <span><i class="legend-dot is-unknown"></i> графік ще очікується</span>
    </div>
  </article>
</template>
