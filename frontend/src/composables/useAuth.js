import { computed, ref } from 'vue'

export function useAuth() {
  const telegramConfig = ref({
    enabled: false,
    botUsername: '',
    requestAccess: 'write'
  })
  const currentUser = ref(null)
  const authError = ref('')

  const callbackName = 'gridloggerTelegramAuth'
  let widgetRoot = null

  const currentUserLabel = computed(() => {
    if (!currentUser.value) return ''
    if (currentUser.value.username) {
      return `@${currentUser.value.username}`
    }

    const firstName = currentUser.value.firstName || ''
    const lastName = currentUser.value.lastName || ''
    return `${firstName} ${lastName}`.trim() || 'Telegram'
  })

  function clearWidget() {
    if (widgetRoot) {
      widgetRoot.innerHTML = ''
    }
  }

  async function loadTelegramConfig() {
    try {
      const resp = await fetch('/api/auth/telegram/config', { credentials: 'include' })
      if (!resp.ok) {
        telegramConfig.value = { enabled: false, botUsername: '', requestAccess: 'write' }
        return
      }

      const data = await resp.json()
      telegramConfig.value = {
        enabled: !!data.enabled,
        botUsername: data.botUsername || '',
        requestAccess: data.requestAccess || 'write'
      }
    } catch {
      telegramConfig.value = { enabled: false, botUsername: '', requestAccess: 'write' }
    }
  }

  async function loadMe() {
    try {
      const resp = await fetch('/api/me', { credentials: 'include' })
      if (!resp.ok) {
        currentUser.value = null
        return
      }

      const data = await resp.json()
      currentUser.value = data.user || null
    } catch {
      currentUser.value = null
    }
  }

  function renderTelegramWidget() {
    clearWidget()
    if (!widgetRoot || !telegramConfig.value.enabled || !telegramConfig.value.botUsername || currentUser.value) {
      return
    }

    window[callbackName] = async (user) => {
      try {
        authError.value = ''
        const body = new URLSearchParams()
        for (const [key, value] of Object.entries(user || {})) {
          if (value === undefined || value === null) continue
          body.append(key, String(value))
        }

        const resp = await fetch('/api/auth/telegram/callback', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/x-www-form-urlencoded'
          },
          credentials: 'include',
          body
        })
        if (!resp.ok) {
          throw new Error(await resp.text())
        }

        const data = await resp.json()
        currentUser.value = data.user || null
        clearWidget()
      } catch (e) {
        authError.value = e.message || 'Не вдалося увійти через Telegram'
      }
    }

    const script = document.createElement('script')
    script.async = true
    script.src = 'https://telegram.org/js/telegram-widget.js?22'
    script.setAttribute('data-telegram-login', telegramConfig.value.botUsername)
    script.setAttribute('data-size', 'medium')
    script.setAttribute('data-userpic', 'false')
    script.setAttribute('data-request-access', telegramConfig.value.requestAccess)
    script.setAttribute('data-onauth', `${callbackName}(user)`)
    widgetRoot.appendChild(script)
  }

  async function initializeAuth(rootEl) {
    widgetRoot = rootEl
    await loadTelegramConfig()
    await loadMe()
    renderTelegramWidget()
  }

  async function logout() {
    try {
      await fetch('/api/auth/logout', {
        method: 'POST',
        credentials: 'include'
      })
    } finally {
      currentUser.value = null
      authError.value = ''
      renderTelegramWidget()
    }
  }

  function disposeAuth() {
    clearWidget()
    delete window[callbackName]
    widgetRoot = null
  }

  return {
    telegramConfig,
    currentUser,
    currentUserLabel,
    authError,
    initializeAuth,
    logout,
    disposeAuth
  }
}
