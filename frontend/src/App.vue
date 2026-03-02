<script setup>
import { computed, onMounted, ref } from 'vue'
import AppTopbar from './components/AppTopbar.vue'
import LandingView from './components/LandingView.vue'
import ProjectDashboard from './components/ProjectDashboard.vue'
import SettingsView from './components/SettingsView.vue'
import SettingsProjectView from './components/SettingsProjectView.vue'
import { useProjectCatalog } from './composables/useProjectCatalog'
import { parseAppRoute } from './utils/route'

const route = ref({ name: 'landing' })

const {
  projects,
  projectsLoading,
  projectsError,
  currentProject,
  projectLoading,
  projectError,
  loadProjectsList,
  loadProjectBySlug
} = useProjectCatalog()

const isLandingPage = computed(() => route.value.name === 'landing')
const isPublicProjectPage = computed(() => route.value.name === 'public-project')
const isSettingsPage = computed(() => route.value.name === 'settings')
const isSettingsProjectPage = computed(() => route.value.name === 'settings-project')

onMounted(async () => {
  route.value = parseAppRoute()

  if (isLandingPage.value) {
    await loadProjectsList()
    return
  }

  if (isPublicProjectPage.value) {
    await loadProjectBySlug(route.value.slug)
  }
})
</script>

<template>
  <main class="page">
    <AppTopbar />

    <LandingView
      v-if="isLandingPage"
      :projects="projects"
      :projects-loading="projectsLoading"
      :projects-error="projectsError"
    />

    <SettingsView v-else-if="isSettingsPage" />

    <SettingsProjectView
      v-else-if="isSettingsProjectPage"
      :project-id="route.projectId"
    />

    <template v-else-if="isPublicProjectPage">
      <p v-if="projectLoading">Завантаження проєкту…</p>
      <p v-else-if="projectError" class="error">{{ projectError }}</p>
      <ProjectDashboard v-else-if="currentProject" :project="currentProject" />
    </template>

    <p v-else class="error">Сторінку не знайдено.</p>
  </main>
</template>
