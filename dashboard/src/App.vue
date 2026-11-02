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
  await checkAuthStatus()
  await loadSettings()
})
</script>

<template>
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
