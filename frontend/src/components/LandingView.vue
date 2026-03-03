<script setup>
import { ref, watch } from 'vue'

const props = defineProps({
  projects: {
    type: Array,
    required: true
  },
  projectsLoading: {
    type: Boolean,
    required: true
  },
  projectsError: {
    type: String,
    required: true
  }
})

const projectStatuses = ref({})
let statusLoadSeq = 0

watch(
  () => [props.projects, props.projectsLoading, props.projectsError],
  () => {
    if (props.projectsLoading || props.projectsError) return
    void loadCurrentStatuses()
  },
  { immediate: true, deep: true }
)

async function loadCurrentStatuses() {
  const list = Array.isArray(props.projects) ? props.projects : []
  if (!list.length) {
    projectStatuses.value = {}
    return
  }

  const seq = ++statusLoadSeq
  const now = new Date()
  const from = new Date(now.getTime() - (3 * 60 * 60 * 1000)).toISOString()
  const to = new Date(now.getTime() + (60 * 1000)).toISOString()

  const statuses = await Promise.all(
    list.map(async (project) => {
      const projectID = Number(project?.id || 0)
      if (!projectID) return [projectID, 'unknown']

      try {
        const url = `/api/projects/${projectID}/availability?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`
        const resp = await fetch(url)
        if (!resp.ok) return [projectID, 'unknown']

        const data = await resp.json()
        const nowTs = now.getTime()
        const hit = (data.intervals || []).find((iv) => {
          const start = new Date(iv.start).getTime()
          const end = new Date(iv.end).getTime()
          return nowTs >= start && nowTs < end
        })
        return [projectID, hit?.status || 'unknown']
      } catch {
        return [projectID, 'unknown']
      }
    })
  )

  if (seq !== statusLoadSeq) return
  const next = {}
  for (const [projectID, status] of statuses) {
    if (projectID > 0) next[projectID] = status
  }
  projectStatuses.value = next
}

</script>

<template>
  <section class="landing">
    <header class="hero">
      <div class="landing-intro">
        <ul class="landing-bullets">
          <li>Оберіть адресу зі списку, щоб переглянути стан електропостачання</li>
          <li>Увійдіть через Telegram, щоб додати свою адресу</li>
          <li>Підпишіться на оновлення щоб отримувати повідомлення про зміну статусу в Telegram</li>
        </ul>
      </div>
    </header>

    <p v-if="projectsLoading">Завантаження…</p>
    <p v-else-if="projectsError" class="error">{{ projectsError }}</p>
    <template v-else-if="projects.length">
      <h2 class="landing-list-title">Доступні адреси</h2>
      <ul class="project-list">
      <li v-for="project in projects" :key="project.id">
        <a :href="`/${project.slug}`" class="project-link">
          <span class="project-main-meta">
            <span class="landing-status-dot" :class="`status-${projectStatuses[project.id] || 'unknown'}`" aria-hidden="true"></span>
            <span class="project-name">{{ project.name }}</span>
          </span>
          <span class="project-city" v-if="project.city">м. {{ project.city }}</span>
        </a>
      </li>
      </ul>
    </template>
    <p v-else class="sub">Поки що проєктів немає.</p>
  </section>
</template>
