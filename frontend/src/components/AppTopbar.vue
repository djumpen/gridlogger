<script setup>
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useAuth } from '../composables/useAuth'

const telegramWidgetRef = ref(null)
const userMenuRef = ref(null)
const userMenuOpen = ref(false)

const {
  telegramConfig,
  currentUser,
  currentUserLabel,
  authError,
  renderTelegramWidget,
  initializeAuth,
  logout,
  disposeAuth
} = useAuth()

onMounted(async () => {
  await initializeAuth()
  document.addEventListener('pointerdown', handleOutsidePointerDown)
})

watch(
  [telegramConfig, currentUser, telegramWidgetRef],
  async () => {
    await nextTick()
    renderTelegramWidget(telegramWidgetRef.value)
  },
  { immediate: true }
)

watch(currentUser, (user) => {
  if (!user) {
    userMenuOpen.value = false
  }
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', handleOutsidePointerDown)
  disposeAuth(telegramWidgetRef.value)
})

function toggleUserMenu() {
  userMenuOpen.value = !userMenuOpen.value
}

async function handleLogout() {
  userMenuOpen.value = false
  await logout()
  window.location.href = '/'
}

function handleOutsidePointerDown(event) {
  if (!userMenuOpen.value) return
  if (userMenuRef.value?.contains(event.target)) return
  userMenuOpen.value = false
}
</script>

<template>
  <div class="topbar">
    <a href="/" class="logo-link" title="На головну">Svitlo.🏘️</a>
    <div class="topbar-actions">
      <template v-if="currentUser">
        <a class="topbar-link" href="/a/settings">Мої адреси</a>

        <div class="user-menu-wrap" ref="userMenuRef">
          <button class="user-menu-btn" type="button" @click="toggleUserMenu">
            <span>{{ currentUserLabel }}</span>
            <span class="user-menu-caret">▾</span>
          </button>

          <div v-if="userMenuOpen" class="user-menu-popover">
            <button class="user-menu-item" type="button" @click="handleLogout">Вийти</button>
          </div>
        </div>
      </template>
      <div v-else-if="telegramConfig.enabled" class="telegram-widget-wrap">
        <div ref="telegramWidgetRef"></div>
      </div>
      <span v-else class="auth-muted">Вхід через Telegram недоступний</span>
    </div>
  </div>

  <p v-if="authError" class="error auth-error">{{ authError }}</p>
</template>
