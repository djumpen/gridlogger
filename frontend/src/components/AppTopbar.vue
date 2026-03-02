<script setup>
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useAuth } from '../composables/useAuth'

const telegramWidgetRef = ref(null)
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
})

watch(
  [telegramConfig, currentUser, telegramWidgetRef],
  async () => {
    await nextTick()
    renderTelegramWidget(telegramWidgetRef.value)
  },
  { immediate: true }
)

onBeforeUnmount(() => {
  disposeAuth(telegramWidgetRef.value)
})
</script>

<template>
  <div class="topbar">
    <a href="/" class="logo-link" title="На головну">Svitlo.🏘️</a>
    <div class="topbar-actions">
      <template v-if="currentUser">
        <span class="user-chip">{{ currentUserLabel }}</span>
        <button class="login-link" type="button" @click="logout">Вийти</button>
      </template>
      <div v-else-if="telegramConfig.enabled" class="telegram-widget-wrap">
        <div ref="telegramWidgetRef"></div>
      </div>
      <span v-else class="auth-muted">Вхід через Telegram недоступний</span>
    </div>
  </div>

  <p v-if="authError" class="error auth-error">{{ authError }}</p>
</template>
