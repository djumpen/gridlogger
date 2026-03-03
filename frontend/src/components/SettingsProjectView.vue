<script setup>
import { computed, onMounted, ref, watch } from 'vue'

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
  slug: '',
  isPublic: true
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
const firmwareSSID = ref('')
const firmwarePassword = ref('')
const firmwarePreparing = ref(false)
const firmwareError = ref('')
const firmwareHint = ref('')
const firmwareManifestURL = ref('')
const firmwareScriptReady = ref(false)
const firmwareScriptLoading = ref(false)

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

watch(activeTab, (tab) => {
  if (tab === 'firmware') {
    void ensureEspWebInstallButton()
  }
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
      slug: project.value?.slug || '',
      isPublic: project.value?.isPublic ?? true
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
        slug,
        isPublic: !!form.value.isPublic
      })
    })

    if (!resp.ok) {
      throw new Error(await resp.text())
    }

    const data = await resp.json()
    project.value = data.project || project.value
    form.value.slug = project.value?.slug || slug
    form.value.isPublic = project.value?.isPublic ?? form.value.isPublic
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

async function ensureEspWebInstallButton() {
  if (firmwareScriptReady.value || window.customElements?.get('esp-web-install-button')) {
    firmwareScriptReady.value = true
    return
  }
  if (firmwareScriptLoading.value) return

  firmwareScriptLoading.value = true
  firmwareError.value = ''

  try {
    await new Promise((resolve, reject) => {
      const script = document.createElement('script')
      script.type = 'module'
      script.src = 'https://unpkg.com/esp-web-tools@10/dist/web/install-button.js?module'
      script.onload = resolve
      script.onerror = () => reject(new Error('Не вдалося завантажити модуль прошивання ESP.'))
      document.head.appendChild(script)
    })
    firmwareScriptReady.value = !!window.customElements?.get('esp-web-install-button')
  } catch (e) {
    firmwareError.value = e.message || 'Не вдалося підготувати інструмент прошивання.'
  } finally {
    firmwareScriptLoading.value = false
  }
}

function serialSupported() {
  return window.isSecureContext && typeof navigator !== 'undefined' && !!navigator.serial
}

async function prepareFirmware() {
  if (!project.value?.id || firmwarePreparing.value) return
  const ssid = String(firmwareSSID.value || '').trim()
  const password = String(firmwarePassword.value || '').trim()
  if (!ssid || !password) {
    firmwareError.value = 'Вкажіть SSID та пароль Wi-Fi.'
    return
  }
  if (!serialSupported()) {
    firmwareError.value = 'Потрібен Chrome у захищеному HTTPS-контексті з підтримкою Web Serial.'
    return
  }

  firmwarePreparing.value = true
  firmwareError.value = ''
  firmwareHint.value = 'Збираємо прошивку для вашого проєкту…'
  firmwareManifestURL.value = ''

  try {
    const startResp = await fetch(`/api/settings/projects/${props.projectId}/firmware/jobs`, {
      method: 'POST',
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        ssid,
        password
      })
    })
    if (!startResp.ok) {
      throw new Error(await startResp.text())
    }
    const started = await startResp.json()
    const jobID = started?.job?.id
    if (!jobID) {
      throw new Error('Сервер не повернув ідентифікатор задачі прошивки.')
    }

    for (let attempt = 0; attempt < 120; attempt += 1) {
      await new Promise((resolve) => window.setTimeout(resolve, 1500))
      const statusResp = await fetch(`/api/settings/projects/${props.projectId}/firmware/jobs/${encodeURIComponent(jobID)}`, {
        credentials: 'include',
        cache: 'no-store',
        headers: {
          'Cache-Control': 'no-cache'
        }
      })
      if (!statusResp.ok) {
        throw new Error(await statusResp.text())
      }
      const payload = await statusResp.json()
      const job = payload?.job || {}
      if (job.status === 'failed') {
        throw new Error(job.error || 'Не вдалося зібрати прошивку.')
      }
      if (job.status === 'succeeded' && job.manifestUrl) {
        firmwareManifestURL.value = job.manifestUrl
        firmwareHint.value = 'Прошивка готова. Натисніть кнопку нижче та оберіть ваш ESP32-C3 у Chrome.'
        await ensureEspWebInstallButton()
        return
      }
      firmwareHint.value = `Збірка прошивки… (${attempt + 1})`
    }

    throw new Error('Збірка триває занадто довго. Спробуйте ще раз.')
  } catch (e) {
    firmwareError.value = e.message || 'Не вдалося підготувати прошивку.'
    firmwareHint.value = ''
    firmwareManifestURL.value = ''
  } finally {
    firmwarePreparing.value = false
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
        <button class="settings-tab-btn" :class="{ active: activeTab === 'firmware' }" type="button" @click="activeTab = 'firmware'">
          Прошивка ESP32
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
        <label class="field-checkbox">
          <input v-model="form.isPublic" type="checkbox" />
          <span>Показувати в загальному списку</span>
        </label>

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

      <section v-else-if="activeTab === 'integration'" class="settings-form-card integration-card">
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

      <section v-else class="settings-form-card integration-card">
        <h2>Прошивка ESP32-C3</h2>
        <p class="sub">Вкажіть Wi-Fi мережу і зберіть індивідуальну прошивку для цієї адреси.</p>

        <label class="field-label" for="firmware-ssid">Wi-Fi SSID</label>
        <input id="firmware-ssid" v-model="firmwareSSID" type="text" class="field-input" autocomplete="off" />

        <label class="field-label" for="firmware-password">Wi-Fi пароль</label>
        <input id="firmware-password" v-model="firmwarePassword" type="password" class="field-input" autocomplete="new-password" />

        <div class="settings-form-actions">
          <button class="primary-btn" type="button" :disabled="firmwarePreparing" @click="prepareFirmware">
            {{ firmwarePreparing ? 'Підготовка…' : 'Компілювати прошивку' }}
          </button>
        </div>

        <p class="helper-text">Працює у Chrome через USB (Web Serial). Після збірки з’явиться кнопка прошивання.</p>
        <p v-if="firmwareHint" class="sub">{{ firmwareHint }}</p>
        <p v-if="firmwareError" class="error form-error">{{ firmwareError }}</p>

        <div v-if="firmwareManifestURL && firmwareScriptReady" class="firmware-install-wrap">
          <esp-web-install-button :manifest="firmwareManifestURL"></esp-web-install-button>
        </div>
      </section>
    </template>
  </section>
</template>
