<script setup>
import { computed, onMounted, ref } from 'vue'

const projects = ref([])
const subscribedProjects = ref([])
const loading = ref(false)
const error = ref('')
const showCreate = ref(false)
const saving = ref(false)
const saveError = ref('')
const activeTab = ref('subscriptions')
const subscriptionsSaving = ref({})
const subscriptionsError = ref({})

const form = ref({
  name: '',
  city: '',
  slug: '',
  isPublic: true
})

const siteHost = window.location.host || 'svitlo.homes'
const slugPattern = /^[a-z0-9-]+$/
const reservedSlug = 'api'

const slugError = computed(() => {
  const slug = String(form.value.slug || '').trim().toLowerCase()
  if (slug.length === 0) return ''
  if (slug.length < 3) return 'Коротка назва має містити щонайменше 3 символи.'
  if (!slugPattern.test(slug)) return 'Дозволені лише малі латинські літери, цифри та дефіс.'
  if (slug === reservedSlug) return 'Коротка назва "api" зарезервована.'
  return ''
})

onMounted(() => {
  loadSettings()
})

async function loadSettings() {
  try {
    loading.value = true
    error.value = ''

    const resp = await fetch('/api/settings', {
      credentials: 'include'
    })

    if (resp.status === 401) {
      error.value = 'Увійдіть через Telegram, щоб керувати адресами.'
      projects.value = []
      return
    }
    if (!resp.ok) {
      throw new Error(await resp.text())
    }

    const data = await resp.json()
    projects.value = Array.isArray(data.projects) ? data.projects : []
    await loadSubscribedProjects()
  } catch (e) {
    error.value = e.message || 'Не вдалося завантажити ваші адреси.'
  } finally {
    loading.value = false
  }
}

async function loadSubscribedProjects() {
  try {
    const resp = await fetch('/api/settings/subscriptions', {
      credentials: 'include',
      cache: 'no-store',
      headers: {
        'Cache-Control': 'no-cache'
      }
    })
    if (!resp.ok) {
      throw new Error(await resp.text())
    }
    const data = await resp.json()
    subscribedProjects.value = Array.isArray(data.projects) ? data.projects : []
  } catch (e) {
    subscribedProjects.value = []
    error.value = e.message || 'Не вдалося завантажити ваші підписки.'
  }
}

async function unsubscribeFromProject(projectID) {
  if (!projectID || subscriptionsSaving.value[projectID]) return
  subscriptionsSaving.value = { ...subscriptionsSaving.value, [projectID]: true }
  subscriptionsError.value = { ...subscriptionsError.value, [projectID]: '' }

  try {
    const resp = await fetch(`/api/projects/${projectID}/notifications/subscription`, {
      method: 'POST',
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        subscribed: false
      })
    })
    if (!resp.ok) {
      throw new Error(await resp.text())
    }
    subscribedProjects.value = subscribedProjects.value.filter((project) => project.id !== projectID)
  } catch (e) {
    subscriptionsError.value = { ...subscriptionsError.value, [projectID]: e.message || 'Не вдалося відписатись.' }
  } finally {
    subscriptionsSaving.value = { ...subscriptionsSaving.value, [projectID]: false }
  }
}

function toggleCreateForm() {
  showCreate.value = !showCreate.value
  saveError.value = ''
}

