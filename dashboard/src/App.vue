<script setup>
import { onMounted, reactive, ref, watch } from 'vue'
import AuthWizard from './components/AuthWizard.vue'
import ConfirmModal from './components/ConfirmModal.vue'
import { useAuthToken } from './composables/useAuthToken'
import { useI18n } from './i18n'
import {
  Bot,
  Layers,
  LogOut,
  Menu,
  Radio,
  ScrollText,
  Settings,
  User,
  Users,
  X
} from './icons'
import { applyLoaderTheme, applyTheme, initTheme } from './utils/theme.js'
import ChatsView from './views/ChatsView.vue'
import LogsView from './views/LogsView.vue'
import SettingsView from './views/SettingsView.vue'

const { authHeaders } = useAuthToken()
const { t, setLocale } = useI18n()

initTheme()

const activeView = ref('all')
const mobileMenuOpen = ref(false)
const bootstrapping = ref(true)
const showSplash = ref(true)

const selectNav = (view) => {
  activeView.value = view
  mobileMenuOpen.value = false
}

const authStatus = ref({
  configured: false,
  has_credentials: false,
  authorized: false,
  authenticated: false,
  state: 'UNCONFIGURED',
  phone: '',
  user: null
})

const settings = reactive({
  color_id: 5,
  loader_color_id: 5,
  language: 'es'
})

const updateSettings = (partial) => {
  Object.assign(settings, partial)
  saveSettings()
}

const saveSettings = async () => {
  try {
    await fetch('/api/settings', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...authHeaders()
      },
      body: JSON.stringify(settings)
    })
  } catch (err) {
    console.error('Failed to save settings:', err)
  }
}

const loadSettings = async () => {
  try {
    const res = await fetch('/api/settings', { headers: authHeaders() })
    if (res.ok) {
      const data = await res.json()
      if (data.color_id !== undefined && data.color_id !== null) {
        settings.color_id = data.color_id
        settings.loader_color_id = data.loader_color_id || data.color_id
        applyTheme(settings.color_id)
        applyLoaderTheme(settings.loader_color_id)
      }
      if (data.language) {
        settings.language = data.language
        setLocale(data.language)
      }
    }
  } catch (err) {
    console.error('Failed to load settings:', err)
  }
}

const checkAuthStatus = async () => {
  try {
    const res = await fetch('/api/auth/status', { headers: authHeaders() })
    if (res.ok) {
      const data = await res.json()
      authStatus.value = data
      if (
        data.user &&
        data.user.color_id !== undefined &&
        data.user.color_id !== null
      ) {
        settings.color_id = data.user.color_id
        applyTheme(data.user.color_id)
      }
    }
  } catch (err) {
    console.error('Failed to fetch auth status:', err)
  }
}

const handleAuthSuccess = (newStatus) => {
  if (newStatus && typeof newStatus === 'object') {
    authStatus.value = { ...authStatus.value, ...newStatus }
  }
  checkAuthStatus()
  loadSettings()
}

const handleLogout = async () => {
  try {
    await fetch('/api/auth/logout', {
      method: 'POST',
      headers: authHeaders()
    })
    authStatus.value = {
      configured: false,
      has_credentials: false,
      authorized: false,
      authenticated: false,
      state: 'UNCONFIGURED',
      phone: '',
      user: null
    }
  } catch (err) {
    console.error('Logout error:', err)
  }
}

const resetColor = () => {
  if (
    authStatus.value.user &&
    authStatus.value.user.color_id !== undefined &&
    authStatus.value.user.color_id !== null
  ) {
    settings.color_id = authStatus.value.user.color_id
  } else {
    settings.color_id = 5
  }
  applyTheme(settings.color_id)
  saveSettings()
}

const resetLoaderColor = () => {
  settings.loader_color_id = settings.color_id
  applyLoaderTheme(settings.loader_color_id)
  saveSettings()
}

watch(
  () => settings.color_id,
  (newVal) => {
    if (newVal !== undefined && newVal !== null) {
      applyTheme(newVal)
    }
  }
)

watch(
  () => settings.language,
  (newVal) => {
    if (newVal) {
      setLocale(newVal)
    }
  }
)

