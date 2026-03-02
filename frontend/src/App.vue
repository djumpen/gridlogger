<script setup>
import { computed, onMounted, ref } from 'vue'
import AppTopbar from './components/AppTopbar.vue'
import LandingView from './components/LandingView.vue'
import ProjectDashboard from './components/ProjectDashboard.vue'
import { useProjectCatalog } from './composables/useProjectCatalog'
import { parseRouteSlug } from './utils/route'

const routeSlug = ref('')
const isLandingPage = computed(() => routeSlug.value === '')

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

onMounted(async () => {
  routeSlug.value = parseRouteSlug()

  if (isLandingPage.value) {
    await loadProjectsList()
    return
  }

  await loadProjectBySlug(routeSlug.value)
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

    <template v-else>
      <p v-if="projectLoading">Завантаження проєкту…</p>
      <p v-else-if="projectError" class="error">{{ projectError }}</p>
      <ProjectDashboard v-else-if="currentProject" :project="currentProject" />
    </template>
  </main>
</template>
