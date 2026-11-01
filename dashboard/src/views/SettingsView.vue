<script setup>
import { useConfirmModal } from '../composables/useConfirmModal'
import { useI18n } from '../i18n'
import { Check, Languages, LogOut, Palette, Settings } from '../icons'
import { applyTheme, themeMap } from '../utils/theme.js'

defineProps({
  settings: {
    type: Object,
    required: true
  },
  authStatus: {
    type: Object,
    default: () => ({})
  }
})

const emit = defineEmits(['update:settings', 'logout'])
const { t, locale, setLocale } = useI18n()
const { confirm } = useConfirmModal()

const patch = (partial) => {
  emit('update:settings', partial)
}

const selectTheme = (colorId) => {
  patch({ color_id: colorId, loader_color_id: colorId })
  applyTheme(colorId)
}

const changeLanguage = (lang) => {
  patch({ language: lang })
  setLocale(lang)
}

const handleLogout = async () => {
  const confirmed = await confirm({
    title: t('settings.logout'),
    message: t('settings.logoutConfirm'),
    confirmText: t('settings.logout'),
    cancelText: t('common.cancel')
  })

  if (confirmed) {
    emit('logout')
  }
}
</script>

<template>
  <div class="settings-view">
    <!-- Theme & Appearance -->
    <div class="panel settings-card">
      <div class="panel-heading">
        <div class="eyebrow">
          <Palette :size="16" />
          <span>{{ t('settings.appearance') }}</span>
        </div>
        <h2>{{ t('settings.themeSelect') }}</h2>
      </div>

      <div class="theme-grid">
        <button
          v-for="(palette, idStr) in themeMap"
          :key="idStr"
          class="theme-swatch"
          :class="{ selected: settings.color_id === Number(idStr) }"
          :style="{ background: palette.gradient || palette.primary }"
          @click="selectTheme(Number(idStr))"
        >
          <Check
            v-if="settings.color_id === Number(idStr)"
            :size="16"
            class="swatch-check"
          />
        </button>
      </div>
    </div>

    <!-- Language Settings -->
    <div class="panel settings-card">
      <div class="panel-heading">
        <div class="eyebrow">
          <Languages :size="16" />
          <span>{{ t('settings.language') }}</span>
        </div>
        <h2>{{ t('settings.languageSelect') }}</h2>
      </div>

      <div class="language-options">
        <button
          class="lang-btn"
          :class="{ selected: locale === 'es' }"
          @click="changeLanguage('es')"
        >
          Español (ES)
        </button>
        <button
          class="lang-btn"
          :class="{ selected: locale === 'en' }"
          @click="changeLanguage('en')"
        >
          English (EN)
        </button>
      </div>
    </div>

    <!-- Session Settings -->
    <div class="panel settings-card">
      <div class="panel-heading">
        <div class="eyebrow">
          <Settings :size="16" />
          <span>{{ t('settings.session') }}</span>
        </div>
        <h2>{{ t('settings.session') }}</h2>
      </div>

      <div v-if="authStatus.user" class="session-info">
        <p>
          <strong>{{ t('settings.connectedAs') }}:</strong>
          {{ authStatus.user.first_name }} {{ authStatus.user.last_name }} (@{{
            authStatus.user.username
          }})
        </p>
        <p v-if="authStatus.user.phone">
          <strong>{{ t('settings.phone') }}:</strong>
          {{ authStatus.user.phone }}
        </p>
      </div>

      <button class="btn-logout" @click="handleLogout">
        <LogOut :size="16" />
        <span>{{ t('settings.logout') }}</span>
      </button>
    </div>
  </div>
</template>

<style scoped>
.settings-view {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.settings-card {
  padding: 24px;
}

.theme-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(44px, 1fr));
  gap: 12px;
  margin-top: 16px;
}

.theme-swatch {
  height: 44px;
  border-radius: 12px;
  border: 2px solid transparent;
  cursor: pointer;
  display: grid;
  place-items: center;
  transition: all 0.2s ease;
  position: relative;
}

.theme-swatch:hover {
  transform: scale(1.08);
}

.theme-swatch.selected {
  border-color: #fff;
  box-shadow: 0 0 14px var(--user-glow);
}

.swatch-check {
  color: #fff;
  filter: drop-shadow(0 1px 3px rgba(0, 0, 0, 0.8));
}

.language-options {
  display: flex;
  gap: 12px;
  margin-top: 16px;
}

.lang-btn {
  padding: 12px 20px;
  border-radius: 10px;
  background: var(--user-surface-light, #0e2032);
  border: 1px solid var(--user-border-light, #234765);
  color: #dbe7f5;
  font-weight: 600;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.lang-btn:hover {
  background: var(--user-border, #1b344b);
}

.lang-btn.selected {
  background: var(--user-primary, #38a7ff);
  color: var(--user-bg-base, #07111f);
  border-color: transparent;
}

.session-info {
  background: var(--user-bg-base, #07111f);
  border: 1px solid var(--user-border, #1b344b);
  padding: 14px 18px;
  border-radius: 10px;
  margin: 16px 0;
  font-size: 13px;
  color: var(--user-text-dim, #728ba2);
}

.session-info p {
  margin: 4px 0;
}

.session-info strong {
  color: #dbe7f5;
}

.btn-logout {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 18px;
  background: #3d171d;
  border: 1px solid #732a36;
  color: #ffadad;
  border-radius: 10px;
  font-weight: 600;
  font-size: 13px;
  cursor: pointer;
}

.btn-logout:hover {
  background: #48252c;
  color: #ffb0b0;
}
</style>
