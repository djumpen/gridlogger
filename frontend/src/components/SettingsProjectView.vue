<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import OutageScheduleCard from './OutageScheduleCard.vue'

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
const telegramBotGroups = ref([])
const telegramBotLoading = ref(false)
const telegramBotLoaded = ref(false)
const telegramBotError = ref('')
const telegramBotTitle = ref('')
const telegramBotSaving = ref(false)
const telegramBotSaveError = ref('')
const telegramBotSuccess = ref('')
const telegramBotDeleting = ref({})
const telegramBotUsername = ref('svitlohomes_bot')
const yasnoLoading = ref(false)
const yasnoLoaded = ref(false)
const yasnoError = ref('')
const yasnoScheduleError = ref('')
const yasnoRegions = ref([])
const yasnoRegionsLoading = ref(false)
const yasnoStreetOptions = ref([])
const yasnoStreetLoading = ref(false)
const yasnoStreetSearchDone = ref(false)
const yasnoHouseOptions = ref([])
const yasnoHouseLoading = ref(false)
const yasnoHouseSearchDone = ref(false)
const yasnoPreviewLoading = ref(false)
const yasnoPreviewError = ref('')
const yasnoSaveLoading = ref(false)
const yasnoSaveError = ref('')
const yasnoSaveSuccess = ref('')
const yasnoDeleteLoading = ref(false)
const yasnoConfig = ref(null)
const yasnoSchedule = ref(null)
const yasnoSelection = ref({
  regionId: '',
  dsoId: '',
  streetQuery: '',
  houseQuery: ''
})
const yasnoSelectedStreet = ref(null)
const yasnoSelectedHouse = ref(null)
let streetDebounceTimer = null
let houseDebounceTimer = null
let streetAbortController = null
let houseAbortController = null
let streetRequestID = 0
let houseRequestID = 0

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
const telegramBotHandle = computed(() => {
  const username = String(telegramBotUsername.value || '').trim().replace(/^@+/, '')
  return username ? `@${username}` : '@svitlohomes_bot'
})
const telegramBotStartGroupLink = computed(() => {
  const username = String(telegramBotUsername.value || '').trim().replace(/^@+/, '')
  return username ? `https://t.me/${username}?startgroup=gridlogger` : ''
})
const yasnoSelectedRegion = computed(() => {
  const regionId = Number(yasnoSelection.value.regionId)
  return yasnoRegions.value.find((item) => item.id === regionId) || null
})
const yasnoDSOOptions = computed(() => Array.isArray(yasnoSelectedRegion.value?.dsos) ? yasnoSelectedRegion.value.dsos : [])
const yasnoSelectedDSO = computed(() => {
  const dsoId = Number(yasnoSelection.value.dsoId)
  return yasnoDSOOptions.value.find((item) => item.id === dsoId) || null
})
const yasnoCanSearchStreets = computed(() => {
  return !!yasnoSelectedRegion.value && !!yasnoSelectedDSO.value && String(yasnoSelection.value.streetQuery || '').trim().length >= 2
})
const yasnoCanSearchHouses = computed(() => {
  return !!yasnoSelectedStreet.value && String(yasnoSelection.value.houseQuery || '').trim().length >= 1
})
const yasnoCanPreview = computed(() => {
  return !!yasnoSelectedRegion.value && !!yasnoSelectedDSO.value && !!yasnoSelectedStreet.value && !!yasnoSelectedHouse.value
})
const yasnoStreetAutocompleteOpen = computed(() => {
  const query = String(yasnoSelection.value.streetQuery || '').trim()
  const selectedName = String(yasnoSelectedStreet.value?.name || '').trim()
  return !!yasnoSelectedRegion.value && !!yasnoSelectedDSO.value && query.length >= 2 && query !== selectedName
})
const yasnoHouseAutocompleteOpen = computed(() => {
  const query = String(yasnoSelection.value.houseQuery || '').trim()
  const selectedName = String(yasnoSelectedHouse.value?.name || '').trim()
  return !!yasnoSelectedStreet.value && query.length >= 1 && query !== selectedName
})