async function createProject() {
  const name = String(form.value.name || '').trim()
  const city = String(form.value.city || '').trim()
  const slug = String(form.value.slug || '').trim().toLowerCase()

  if (!name || !city || !slug) {
    saveError.value = 'Заповніть усі поля.'
    return
  }
  if (slug.length < 3 || !slugPattern.test(slug)) {
    saveError.value = 'Некоректна коротка назва.'
    return
  }
  if (slug === reservedSlug) {
    saveError.value = 'Коротка назва "api" зарезервована.'
    return
  }

  try {
    saving.value = true
    saveError.value = ''

    const resp = await fetch('/api/settings/projects', {
      method: 'POST',
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        name,
        city,
        slug,
        isPublic: !!form.value.isPublic
      })
    })

    if (!resp.ok) {
      throw new Error(await resp.text())
    }

    const data = await resp.json()
    const fallback = data?.project?.id ? `/a/settings/project/${data.project.id}` : '/a/settings'
    window.location.href = data.redirectTo || fallback
  } catch (e) {
    saveError.value = e.message || 'Не вдалося створити проєкт.'
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <section class="landing">
    <div class="settings-tabs-row">
      <div class="settings-tabs settings-page-tabs" role="tablist" aria-label="Налаштування адрес">
        <button
          class="settings-tab-btn"
          :class="{ active: activeTab === 'subscriptions' }"
          type="button"
          role="tab"
          :aria-selected="activeTab === 'subscriptions'"
          @click="activeTab = 'subscriptions'"
        >
          Мої підписки
        </button>
        <button
          class="settings-tab-btn"
          :class="{ active: activeTab === 'projects' }"
          type="button"
          role="tab"
          :aria-selected="activeTab === 'projects'"
          @click="activeTab = 'projects'"
        >
          Керування адресами
        </button>
      </div>
      <button v-if="activeTab === 'projects'" class="primary-btn" type="button" @click="toggleCreateForm">
        Додати адресу
      </button>
    </div>

    <p class="sub settings-tab-description">
      {{ activeTab === 'subscriptions' ? 'Список адрес, на які ви підписані в Telegram.' : 'Адреси, які ви створили та можете налаштовувати.' }}
    </p>

    <section v-if="activeTab === 'projects' && showCreate" class="settings-form-card">
      <h2>Нова адреса</h2>
      <label class="field-label" for="project-name">Адреса</label>
      <input id="project-name" v-model="form.name" type="text" class="field-input" />

      <label class="field-label" for="project-city">Місто</label>
      <input id="project-city" v-model="form.city" type="text" class="field-input" />

      <label class="field-label" for="project-slug">Коротка назва</label>
      <input id="project-slug" v-model="form.slug" type="text" class="field-input" placeholder="shevchenka-7a" />
      <p class="helper-text">Цей проєкт буде доступний за адресою {{ siteHost }}/&lt;коротка-назва&gt;</p>
      <label class="field-checkbox">
        <input v-model="form.isPublic" type="checkbox" />
        <span>Показувати в загальному списку</span>
      </label>
      <p v-if="slugError" class="error form-error">{{ slugError }}</p>
      <p v-if="saveError" class="error form-error">{{ saveError }}</p>

      <div class="settings-form-actions">
        <button class="primary-btn" type="button" :disabled="saving" @click="createProject">
          {{ saving ? 'Створення…' : 'Додати адресу' }}
        </button>
      </div>
    </section>

    <p v-if="loading">Завантаження…</p>
    <p v-else-if="error" class="error">{{ error }}</p>

    <ul v-else-if="activeTab === 'subscriptions' && subscribedProjects.length" class="project-list settings-project-list">
      <li v-for="project in subscribedProjects" :key="project.id" class="settings-project-row">
        <a :href="`/${project.slug}`" class="project-link settings-project-main">
          <span class="project-name">{{ project.name }}</span>
          <span class="project-city" v-if="project.city">м. {{ project.city }}</span>
        </a>
        <div class="settings-project-actions">
          <button
            class="secondary-btn settings-project-subscribe notify-unsubscribe"
            type="button"
            :disabled="subscriptionsSaving[project.id]"
            title="Відписатись"
            @click="unsubscribeFromProject(project.id)"
          >
            <span class="settings-btn-emoji" aria-hidden="true">{{ subscriptionsSaving[project.id] ? '⏳' : '🔕' }}</span>
            <span class="settings-btn-text">{{ subscriptionsSaving[project.id] ? 'Оновлення…' : 'Відписатись' }}</span>
          </button>
        </div>
        <p v-if="subscriptionsError[project.id]" class="error settings-project-error">{{ subscriptionsError[project.id] }}</p>
      </li>
    </ul>

    <p v-else-if="activeTab === 'subscriptions'" class="sub settings-empty">У вас ще немає підписок.</p>

    <ul v-else-if="projects.length" class="project-list settings-project-list">
      <li v-for="project in projects" :key="project.id" class="settings-project-row">
        <a :href="`/${project.slug}`" class="project-link settings-project-main">
          <span class="project-name">{{ project.name }}</span>
          <span class="project-city" v-if="project.city">м. {{ project.city }}</span>
        </a>
        <div class="settings-project-actions">
          <a :href="`/a/settings/project/${project.id}`" class="secondary-btn settings-project-settings" title="Налаштування">
            <span class="settings-btn-emoji" aria-hidden="true">⚙️</span>
            <span class="settings-btn-text">Налаштування</span>
          </a>
        </div>
      </li>
    </ul>
    <p v-else class="sub settings-empty">У вас ще немає адрес. Додайте першу адресу.</p>
  </section>
</template>
