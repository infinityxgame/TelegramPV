<script setup>
import { onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import AuthWizard from './components/AuthWizard.vue'
import ConfirmModal from './components/ConfirmModal.vue'
import { useAuthToken } from './composables/useAuthToken'
import { useConfirmModal } from './composables/useConfirmModal'
import { useUpdater } from './composables/useUpdater'
import { useI18n } from './i18n'
import {
  ArrowUpRight,
  Bot,
  Layers,
  LogOut,
  Menu,
  Radio,
  ScrollText,
  Settings,
  User,
  Users,
  X,
  Zap
} from './icons'
import { applyLoaderTheme, applyTheme, initTheme } from './utils/theme.js'
import ChatsView from './views/ChatsView.vue'
import LogsView from './views/LogsView.vue'
import SettingsView from './views/SettingsView.vue'

const { authHeaders } = useAuthToken()
const { t, setLocale } = useI18n()
const { modal, openConfirm, handleConfirm, handleCancel } = useConfirmModal()

initTheme()

const logoUrl = `${import.meta.env.BASE_URL}telegrampv-android-icon.png`
const activeView = ref('all')
const mobileMenuOpen = ref(false)
const bootstrapping = ref(true)
const showSplash = ref(true)

const api = async (url, options = {}) => {
  const res = await fetch(url, {
    ...options,
    headers: {
      ...(options.headers || {}),
      ...authHeaders()
    }
  })
  const data = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error(data.detail || data.error || t('common.error'))
  }
  return data
}

const showMessage = (msg) => {
  console.log(msg)
}

const {
  version,
  updateInfo,
  isUpdating,
  isUpdateForced,
  updateProgress,
  installUpdate,
  checkForUpdates
} = useUpdater({ api, showMessage, openConfirm })

let updateCheckTimer