onMounted(() => {
  loadProject()
})

onBeforeUnmount(() => {
  cancelStreetAutocomplete()
  cancelHouseAutocomplete()
})

watch(activeTab, (tab) => {
  if (tab === 'yasno') {
    void loadYasnoState()
    void loadYasnoRegions()
  }
  if (tab === 'telegram-bot') {
    void loadTelegramBotGroups()
  }
  if (tab === 'firmware') {
    void ensureEspWebInstallButton()
  }
})

watch(() => yasnoSelection.value.regionId, () => {
  yasnoSelection.value.dsoId = ''
  resetYasnoStreetSelection()
  resetYasnoHouseSelection()
  clearYasnoPreviewMessages()
})

watch(() => yasnoSelection.value.dsoId, () => {
  resetYasnoStreetSelection()
  resetYasnoHouseSelection()
  clearYasnoPreviewMessages()
})

watch(() => yasnoSelection.value.streetQuery, (nextValue) => {
  const query = String(nextValue || '').trim()
  const selectedName = String(yasnoSelectedStreet.value?.name || '').trim()

  if (query !== selectedName) {
    yasnoSelectedStreet.value = null
    resetYasnoHouseSelection()
  }
  clearYasnoPreviewMessages()
  cancelStreetAutocomplete()

  if (!yasnoStreetAutocompleteOpen.value) {
    yasnoStreetLoading.value = false
    yasnoStreetSearchDone.value = false
    if (!query) {
      yasnoStreetOptions.value = []
    }
    return
  }

  yasnoStreetLoading.value = true
  yasnoStreetSearchDone.value = false
  streetDebounceTimer = window.setTimeout(() => {
    void searchYasnoStreets(query)
  }, 350)
})