onMounted(async () => {
  const startBoot = Date.now()
  await checkAuthStatus()
  await loadSettings()

  const elapsed = Date.now() - startBoot
  const minTime = 4200
  const remaining = Math.max(0, minTime - elapsed)
  setTimeout(() => {
    bootstrapping.value = false
  }, remaining)
})
</script>

<template>
  <!-- Pantalla de arranque / Loader -->
  <transition name="fade">
    <div v-if="bootstrapping && showSplash" class="boot-screen">
      <div class="boot-container">
        <svg viewBox="0 0 500 150" class="hello-svg">
          <text
            x="50%"
            y="50%"
            text-anchor="middle"
            dominant-baseline="middle"
            class="hello-text"
          >
            <tspan class="c1">T</tspan>
            <tspan class="c2">e</tspan>
            <tspan class="c3">l</tspan>
            <tspan class="c4">e</tspan>
            <tspan class="c5">g</tspan>
            <tspan class="c6">r</tspan>
            <tspan class="c7">a</tspan>
            <tspan class="c8">m</tspan>
            <tspan class="c9">P</tspan>
            <tspan class="c10">V</tspan>
            <tspan class="dots" dx="20" dy="8">
              <tspan class="dot1">●</tspan>
              <tspan class="dot2">●</tspan>
              <tspan class="dot3">●</tspan>
            </tspan>
          </text>
        </svg>
      </div>
    </div>
  </transition>

  <template v-if="!bootstrapping">
    <!-- Unauthenticated State: Show AuthWizard without sidebar -->
    <AuthWizard
      v-if="!authStatus.authenticated"
      :auth-status="authStatus"
      @auth-success="handleAuthSuccess"
    />

    <!-- Authenticated State: Show App Shell with Sidebar -->
    <div v-else class="app-shell">
      <!-- Sidebar -->
      <aside class="sidebar" :class="{ 'mobile-open': mobileMenuOpen }">
        <div class="brand">
          <div class="brand-text">
            <span>Telegram<span class="brand-accent">PV</span></span>
            <span class="version-tag">v{{ t('app.version') }}</span>
          </div>
          <button
            class="mobile-menu-toggle"
            @click="mobileMenuOpen = !mobileMenuOpen"
          >
            <X v-if="mobileMenuOpen" :size="20" />
            <Menu v-else :size="20" />
          </button>
        </div>

        <p class="sidebar-copy">{{ t('app.subtitle') }}</p>

        <nav class="sidebar-nav">
          <button
            :class="{ selected: activeView === 'all' }"
            @click="selectNav('all')"
          >
            <Layers :size="18" />
            <span>{{ t('nav.allChats') }}</span>
          </button>

          <button
            :class="{ selected: activeView === 'groups' }"
            @click="selectNav('groups')"
          >
            <Users :size="18" />
            <span>{{ t('nav.groups') }}</span>
          </button>

          <button
            :class="{ selected: activeView === 'channels' }"
            @click="selectNav('channels')"
          >
            <Radio :size="18" />
            <span>{{ t('nav.channels') }}</span>
          </button>

          <button
            :class="{ selected: activeView === 'bots' }"
            @click="selectNav('bots')"
          >
            <Bot :size="18" />
            <span>{{ t('nav.bots') }}</span>
          </button>

          <button
            :class="{ selected: activeView === 'users' }"
            @click="selectNav('users')"
          >
            <User :size="18" />
            <span>{{ t('nav.users') }}</span>
          </button>

          <button
            :class="{ selected: activeView === 'logs' }"
            @click="selectNav('logs')"
          >
            <ScrollText :size="18" />
            <span>{{ t('nav.logs') }}</span>
          </button>

          <button
            :class="{ selected: activeView === 'settings' }"
            @click="selectNav('settings')"
          >
            <Settings :size="18" />
            <span>{{ t('nav.settings') }}</span>
          </button>
        </nav>

        <div v-if="authStatus.user" class="sidebar-footer">
          <div class="sidebar-user-badge">
            <div class="user-info">
              <span class="user-name"
                >{{ authStatus.user.first_name }}
                {{ authStatus.user.last_name }}</span
              >
            </div>
            <button
              class="logout-btn"
              :title="t('settings.logout')"
              @click="handleLogout"
            >
              <LogOut :size="14" />
            </button>
          </div>
        </div>
      </aside>

      <!-- Main Content -->
      <main class="main-content">
        <!-- Topbar -->
        <div class="topbar">
          <div>
            <span class="eyebrow">{{ t('app.title') }}</span>
            <h1 v-if="activeView === 'all'">{{ t('nav.allChats') }}</h1>
            <h1 v-else-if="activeView === 'groups'">{{ t('nav.groups') }}</h1>
            <h1 v-else-if="activeView === 'channels'">
              {{ t('nav.channels') }}
            </h1>
            <h1 v-else-if="activeView === 'bots'">{{ t('nav.bots') }}</h1>
            <h1 v-else-if="activeView === 'users'">{{ t('nav.users') }}</h1>
            <h1 v-else-if="activeView === 'settings'">
              {{ t('settings.title') }}
            </h1>
            <h1 v-else-if="activeView === 'logs'">{{ t('logs.title') }}</h1>
          </div>
        </div>

        <!-- Views -->
        <ChatsView
          v-if="
            ['all', 'groups', 'channels', 'bots', 'users'].includes(activeView)
          "
          :category="activeView"
        />

        <SettingsView
          v-else-if="activeView === 'settings'"
          :settings="settings"
          @update:settings="updateSettings"
          @reset-color="resetColor"
          @reset-loader-color="resetLoaderColor"
        />

        <LogsView v-else-if="activeView === 'logs'" />
      </main>

      <!-- Global Confirm Modal -->
      <ConfirmModal />
    </div>
  </template>
