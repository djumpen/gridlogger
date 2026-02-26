<script setup>
import { computed } from 'vue'

const props = defineProps({
  intervals: {
    type: Array,
    required: true
  },
  from: {
    type: String,
    required: true
  },
  to: {
    type: String,
    required: true
  },
  showLabels: {
    type: Boolean,
    default: true
  }
})

const fromTs = computed(() => new Date(props.from).getTime())
const toTs = computed(() => new Date(props.to).getTime())
const total = computed(() => toTs.value - fromTs.value)

const hourMarkers = computed(() => {
  if (total.value <= 0) return []
  const markers = []
  for (let ts = fromTs.value + 60 * 60 * 1000; ts < toTs.value; ts += 60 * 60 * 1000) {
    markers.push(ts)
  }
  return markers
})

const hourLabels = computed(() => {
  if (total.value <= 0) return []
  const labels = [{ ts: fromTs.value, label: '00' }]
  for (let ts = fromTs.value + 6 * 60 * 60 * 1000; ts <= toTs.value; ts += 6 * 60 * 60 * 1000) {
    const hoursFromStart = Math.round((ts - fromTs.value) / (60 * 60 * 1000))
    labels.push({
      ts,
      label: String(Math.max(0, Math.min(24, hoursFromStart))).padStart(2, '0')
    })
  }
  if (labels[labels.length - 1]?.ts !== toTs.value) {
    labels.push({ ts: toTs.value, label: '24' })
  }
  return labels
})

function positionStyle(ts) {
  const left = ((ts - fromTs.value) / total.value) * 100
  return {
    left: `${Math.max(0, Math.min(100, left))}%`
  }
}

function segmentStyle(interval) {
  const start = new Date(interval.start).getTime()
  const end = new Date(interval.end).getTime()
  const left = ((start - fromTs.value) / total.value) * 100
  const width = ((end - start) / total.value) * 100
  return {
    left: `${Math.max(0, left)}%`,
    width: `${Math.max(0.2, width)}%`
  }
}

function formatTime(value) {
  return new Intl.DateTimeFormat('en-GB', {
    hour: '2-digit',
    minute: '2-digit',
    hour12: false
  }).format(new Date(value))
}

function formatDuration(startValue, endValue) {
  const durationMs = Math.max(0, new Date(endValue).getTime() - new Date(startValue).getTime())
  const totalMinutes = Math.floor(durationMs / (60 * 1000))
  const hours = Math.floor(totalMinutes / 60)
  const minutes = totalMinutes % 60
  return `${String(hours).padStart(2, '0')}:${String(minutes).padStart(2, '0')}`
}

function segmentTitle(interval) {
  return `${interval.status}: ${formatTime(interval.start)} -> ${formatTime(interval.end)} (${formatDuration(interval.start, interval.end)})`
}
</script>

<template>
  <div class="interval-bar-wrap">
    <div class="interval-bar">
      <div class="hour-grid">
        <span
          v-for="(marker, idx) in hourMarkers"
          :key="idx"
          class="hour-marker"
          :style="positionStyle(marker)"
        ></span>
      </div>
      <div
        v-for="(interval, idx) in intervals"
        :key="idx"
        class="segment"
        :class="interval.status"
        :style="segmentStyle(interval)"
        :title="segmentTitle(interval)"
      ></div>
    </div>
    <div v-if="showLabels" class="hour-labels">
      <span
        v-for="(item, idx) in hourLabels"
        :key="`lbl-${idx}`"
        class="hour-label"
        :style="positionStyle(item.ts)"
      >
        {{ item.label }}
      </span>
    </div>
  </div>
</template>
