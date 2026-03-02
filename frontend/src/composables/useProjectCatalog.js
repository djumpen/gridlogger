import { ref } from 'vue'

export function useProjectCatalog() {
  const projects = ref([])
  const projectsLoading = ref(false)
  const projectsError = ref('')

  const currentProject = ref(null)
  const projectLoading = ref(false)
  const projectError = ref('')

  async function loadProjectsList() {
    try {
      projectsLoading.value = true
      projectsError.value = ''

      const resp = await fetch('/api/projects')
      if (!resp.ok) {
        throw new Error(await resp.text())
      }

      const data = await resp.json()
      projects.value = Array.isArray(data.projects) ? data.projects : []
    } catch (e) {
      projectsError.value = e.message || 'Не вдалося завантажити проєкти'
      projects.value = []
    } finally {
      projectsLoading.value = false
    }
  }

  async function loadProjectBySlug(slug) {
    try {
      projectLoading.value = true
      projectError.value = ''
      currentProject.value = null

      const resp = await fetch(`/api/project-slugs/${encodeURIComponent(slug)}`)
      if (resp.status === 404) {
        projectError.value = 'Проєкт не знайдено'
        return false
      }
      if (!resp.ok) {
        throw new Error(await resp.text())
      }

      const data = await resp.json()
      currentProject.value = data.project || null
      if (!currentProject.value?.id) {
        projectError.value = 'Проєкт не знайдено'
        return false
      }
      return true
    } catch (e) {
      projectError.value = e.message || 'Не вдалося завантажити проєкт'
      return false
    } finally {
      projectLoading.value = false
    }
  }

  return {
    projects,
    projectsLoading,
    projectsError,
    currentProject,
    projectLoading,
    projectError,
    loadProjectsList,
    loadProjectBySlug
  }
}