</template>

<style>
@font-face {
  font-family: 'LoaderFont';
  src: url('/fonts/loader.otf') format('opentype');
  font-weight: normal;
  font-style: normal;
}

.boot-screen {
  position: fixed;
  inset: 0;
  background: var(--user-bg-base, #07111f);
  background-image: radial-gradient(
    circle at 70% -10%,
    var(--user-bg-top, #163557) 0,
    var(--user-bg-base, #07111f) 40%
  );
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10000;
}

.boot-container {
  width: 100%;
  display: flex;
  justify-content: center;
  align-items: center;
}

.hello-svg {
  width: 90%;
  max-width: 600px;
  overflow: visible;
}

.hello-text {
  font-family: 'LoaderFont', cursive, sans-serif;
  font-weight: 700;
  font-size: 86px;
  filter: drop-shadow(0 0 25px var(--loader-glow, rgba(56, 167, 255, 0.4)));
}

.hello-text tspan:not(.dots):not(.dot1):not(.dot2):not(.dot3) {
  fill: var(--loader-primary, #38a7ff);
  fill-opacity: 0;
  stroke: var(--loader-primary, #38a7ff);
  stroke-width: 1.2;
  stroke-dasharray: 400;
  stroke-dashoffset: 400;
  animation: writeAndFill 1.8s cubic-bezier(0.445, 0.05, 0.55, 0.95) forwards;
}

.c1 {
  animation-delay: 0.1s;
}
.c2 {
  animation-delay: 0.4s;
}
.c3 {
  animation-delay: 0.7s;
}
.c4 {
  animation-delay: 1s;
}
.c5 {
  animation-delay: 1.3s;
}
.c6 {
  animation-delay: 1.6s;
}
.c7 {
  animation-delay: 1.9s;
}
.c8 {
  animation-delay: 2.2s;
}
.c9 {
  animation-delay: 2.5s;
}
.c10 {
  animation-delay: 2.8s;
}

.dots {
  font-family: Arial, sans-serif;
  font-weight: bold;
  font-size: 24px;
  stroke: none !important;
  stroke-width: 0 !important;
  vertical-align: middle;
}

.dot1,
.dot2,
.dot3 {
  opacity: 0;
  fill: var(--loader-primary, #38a7ff) !important;
  animation: dotFade 1.5s infinite;
}

.dot1 {
  animation-delay: 3.2s;
}
.dot2 {
  animation-delay: 3.4s;
}
.dot3 {
  animation-delay: 3.6s;
}

@keyframes dotFade {
  0%,
  100% {
    opacity: 0;
  }
  50% {
    opacity: 1;
  }
}

@keyframes writeAndFill {
  0% {
    stroke-dashoffset: 400;
    fill-opacity: 0;
  }
  40% {
    fill-opacity: 0.3;
  }
  100% {
    stroke-dashoffset: 0;
    fill-opacity: 1;
  }
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.8s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