const formatSize = (bytes) => {
  if (!bytes) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

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

const fontReady = ref(false)

onMounted(async () => {
  try {
    if (document.fonts && document.fonts.load) {
      await document.fonts.load('86px LoaderFont')
    }
  } catch {
    /* fallback */
  }
  fontReady.value = true

  // Inicio de comprobación de actualizaciones
  setTimeout(() => checkForUpdates(false), 1000)
  updateCheckTimer = setInterval(() => checkForUpdates(false), 5 * 60 * 1000)

  const startBoot = Date.now()
  await checkAuthStatus()
  await loadSettings()
  await checkForUpdates(true)

  const elapsed = Date.now() - startBoot
  const minTime = 6500
  const remaining = Math.max(0, minTime - elapsed)
  setTimeout(() => {
    bootstrapping.value = false
  }, remaining)
})

onUnmounted(() => {
  if (updateCheckTimer) {
    clearInterval(updateCheckTimer)
  }
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
            :class="{ 'font-ready': fontReady }"
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
    <!-- Diálogo de actualización obligatoria -->
    <div v-if="updateInfo && isUpdateForced" class="update-required-overlay">
      <div class="update-card">
        <Zap :size="48" class="update-icon" />
        <h2>{{ t('update.forcedTitle') }}</h2>
        <p v-if="!isUpdating">
          {{ t('update.forcedText', { version: updateInfo.latest }) }}
        </p>

        <div class="update-action-area" :class="{ 'is-loading': isUpdating }">
          <button
            v-if="!isUpdating"
            class="primary-button update-btn"
            @click="installUpdate"
          >
            <span>{{ t('update.updateNow') }}</span>
            <ArrowUpRight :size="18" />
          </button>

          <div v-else class="update-progress-container">
            <div class="update-status-text">
              {{
                updateProgress.status === 'downloading'
                  ? t('update.statusDownloading')
                  : updateProgress.status === 'extracting'
                    ? t('update.statusExtracting')
                    : updateProgress.status === 'finishing'
                      ? t('update.statusFinishing')
                      : t('update.statusStarting')
              }}
            </div>
            <div class="update-progress-bar">
              <div
                class="update-progress-fill"
                :style="{ width: updateProgress.percentage + '%' }"
              >
                <div class="nitro-wind">
                  <span></span><span></span><span></span><span></span>
                </div>
              </div>
            </div>
            <div class="update-progress-stats">
              <span
                >{{ formatSize(updateProgress.downloaded) }} /
                {{ formatSize(updateProgress.total) }}</span
              >
              <span>{{ updateProgress.percentage }}%</span>
            </div>
          </div>
        </div>

        <div v-if="isUpdating" class="update-warning">
          {{ t('update.doNotClose') }}
        </div>

        <small>{{
          t('update.currentVersion', { version: updateInfo.current })
        }}</small>
      </div>
    </div>

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
          <span class="brand-mark">
            <img :src="logoUrl" alt="TelegramPV" />
          </span>
          <div class="brand-text">
            <span class="brand-title">
              <span class="b1">T</span>
              <span class="b2">e</span>
              <span class="b3">l</span>
              <span class="b4">e</span>
              <span class="b5">g</span>
              <span class="b6">r</span>
              <span class="b7">a</span>
              <span class="b8">m</span>
              <span class="brand-accent b9">P</span>
              <span class="brand-accent b10">V</span>
            </span>
            <span class="brand-version">
              <span
                v-for="(char, i) in (
                  'v' + (version || t('app.version') || '1.0.3')
                ).split('')"
                :key="i"
                :class="'b' + (i + 1)"
              >
                {{ char }}
              </span>
            </span>
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
      <ConfirmModal
        :show="modal.show"
        :title="modal.title"
        :message="modal.message"
        :confirm-text="modal.confirmText"
        :cancel-text="modal.cancelText"
        :type="modal.type"
        @confirm="handleConfirm"
        @cancel="handleCancel"
      />
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

.hello-text.font-ready tspan:not(.dots):not(.dot1):not(.dot2):not(.dot3) {
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
  animation-delay: 0.5s;
}
.c3 {
  animation-delay: 0.9s;
}
.c4 {
  animation-delay: 1.3s;
}
.c5 {
  animation-delay: 1.7s;
}
.c6 {
  animation-delay: 2.1s;
}
.c7 {
  animation-delay: 2.5s;
}
.c8 {
  animation-delay: 2.9s;
}
.c9 {
  animation-delay: 3.3s;
}
.c10 {
  animation-delay: 3.7s;
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
  animation-delay: 4.2s;
}
.dot2 {
  animation-delay: 4.4s;
}
.dot3 {
  animation-delay: 4.6s;
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

/* Update Modal Overlay Styles */
.update-required-overlay {
  position: fixed;
  inset: 0;
  background: color-mix(in srgb, var(--user-bg-base, #07111f), transparent 5%);
  backdrop-filter: blur(8px);
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}
.update-card {
  background: var(--user-surface, #0b1a2a);
  border: 1px solid var(--user-border, #1b344b);
  border-radius: 24px;
  padding: 40px;
  max-width: 400px;
  width: 100%;
  text-align: center;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
}
.update-icon {
  color: var(--user-primary, #38a7ff);
  margin-bottom: 20px;
}
.update-card h2 {
  color: #f8fafc;
  margin-bottom: 12px;
  font-size: 24px;
}
.update-card p {
  color: var(--user-text-dim, #728ba2);
  margin-bottom: 30px;
  line-height: 1.6;
}
.update-action-area {
  transition: all 0.6s cubic-bezier(0.34, 1.56, 0.64, 1);
  min-height: 80px;
  display: flex;
  flex-direction: column;
  justify-content: center;
  position: relative;
}
.update-action-area.is-loading {
  transform: translateY(-5px);
}
.update-progress-container {
  width: 100%;
  text-align: left;
}
.update-status-text {
  color: #f8fafc;
  font-size: 14px;
  margin-bottom: 12px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}
.update-progress-bar {
  height: 16px;
  background: var(--user-bg-base, #07111f);
  border-radius: 20px;
  overflow: hidden;
  margin-bottom: 10px;
  position: relative;
  border: 1px solid var(--user-border, #1b344b);
  box-shadow: inset 0 2px 8px rgba(0, 0, 0, 0.5);
}
.update-progress-fill {
  height: 100%;
  background: linear-gradient(
    90deg,
    var(--user-bg-top, #163557),
    var(--user-primary, #38a7ff)
  );
  transition: width 0.4s cubic-bezier(0.1, 0.7, 0.1, 1);
  position: relative;
  display: flex;
  align-items: center;
  justify-content: flex-end;
}

.nitro-wind {
  position: absolute;
  inset: 0;
  overflow: hidden;
  pointer-events: none;
}
.nitro-wind span {
  position: absolute;
  background: linear-gradient(
    90deg,
    transparent,
    rgba(255, 255, 255, 0.6),
    transparent
  );
  height: 2px;
  border-radius: 2px;
  animation: windBackwards 0.6s linear infinite;
}
.nitro-wind span:nth-child(1) {
  top: 25%;
  width: 40px;
  animation-duration: 0.4s;
}
.nitro-wind span:nth-child(2) {
  top: 50%;
  width: 60px;
  animation-duration: 0.7s;
  animation-delay: 0.1s;
}
.nitro-wind span:nth-child(3) {
  top: 75%;
  width: 30px;
  animation-duration: 0.5s;
  animation-delay: 0.2s;
}
.nitro-wind span:nth-child(4) {
  top: 40%;
  width: 50px;
  animation-duration: 0.8s;
  animation-delay: 0.3s;
}

.update-progress-stats {
  display: flex;
  justify-content: space-between;
  color: var(--user-text-dim, #728ba2);
  font-size: 12px;
  font-family: monospace;
  font-weight: 600;
}
.update-warning {
  color: #f87171;
  font-size: 13px;
  margin-top: 15px;
  font-weight: 500;
  text-align: center;
}

.update-btn {
  width: fit-content;
  min-width: 240px;
  height: 50px;
  padding: 0 30px;
  font-size: 16px;
  font-weight: 700;
  margin: 0 auto;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 25px;
  gap: 10px;
  background: var(--user-primary, #38a7ff);
  color: white;
  border: none;
  cursor: pointer;
  transition: all 0.3s ease;
}
.update-btn:hover {
  transform: scale(1.05) translateY(-2px);
}
.update-card small {
  color: var(--user-text-dim, #728ba2);
  opacity: 0.8;
  display: block;
  margin-top: 15px;
}

@keyframes windBackwards {
  from {
    transform: translateX(300px);
    opacity: 0;
  }
  50% {
    opacity: 1;
  }
  to {
    transform: translateX(-100px);
    opacity: 0;
  }
}
</style>