watch(() => yasnoSelection.value.houseQuery, (nextValue) => {
  const query = String(nextValue || '').trim()
  const selectedName = String(yasnoSelectedHouse.value?.name || '').trim()

  if (query !== selectedName) {
    yasnoSelectedHouse.value = null
  }
  clearYasnoPreviewMessages()
  cancelHouseAutocomplete()

  if (!yasnoHouseAutocompleteOpen.value) {
    yasnoHouseLoading.value = false
    yasnoHouseSearchDone.value = false
    if (!query) {
      yasnoHouseOptions.value = []
    }
    return
  }

  yasnoHouseLoading.value = true
  yasnoHouseSearchDone.value = false
  houseDebounceTimer = window.setTimeout(() => {
    void searchYasnoHouses(query)
  }, 350)
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

function clearYasnoPreviewMessages() {
  yasnoPreviewError.value = ''
  yasnoSaveError.value = ''
  yasnoSaveSuccess.value = ''
}

function cancelStreetAutocomplete() {
  if (streetDebounceTimer) {
    window.clearTimeout(streetDebounceTimer)
    streetDebounceTimer = null
  }
  if (streetAbortController) {
    streetAbortController.abort()
    streetAbortController = null
  }
}

function cancelHouseAutocomplete() {
  if (houseDebounceTimer) {
    window.clearTimeout(houseDebounceTimer)
    houseDebounceTimer = null
  }
  if (houseAbortController) {
    houseAbortController.abort()
    houseAbortController = null
  }
}

function resetYasnoStreetSelection() {
  cancelStreetAutocomplete()
  yasnoStreetOptions.value = []
  yasnoStreetLoading.value = false
  yasnoStreetSearchDone.value = false
  yasnoSelection.value.streetQuery = ''
  yasnoSelectedStreet.value = null
}

function resetYasnoHouseSelection() {
  cancelHouseAutocomplete()
  yasnoHouseOptions.value = []
  yasnoHouseLoading.value = false
  yasnoHouseSearchDone.value = false
  yasnoSelection.value.houseQuery = ''
  yasnoSelectedHouse.value = null
}

function hydrateYasnoSelection(config) {
  if (!config) return
  yasnoSelection.value.regionId = String(config.regionId || '')
  yasnoSelection.value.dsoId = String(config.dsoId || '')
  yasnoSelectedStreet.value = config.streetId
    ? { id: config.streetId, name: config.streetName }
    : null
  yasnoSelectedHouse.value = config.houseId
    ? { id: config.houseId, name: config.houseName }
    : null
  yasnoSelection.value.streetQuery = config.streetName || ''
  yasnoSelection.value.houseQuery = config.houseName || ''
  yasnoStreetOptions.value = yasnoSelectedStreet.value ? [yasnoSelectedStreet.value] : []
  yasnoHouseOptions.value = yasnoSelectedHouse.value ? [yasnoSelectedHouse.value] : []
}

function buildYasnoPayload() {
  if (!yasnoSelectedRegion.value || !yasnoSelectedDSO.value || !yasnoSelectedStreet.value || !yasnoSelectedHouse.value) {
    return null
  }
  return {
    regionId: yasnoSelectedRegion.value.id,
    regionName: yasnoSelectedRegion.value.name,
    dsoId: yasnoSelectedDSO.value.id,
    dsoName: yasnoSelectedDSO.value.name,
    streetId: yasnoSelectedStreet.value.id,
    streetName: yasnoSelectedStreet.value.name,
    houseId: yasnoSelectedHouse.value.id,
    houseName: yasnoSelectedHouse.value.name
  }
}

async function loadYasnoState(force = false) {
  if ((yasnoLoaded.value && !force) || yasnoLoading.value) return

  try {
    yasnoLoading.value = true
    yasnoError.value = ''
    yasnoScheduleError.value = ''

    const resp = await fetch(`/api/settings/projects/${props.projectId}/yasno`, {
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
    yasnoConfig.value = data.config || null
    yasnoSchedule.value = data.schedule || null
    yasnoScheduleError.value = data.scheduleError || ''
    if (yasnoConfig.value) {
      hydrateYasnoSelection(yasnoConfig.value)
    }
    if (!yasnoConfig.value) {
      yasnoSchedule.value = null
      yasnoScheduleError.value = ''
    }
    yasnoLoaded.value = true
  } catch (e) {
    yasnoError.value = e.message || 'Не вдалося завантажити налаштування графіка.'
  } finally {
    yasnoLoading.value = false
  }
}

async function loadYasnoRegions(force = false) {
  if ((yasnoRegions.value.length && !force) || yasnoRegionsLoading.value) return

  try {
    yasnoRegionsLoading.value = true
    const resp = await fetch(`/api/settings/projects/${props.projectId}/yasno/regions`, {
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
    yasnoRegions.value = Array.isArray(data.regions) ? data.regions : []
  } catch (e) {
    yasnoError.value = e.message || 'Не вдалося завантажити список регіонів.'
  } finally {
    yasnoRegionsLoading.value = false
  }
}

async function searchYasnoStreets(queryOverride = '') {
  const query = String(queryOverride || yasnoSelection.value.streetQuery || '').trim()
  if (!yasnoSelectedRegion.value || !yasnoSelectedDSO.value || query.length < 2) return
  let controller = null
  let requestID = 0

  try {
    requestID = ++streetRequestID
    if (streetAbortController) {
      streetAbortController.abort()
    }
    const controller = new AbortController()
    streetAbortController = controller
    clearYasnoPreviewMessages()
    yasnoStreetLoading.value = true
    yasnoStreetSearchDone.value = false
    const params = new URLSearchParams({
      regionId: String(yasnoSelectedRegion.value.id),
      dsoId: String(yasnoSelectedDSO.value.id),
      query
    })
    const resp = await fetch(`/api/settings/projects/${props.projectId}/yasno/streets?${params.toString()}`, {
      credentials: 'include',
      signal: controller.signal
    })
    if (!resp.ok) {
      throw new Error(await resp.text())
    }
    const data = await resp.json()
    if (requestID !== streetRequestID) return
    yasnoStreetOptions.value = Array.isArray(data.streets) ? data.streets : []
    yasnoStreetSearchDone.value = true
  } catch (e) {
    if (e?.name === 'AbortError') return
    yasnoStreetOptions.value = []
    yasnoStreetSearchDone.value = false
    yasnoPreviewError.value = e.message || 'Не вдалося знайти вулицю.'
  } finally {
    if (streetAbortController === controller) {
      streetAbortController = null
    }
    if (requestID === streetRequestID) {
      yasnoStreetLoading.value = false
    }
  }
}

function chooseYasnoStreet(street) {
  yasnoSelectedStreet.value = street
  yasnoSelection.value.streetQuery = street.name
  resetYasnoHouseSelection()
  clearYasnoPreviewMessages()
}

async function searchYasnoHouses(queryOverride = '') {
  const query = String(queryOverride || yasnoSelection.value.houseQuery || '').trim()
  if (!yasnoSelectedRegion.value || !yasnoSelectedDSO.value || !yasnoSelectedStreet.value || query.length < 1) return
  let controller = null
  let requestID = 0

  try {
    requestID = ++houseRequestID
    if (houseAbortController) {
      houseAbortController.abort()
    }
    const controller = new AbortController()
    houseAbortController = controller
    clearYasnoPreviewMessages()
    yasnoHouseLoading.value = true
    yasnoHouseSearchDone.value = false
    const params = new URLSearchParams({
      regionId: String(yasnoSelectedRegion.value.id),
      dsoId: String(yasnoSelectedDSO.value.id),
      streetId: String(yasnoSelectedStreet.value.id),
      query
    })
    const resp = await fetch(`/api/settings/projects/${props.projectId}/yasno/houses?${params.toString()}`, {
      credentials: 'include',
      signal: controller.signal
    })
    if (!resp.ok) {
      throw new Error(await resp.text())
    }
    const data = await resp.json()
    if (requestID !== houseRequestID) return
    yasnoHouseOptions.value = Array.isArray(data.houses) ? data.houses : []
    yasnoHouseSearchDone.value = true
  } catch (e) {
    if (e?.name === 'AbortError') return
    yasnoHouseOptions.value = []
    yasnoHouseSearchDone.value = false
    yasnoPreviewError.value = e.message || 'Не вдалося знайти будинок.'
  } finally {
    if (houseAbortController === controller) {
      houseAbortController = null
    }
    if (requestID === houseRequestID) {
      yasnoHouseLoading.value = false
    }
  }
}

function chooseYasnoHouse(house) {
  yasnoSelectedHouse.value = house
  yasnoSelection.value.houseQuery = house.name
  clearYasnoPreviewMessages()
}

async function previewYasnoSchedule() {
  const payload = buildYasnoPayload()
  if (!payload || yasnoPreviewLoading.value) return

  try {
    yasnoPreviewLoading.value = true
    yasnoPreviewError.value = ''
    yasnoSaveError.value = ''
    yasnoSaveSuccess.value = ''

    const resp = await fetch(`/api/settings/projects/${props.projectId}/yasno/preview`, {
      method: 'POST',
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(payload)
    })
    if (!resp.ok) {
      throw new Error(await resp.text())
    }
    const data = await resp.json()
    yasnoConfig.value = data.config || yasnoConfig.value
    yasnoSchedule.value = data.schedule || null
    yasnoScheduleError.value = ''
  } catch (e) {
    yasnoPreviewError.value = e.message || 'Не вдалося отримати графік.'
  } finally {
    yasnoPreviewLoading.value = false
  }
}

async function saveYasnoSchedule() {
  const payload = buildYasnoPayload()
  if (!payload || yasnoSaveLoading.value) return

  try {
    yasnoSaveLoading.value = true
    yasnoSaveError.value = ''
    yasnoSaveSuccess.value = ''

    const resp = await fetch(`/api/settings/projects/${props.projectId}/yasno`, {
      method: 'POST',
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(payload)
    })
    if (!resp.ok) {
      throw new Error(await resp.text())
    }
    const data = await resp.json()
    yasnoConfig.value = data.config || null
    yasnoSchedule.value = data.schedule || null
    yasnoScheduleError.value = ''
    yasnoSaveSuccess.value = 'Графік Yasno підключено до цього проєкту.'
    project.value = {
      ...project.value,
      hasOutageSchedule: true
    }
    yasnoLoaded.value = true
  } catch (e) {
    yasnoSaveError.value = e.message || 'Не вдалося зберегти групу.'
  } finally {
    yasnoSaveLoading.value = false
  }
}

async function deleteYasnoSchedule() {
  if (!yasnoConfig.value || yasnoDeleteLoading.value) return

  const confirmed = window.confirm('Відключити графік Yasno для цього проєкту?')
  if (!confirmed) return

  try {
    yasnoDeleteLoading.value = true
    yasnoSaveError.value = ''
    yasnoSaveSuccess.value = ''
    const resp = await fetch(`/api/settings/projects/${props.projectId}/yasno`, {
      method: 'DELETE',
      credentials: 'include'
    })
    if (!resp.ok) {
      throw new Error(await resp.text())
    }
    yasnoConfig.value = null
    yasnoSchedule.value = null
    yasnoScheduleError.value = ''
    project.value = {
      ...project.value,
      hasOutageSchedule: false
    }
    resetYasnoStreetSelection()
    resetYasnoHouseSelection()
    yasnoSelection.value.regionId = ''
    yasnoSelection.value.dsoId = ''
    yasnoSaveSuccess.value = 'Графік Yasno відключено.'
  } catch (e) {
    yasnoSaveError.value = e.message || 'Не вдалося відключити графік.'
  } finally {
    yasnoDeleteLoading.value = false
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

async function loadTelegramBotGroups(force = false) {
  if ((telegramBotLoaded.value && !force) || telegramBotLoading.value) return

  try {
    telegramBotLoading.value = true
    telegramBotError.value = ''

    const resp = await fetch(`/api/settings/projects/${props.projectId}/telegram-bot/groups`, {
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
    telegramBotGroups.value = Array.isArray(data.groups) ? data.groups : []
    telegramBotUsername.value = data.botUsername || telegramBotUsername.value
    telegramBotLoaded.value = true
  } catch (e) {
    telegramBotError.value = e.message || 'Не вдалося завантажити список груп.'
  } finally {
    telegramBotLoading.value = false
  }
}

function upsertTelegramBotGroup(nextGroup) {
  const current = Array.isArray(telegramBotGroups.value) ? telegramBotGroups.value : []
  const filtered = current.filter((item) => item.virtualUserId !== nextGroup.virtualUserId)
  telegramBotGroups.value = [...filtered, nextGroup].sort((a, b) => String(a.title || '').localeCompare(String(b.title || '')))
}

async function addTelegramBotGroup() {
  const title = String(telegramBotTitle.value || '').trim()
  if (!title) {
    telegramBotSaveError.value = 'Вкажіть повну назву групи.'
    return
  }

  try {
    telegramBotSaving.value = true
    telegramBotSaveError.value = ''
    telegramBotSuccess.value = ''

    const resp = await fetch(`/api/settings/projects/${props.projectId}/telegram-bot/groups`, {
      method: 'POST',
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ title })
    })
    if (!resp.ok) {
      throw new Error(await resp.text())
    }

    const data = await resp.json()
    if (data.group) {
      upsertTelegramBotGroup(data.group)
    }
    telegramBotTitle.value = ''
    telegramBotLoaded.value = true
    telegramBotSuccess.value = 'Групу підключено до Telegram сповіщень цього проєкту.'
  } catch (e) {
    telegramBotSaveError.value = e.message || 'Не вдалося підключити групу.'
  } finally {
    telegramBotSaving.value = false
  }
}

async function removeTelegramBotGroup(group) {
  if (!group?.virtualUserId || telegramBotDeleting.value[group.virtualUserId]) return

  const confirmed = window.confirm(`Видалити групу "${group.title}" зі сповіщень цього проєкту?`)
  if (!confirmed) return

  try {
    telegramBotDeleting.value = { ...telegramBotDeleting.value, [group.virtualUserId]: true }
    telegramBotError.value = ''
    telegramBotSuccess.value = ''

    const resp = await fetch(`/api/settings/projects/${props.projectId}/telegram-bot/groups/${group.virtualUserId}`, {
      method: 'DELETE',
      credentials: 'include'
    })
    if (!resp.ok) {
      throw new Error(await resp.text())
    }

    telegramBotGroups.value = telegramBotGroups.value.filter((item) => item.virtualUserId !== group.virtualUserId)
    telegramBotSuccess.value = 'Групу відключено.'
  } catch (e) {
    telegramBotError.value = e.message || 'Не вдалося видалити групу.'
  } finally {
    telegramBotDeleting.value = { ...telegramBotDeleting.value, [group.virtualUserId]: false }
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
        <button class="settings-tab-btn" :class="{ active: activeTab === 'yasno' }" type="button" @click="activeTab = 'yasno'">
          Графік відключень
        </button>
        <button class="settings-tab-btn" :class="{ active: activeTab === 'telegram-bot' }" type="button" @click="activeTab = 'telegram-bot'">
          Телеграм бот
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

      <section v-else-if="activeTab === 'yasno'" class="settings-form-card integration-card yasno-settings-card">
        <h2>Графік відключень</h2>
        <p class="sub">Підберіть вашу адресу в Yasno, перевірте сьогоднішній графік та підтвердьте знайдену групу.</p>

        <div class="yasno-steps">
          <div class="yasno-step-field">
            <label class="field-label" for="yasno-region">1. Оберіть регіон</label>
            <select id="yasno-region" v-model="yasnoSelection.regionId" class="field-input" :disabled="yasnoRegionsLoading">
              <option value="">Оберіть регіон</option>
              <option v-for="region in yasnoRegions" :key="region.id" :value="String(region.id)">{{ region.name }}</option>
            </select>
          </div>

          <div class="yasno-step-field">
            <label class="field-label" for="yasno-dso">2. Оберіть оператора</label>
            <select id="yasno-dso" v-model="yasnoSelection.dsoId" class="field-input" :disabled="!yasnoSelectedRegion">
              <option value="">Оберіть оператора</option>
              <option v-for="item in yasnoDSOOptions" :key="item.id" :value="String(item.id)">{{ item.name }}</option>
            </select>
          </div>

          <div class="yasno-step-field">
            <label class="field-label" for="yasno-street">3. Знайдіть вулицю</label>
            <div class="yasno-autocomplete">
              <input
                id="yasno-street"
                v-model="yasnoSelection.streetQuery"
                type="text"
                class="field-input"
                :disabled="!yasnoSelectedRegion || !yasnoSelectedDSO"
                autocomplete="off"
                placeholder="Почніть вводити назву вулиці"
              />
              <div v-if="yasnoStreetAutocompleteOpen" class="yasno-autocomplete-panel">
                <p v-if="yasnoStreetLoading" class="yasno-autocomplete-status">Шукаємо вулиці…</p>
                <div v-else-if="yasnoStreetOptions.length" class="yasno-autocomplete-list">
                  <button
                    v-for="street in yasnoStreetOptions"
                    :key="street.id"
                    type="button"
                    class="yasno-autocomplete-option"
                    @click="chooseYasnoStreet(street)"
                  >
                    {{ street.name }}
                  </button>
                </div>
                <p v-else-if="yasnoStreetSearchDone" class="yasno-autocomplete-empty">
                  Вулицю не знайдено. Спробуйте іншу форму назви або менше слів.
                </p>
              </div>
            </div>
            <p v-if="yasnoSelectedStreet" class="helper-text yasno-selected-option">Обрано: {{ yasnoSelectedStreet.name }}</p>
            <p v-else class="helper-text">Вводьте назву, а список підтягнеться автоматично.</p>
          </div>

          <div class="yasno-step-field">
            <label class="field-label" for="yasno-house">4. Знайдіть будинок</label>
            <div class="yasno-autocomplete">
              <input
                id="yasno-house"
                v-model="yasnoSelection.houseQuery"
                type="text"
                class="field-input"
                :disabled="!yasnoSelectedStreet"
                autocomplete="off"
                placeholder="Почніть вводити номер будинку"
              />
              <div v-if="yasnoHouseAutocompleteOpen" class="yasno-autocomplete-panel">
                <p v-if="yasnoHouseLoading" class="yasno-autocomplete-status">Шукаємо будинок…</p>
                <div v-else-if="yasnoHouseOptions.length" class="yasno-autocomplete-list">
                  <button
                    v-for="house in yasnoHouseOptions"
                    :key="house.id"
                    type="button"
                    class="yasno-autocomplete-option"
                    @click="chooseYasnoHouse(house)"
                  >
                    {{ house.name }}
                  </button>
                </div>
                <p v-else-if="yasnoHouseSearchDone" class="yasno-autocomplete-empty">
                  Будинок не знайдено. Перевірте формат номера або спробуйте коротший запит.
                </p>
              </div>
            </div>
            <p v-if="yasnoSelectedHouse" class="helper-text yasno-selected-option">Обрано: {{ yasnoSelectedHouse.name }}</p>
            <p v-else class="helper-text">
              {{ yasnoSelectedStreet ? 'Вводьте номер будинку, і варіанти з’являться автоматично.' : 'Пошук будинку стане доступним після вибору вулиці.' }}
            </p>
          </div>
        </div>

        <div class="settings-form-actions">
          <button class="primary-btn" type="button" :disabled="!yasnoCanPreview || yasnoPreviewLoading" @click="previewYasnoSchedule">
            {{ yasnoPreviewLoading ? 'Перевіряємо…' : 'Показати графік' }}
          </button>
          <button
            v-if="yasnoSchedule"
            class="secondary-btn"
            type="button"
            :disabled="yasnoSaveLoading"
            @click="saveYasnoSchedule"
          >
            {{ yasnoSaveLoading ? 'Збереження…' : 'Підтвердити групу' }}
          </button>
          <button
            v-if="yasnoConfig"
            class="secondary-btn danger-btn"
            type="button"
            :disabled="yasnoDeleteLoading"
            @click="deleteYasnoSchedule"
          >
            {{ yasnoDeleteLoading ? 'Відключення…' : 'Відключити Yasno' }}
          </button>
        </div>

        <p v-if="yasnoRegionsLoading || yasnoLoading" class="sub">Завантаження довідників Yasno…</p>
        <p v-if="yasnoError" class="error form-error">{{ yasnoError }}</p>
        <p v-if="yasnoPreviewError" class="error form-error">{{ yasnoPreviewError }}</p>
        <p v-if="yasnoSaveError" class="error form-error">{{ yasnoSaveError }}</p>
        <p v-if="yasnoSaveSuccess" class="success form-success">{{ yasnoSaveSuccess }}</p>
        <p v-if="yasnoScheduleError" class="error form-error">{{ yasnoScheduleError }}</p>

        <div v-if="yasnoConfig" class="yasno-current-config">
          <p class="field-label">Поточна привʼязка</p>
          <p class="helper-text">
            {{ yasnoConfig.regionName }} · {{ yasnoConfig.dsoName }} · {{ yasnoConfig.streetName }}, {{ yasnoConfig.houseName }} · група {{ yasnoConfig.group }}
          </p>
        </div>

        <OutageScheduleCard
          v-if="yasnoSchedule"
          :schedule="yasnoSchedule"
          title="Підтвердження групи"
          subtitle="Перевірте, чи збігається адреса і сьогоднішній графік."
        />
      </section>

      <section v-else-if="activeTab === 'telegram-bot'" class="settings-form-card integration-card telegram-bot-card">
        <h2>Телеграм бот</h2>
        <p class="sub">Підключіть {{ telegramBotHandle }} до групи, щоб бот надсилав туди сповіщення про стан світла.</p>

        <div class="telegram-bot-manual">
          <ol class="telegram-bot-steps">
            <li>
              Додайте бота <strong>{{ telegramBotHandle }}</strong> у потрібну Telegram-групу.
              <a
                v-if="telegramBotStartGroupLink"
                class="secondary-btn telegram-bot-link-btn"
                :href="telegramBotStartGroupLink"
                target="_blank"
                rel="noreferrer"
              >
                Додати бота
              </a>
            </li>
            <li>Після додавання натисніть нижче “Знайти групу” і вкажіть повну назву групи так, як вона написана в Telegram.</li>
            <li>Якщо група не знаходиться, надішліть будь-яке повідомлення у групу або перевірте, що бот справді доданий, і повторіть спробу.</li>
          </ol>
        </div>

        <label class="field-label" for="telegram-bot-group-title">Повна назва групи</label>
        <input
          id="telegram-bot-group-title"
          v-model="telegramBotTitle"
          type="text"
          class="field-input"
          placeholder="Наприклад: Світло ЖК Сонце"
        />

        <div class="settings-form-actions">
          <button class="primary-btn" type="button" :disabled="telegramBotSaving" @click="addTelegramBotGroup">
            {{ telegramBotSaving ? 'Перевіряємо…' : 'Знайти групу' }}
          </button>
        </div>

        <p v-if="telegramBotSaveError" class="error form-error">{{ telegramBotSaveError }}</p>
        <p v-if="telegramBotSuccess" class="success form-success">{{ telegramBotSuccess }}</p>
        <p v-if="telegramBotError" class="error form-error">{{ telegramBotError }}</p>
        <p v-if="telegramBotLoading" class="sub">Завантаження списку груп…</p>

        <div class="telegram-bot-groups">
          <p class="field-label">Групи з підключеним ботом</p>

          <ul v-if="telegramBotGroups.length" class="telegram-bot-group-list">
            <li v-for="group in telegramBotGroups" :key="group.virtualUserId" class="telegram-bot-group-item">
              <div class="telegram-bot-group-main">
                <strong>{{ group.title }}</strong>
                <span class="helper-text">chat_id: {{ group.telegramId }}</span>
              </div>
              <button
                class="secondary-btn danger-btn telegram-bot-delete-btn"
                type="button"
                :disabled="telegramBotDeleting[group.virtualUserId]"
                :title="`Видалити ${group.title}`"
                :aria-label="`Видалити ${group.title}`"
                @click="removeTelegramBotGroup(group)"
              >
                {{ telegramBotDeleting[group.virtualUserId] ? '⏳' : '🗑' }}
              </button>
            </li>
          </ul>

          <p v-else-if="!telegramBotLoading" class="sub settings-empty telegram-bot-empty">Ще немає жодної групи для цього проєкту.</p>
        </div>
      </section>

      <section v-else-if="activeTab === 'firmware'" class="settings-form-card integration-card">
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
