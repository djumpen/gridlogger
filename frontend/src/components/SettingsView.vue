<script setup>
import { computed, onMounted, ref } from 'vue'

const projects = ref([])
const loading = ref(false)
const error = ref('')
const showCreate = ref(false)
const saving = ref(false)
const saveError = ref('')
const subscriptions = ref({})
const subscriptionsLoading = ref({})
const subscriptionsSaving = ref({})
const subscriptionsError = ref({})

const form = ref({
  name: '',
  city: '',
  slug: ''
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
    await loadSubscriptions()
  } catch (e) {
    error.value = e.message || 'Не вдалося завантажити ваші адреси.'
  } finally {
    loading.value = false
  }
}

async function loadSubscriptions() {
  const items = Array.isArray(projects.value) ? projects.value : []
  if (!items.length) return

  await Promise.all(items.map(async (project) => {
    const projectID = project?.id
    if (!projectID) return

    subscriptionsLoading.value = { ...subscriptionsLoading.value, [projectID]: true }
    subscriptionsError.value = { ...subscriptionsError.value, [projectID]: '' }

    try {
      const resp = await fetch(`/api/projects/${projectID}/notifications/subscription`, {
        credentials: 'include'
      })
      if (!resp.ok) {
        throw new Error(await resp.text())
      }
      const data = await resp.json()
      subscriptions.value = { ...subscriptions.value, [projectID]: !!data.subscribed }
    } catch (e) {
      subscriptionsError.value = { ...subscriptionsError.value, [projectID]: e.message || 'Не вдалося завантажити підписку.' }
      subscriptions.value = { ...subscriptions.value, [projectID]: false }
    } finally {
      subscriptionsLoading.value = { ...subscriptionsLoading.value, [projectID]: false }
    }
  }))
}

function subscriptionLabel(projectID) {
  return subscriptions.value[projectID] ? 'Відписатись' : 'Підписатись'
}

async function toggleProjectSubscription(projectID) {
  if (!projectID || subscriptionsLoading.value[projectID] || subscriptionsSaving.value[projectID]) return

  const nextValue = !subscriptions.value[projectID]
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
        subscribed: nextValue
      })
    })
    if (!resp.ok) {
      throw new Error(await resp.text())
    }
    const data = await resp.json()
    subscriptions.value = { ...subscriptions.value, [projectID]: !!data.subscribed }
  } catch (e) {
    subscriptionsError.value = { ...subscriptionsError.value, [projectID]: e.message || 'Не вдалося оновити підписку.' }
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
        slug
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
    <header class="hero settings-hero">
      <div>
        <h1>Мої адреси</h1>
        <p class="sub">Керуйте адресами, які належать вашому акаунту.</p>
      </div>
      <button class="primary-btn" type="button" @click="toggleCreateForm">
        Додати адресу
      </button>
    </header>

    <section v-if="showCreate" class="settings-form-card">
      <h2>Нова адреса</h2>
      <label class="field-label" for="project-name">Адреса</label>
      <input id="project-name" v-model="form.name" type="text" class="field-input" />

      <label class="field-label" for="project-city">Місто</label>
      <input id="project-city" v-model="form.city" type="text" class="field-input" />

      <label class="field-label" for="project-slug">Коротка назва</label>
      <input id="project-slug" v-model="form.slug" type="text" class="field-input" placeholder="shevchenka-7a" />
      <p class="helper-text">Цей проєкт буде доступний за адресою {{ siteHost }}/&lt;коротка-назва&gt;</p>
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
          <button
            class="secondary-btn settings-project-subscribe"
            :class="{ 'notify-unsubscribe': subscriptions[project.id], 'notify-subscribe': !subscriptions[project.id] }"
            type="button"
            :disabled="subscriptionsLoading[project.id] || subscriptionsSaving[project.id]"
            :title="subscriptions[project.id] ? 'Відписатись' : 'Підписатись'"
            @click="toggleProjectSubscription(project.id)"
          >
            <span class="settings-btn-emoji" aria-hidden="true">
              {{ subscriptionsSaving[project.id] ? '⏳' : (subscriptions[project.id] ? '🔕' : '🔔') }}
            </span>
            <span class="settings-btn-text">{{ subscriptionsSaving[project.id] ? 'Оновлення…' : subscriptionLabel(project.id) }}</span>
          </button>
        </div>
        <p v-if="subscriptionsError[project.id]" class="error settings-project-error">{{ subscriptionsError[project.id] }}</p>
      </li>
    </ul>
    <p v-else class="sub">У вас ще немає адрес. Додайте першу адресу.</p>
  </section>
</template>
