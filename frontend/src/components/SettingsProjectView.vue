<script setup>
import { computed, onMounted, ref } from 'vue'

const props = defineProps({
  projectId: {
    type: Number,
    required: true
  }
})

const loading = ref(false)
const error = ref('')
const project = ref(null)
const activeTab = ref('settings')

const form = ref({
  name: '',
  city: '',
  slug: ''
})

const saving = ref(false)
const saveError = ref('')
const saveSuccess = ref('')
const deleting = ref(false)
const deleteError = ref('')

const revealSecret = ref(false)
const copyingSecret = ref(false)
const copySuccess = ref('')
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

const maskedSecret = computed(() => {
  const secret = String(project.value?.secret || '')
  if (!secret) return ''
  if (secret.length <= 8) return '••••••••'
  return `${secret.slice(0, 4)}••••••••${secret.slice(-4)}`
})

const pingEndpoint = computed(() => `${window.location.origin}/api/projects/${props.projectId}/ping`)

onMounted(() => {
  loadProject()
})

async function loadProject() {
  try {
    loading.value = true
    error.value = ''

    const resp = await fetch(`/api/settings/projects/${props.projectId}`, {
      credentials: 'include'
    })

    if (resp.status === 401) {
      error.value = 'Увійдіть через Telegram, щоб керувати адресою.'
      return
    }
    if (resp.status === 403) {
      error.value = 'У вас немає доступу до цієї адреси.'
      return
    }
    if (resp.status === 404) {
      error.value = 'Адресу не знайдено.'
      return
    }
    if (!resp.ok) {
      throw new Error(await resp.text())
    }

    const data = await resp.json()
    project.value = data.project || null

    form.value = {
      name: project.value?.name || '',
      city: project.value?.city || '',
      slug: project.value?.slug || ''
    }
  } catch (e) {
    error.value = e.message || 'Не вдалося завантажити адресу.'
  } finally {
    loading.value = false
  }
}

async function saveProject() {
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
    saveSuccess.value = ''

    const resp = await fetch(`/api/settings/projects/${props.projectId}`, {
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
    project.value = data.project || project.value
    form.value.slug = project.value?.slug || slug
    saveSuccess.value = 'Зміни збережено.'
  } catch (e) {
    saveError.value = e.message || 'Не вдалося зберегти зміни.'
  } finally {
    saving.value = false
  }
}

async function copySecret() {
  if (!project.value?.secret) return

  try {
    copyingSecret.value = true
    copySuccess.value = ''
    await navigator.clipboard.writeText(project.value.secret)
    copySuccess.value = 'Скопійовано.'
  } catch {
    copySuccess.value = 'Не вдалося скопіювати.'
  } finally {
    copyingSecret.value = false
  }
}

async function deleteProject() {
  if (!project.value?.id || deleting.value) return

  const confirmed = window.confirm('Ви впевнені, що хочете видалити цю адресу? Дію неможливо скасувати.')
  if (!confirmed) return

  try {
    deleting.value = true
    deleteError.value = ''

    const resp = await fetch(`/api/settings/projects/${props.projectId}`, {
      method: 'DELETE',
      credentials: 'include'
    })
    if (!resp.ok) {
      throw new Error(await resp.text())
    }
    window.location.href = '/a/settings'
  } catch (e) {
    deleteError.value = e.message || 'Не вдалося видалити адресу.'
  } finally {
    deleting.value = false
  }
}
</script>

<template>
  <section class="landing">
    <header class="hero settings-hero">
      <div>
        <h1>{{ project?.name || `Адреса #${projectId}` }}</h1>
        <p class="sub">Керування адресою та інтеграцією</p>
      </div>
      <a class="topbar-link" href="/a/settings">← Мої адреси</a>
    </header>

    <p v-if="loading">Завантаження…</p>
    <p v-else-if="error" class="error">{{ error }}</p>

    <template v-else-if="project">
      <div class="settings-tabs">
        <button class="settings-tab-btn" :class="{ active: activeTab === 'settings' }" type="button" @click="activeTab = 'settings'">
          Налаштування
        </button>
        <button class="settings-tab-btn" :class="{ active: activeTab === 'integration' }" type="button" @click="activeTab = 'integration'">
          Інтеграція
        </button>
      </div>

      <section v-if="activeTab === 'settings'" class="settings-form-card">
        <h2>Налаштування адреси</h2>
        <label class="field-label" for="settings-name">Адреса</label>
        <input id="settings-name" v-model="form.name" type="text" class="field-input" />

        <label class="field-label" for="settings-city">Місто</label>
        <input id="settings-city" v-model="form.city" type="text" class="field-input" />

        <label class="field-label" for="settings-slug">Коротка назва</label>
        <input id="settings-slug" v-model="form.slug" type="text" class="field-input" />
        <p class="helper-text">Цей проєкт буде доступний за адресою {{ siteHost }}/&lt;slug&gt;</p>

        <p v-if="slugError" class="error form-error">{{ slugError }}</p>
        <p v-if="saveError" class="error form-error">{{ saveError }}</p>
        <p v-if="saveSuccess" class="success form-success">{{ saveSuccess }}</p>

        <div class="settings-form-actions">
          <button class="primary-btn" type="button" :disabled="saving" @click="saveProject">
            {{ saving ? 'Збереження…' : 'Зберегти' }}
          </button>
          <button class="secondary-btn danger-btn" type="button" :disabled="saving || deleting" @click="deleteProject">
            {{ deleting ? 'Видалення…' : 'Видалити адресу' }}
          </button>
        </div>
        <p v-if="deleteError" class="error form-error">{{ deleteError }}</p>
      </section>

      <section v-else class="settings-form-card integration-card">
        <h2>Інтеграція</h2>
        <p class="sub">Надсилайте ping кожні ~30 секунд, щоб система обчислювала наявність світла.</p>

        <p class="field-label">Project ID</p>
        <p class="integration-value">{{ project.id }}</p>

        <p class="field-label">Секрет проєкту</p>
        <div class="secret-row">
          <code class="integration-secret">{{ revealSecret ? project.secret : maskedSecret }}</code>
          <button class="secondary-btn" type="button" @click="revealSecret = !revealSecret">
            {{ revealSecret ? 'Сховати' : 'Показати' }}
          </button>
          <button class="secondary-btn" type="button" :disabled="copyingSecret" @click="copySecret">
            {{ copyingSecret ? 'Копіювання…' : 'Копіювати' }}
          </button>
        </div>
        <p class="warning-text">Зберігайте секрет приватним (keep it private).</p>
        <p v-if="copySuccess" class="sub">{{ copySuccess }}</p>

        <p class="field-label">Endpoint</p>
        <pre class="code-block"><code>{{ pingEndpoint }}</code></pre>

        <p class="field-label">Обовʼязкові заголовки</p>
        <pre class="code-block"><code>X-Project-Secret: {{ revealSecret ? project.secret : maskedSecret }}</code></pre>

        <p class="field-label">Приклад запиту</p>
        <pre class="code-block"><code>curl -X POST '{{ pingEndpoint }}' \
  -H 'X-Project-Secret: {{ revealSecret ? project.secret : "<your-project-secret>" }}'</code></pre>
      </section>
    </template>
  </section>
</template>
