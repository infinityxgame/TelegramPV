<script setup>
import { computed } from 'vue'
import { useI18n } from '../i18n'
import { Languages, Palette, Zap } from '../icons'
import { applyLoaderTheme, applyTheme, themeMap } from '../utils/theme.js'

const props = defineProps({
  settings: {
    type: Object,
    required: true
  }
})

const emit = defineEmits([
  'update:settings',
  'reset-color',
  'reset-loader-color'
])

const { t, locale, setLocale } = useI18n()

const patch = (partial) => {
  emit('update:settings', partial)
}

const specialThemes = computed(() =>
  Object.keys(themeMap)
    .map(Number)
    .filter((id) => id >= 16)
    .sort((a, b) => a - b)
)

const setAccentColor = (id) => {
  patch({ color_id: id })
  applyTheme(id)
}

const setLoaderColor = (id) => {
  patch({ loader_color_id: id })
  applyLoaderTheme(id)
}

const changeLanguage = (lang) => {
  patch({ language: lang })
  setLocale(lang)
}
</script>

<template>
  <div class="settings-view">
    <!-- 1. Temas y Colores -->
    <div class="panel settings-card">
      <div class="panel-heading">
        <div class="eyebrow">
          <Palette :size="16" />
          <span>{{ t('settings.themesTitle') }}</span>
        </div>
        <h2>{{ t('settings.accentTitle') }}</h2>
      </div>

      <div class="settings-columns">
        <!-- Paleta de Color de Acento y Tema -->
        <div class="settings-group color-group">
          <span class="color-group-title">{{
            t('settings.telegramGroup')
          }}</span>
          <div class="color-selector-container">
            <div class="color-row">
              <button
                v-for="id in [0, 1, 2, 3, 4, 5, 6, 7]"
                :key="id"
                type="button"
                class="color-dot"
                :class="{ active: props.settings.color_id === id }"
                :style="{
                  background: themeMap[id]?.gradient || '#38a7ff'
                }"
                :title="t('settings.colorN', { n: id })"
                @click="setAccentColor(id)"
              ></button>
            </div>
            <div class="color-row">
              <button
                v-for="id in [8, 9, 10, 11, 12, 13, 14, 15]"
                :key="id"
                type="button"
                class="color-dot gradient-dot"
                :class="{ active: props.settings.color_id === id }"
                :style="{
                  background: `linear-gradient(135deg, ${themeMap[id]?.primary || '#38a7ff'} 49.8%, ${themeMap[id]?.secondary || '#b48bf2'} 50.2%)`
                }"
                :title="t('settings.gradientN', { n: id })"
                @click="setAccentColor(id)"
              ></button>
            </div>
          </div>

          <span class="color-group-title">{{
            t('settings.thematicGroup')
          }}</span>
          <div class="themes-list">
            <button
              v-for="id in specialThemes"
              :key="'theme-' + id"
              type="button"
              class="theme-chip"
              :class="{ active: props.settings.color_id === id }"
              :title="themeMap[id]?.name || t('settings.themeN', { n: id })"
              @click="setAccentColor(id)"
            >
              <span
                class="theme-swatch"
                :style="{ background: themeMap[id]?.gradient }"
              ></span>
              <span class="theme-name">{{
                themeMap[id]?.name || t('settings.themeN', { n: id })
              }}</span>
            </button>
          </div>

          <button
            type="button"
            class="reset-button-alt"
            @click="emit('reset-color')"
          >
            <Zap :size="14" /> {{ t('settings.resetAccountColor') }}
          </button>
        </div>

        <!-- Color del Loader -->
        <div class="settings-group color-group separate-section">
          <span class="setting-label">{{ t('settings.loaderTitle') }}</span>

          <span class="color-group-title">{{
            t('settings.telegramGroup')
          }}</span>
          <div class="color-selector-container">
            <div class="color-row">
              <button
                v-for="id in [0, 1, 2, 3, 4, 5, 6, 7]"
                :key="'loader-' + id"
                type="button"
                class="color-dot"
                :class="{
                  active: props.settings.loader_color_id === id
                }"
                :style="{
                  background: themeMap[id]?.gradient || '#38a7ff'
                }"
                :title="t('settings.colorLoaderN', { n: id })"
                @click="setLoaderColor(id)"
              ></button>
            </div>
            <div class="color-row">
              <button
                v-for="id in [8, 9, 10, 11, 12, 13, 14, 15]"
                :key="'loader-grad-' + id"
                type="button"
                class="color-dot gradient-dot"
                :class="{
                  active: props.settings.loader_color_id === id
                }"
                :style="{
                  background: `linear-gradient(135deg, ${themeMap[id]?.primary || '#38a7ff'} 49.8%, ${themeMap[id]?.secondary || '#b48bf2'} 50.2%)`
                }"
                :title="t('settings.gradientLoaderN', { n: id })"
                @click="setLoaderColor(id)"
              ></button>
            </div>
          </div>

          <span class="color-group-title">{{
            t('settings.thematicGroupShort')
          }}</span>
          <div class="color-selector-container">
            <div class="color-row">
              <button
                v-for="id in specialThemes"
                :key="'loader-theme-' + id"
                type="button"
                class="color-dot theme-dot"
                :class="{
                  active: props.settings.loader_color_id === id
                }"
                :style="{ background: themeMap[id]?.gradient }"
                :title="themeMap[id]?.name || t('settings.themeN', { n: id })"
                @click="setLoaderColor(id)"
              ></button>
            </div>
          </div>

          <button
            type="button"
            class="reset-button-alt"
            @click="emit('reset-loader-color')"
          >
            <Zap :size="14" /> {{ t('settings.resetLoaderColor') }}
          </button>
        </div>
      </div>
    </div>

    <!-- 2. Idioma -->
    <div class="panel settings-card">
      <div class="panel-heading">
        <div class="eyebrow">
          <Languages :size="16" />
          <span>{{ t('settings.langTitle') }}</span>
        </div>
        <h2>{{ t('settings.langSub') }}</h2>
      </div>

      <div class="settings-group">
        <div class="themes-list">
          <button
            type="button"
            class="theme-chip language-chip"
            :class="{ active: locale === 'es' }"
            :aria-pressed="locale === 'es'"
            @click="changeLanguage('es')"
          >
            <span class="theme-name">Español</span>
          </button>
          <button
            type="button"
            class="theme-chip language-chip"
            :class="{ active: locale === 'en' }"
            :aria-pressed="locale === 'en'"
            @click="changeLanguage('en')"
          >
            <span class="theme-name">English</span>
          </button>
        </div>
        <small class="lang-note">{{ t('settings.langNote') }}</small>
      </div>
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

.settings-columns {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 24px;
  margin-top: 16px;
}

.settings-group {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.separate-section {
  border-top: 1px solid var(--user-border, #1b344b);
  padding-top: 18px;
}

.setting-label {
  font-size: 13px;
  font-weight: 700;
  color: #f1f7ff;
}

.color-group-title {
  display: block;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  color: var(--user-text-dim, #728ba2);
  margin-top: 4px;
}

.color-selector-container {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.color-row {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.color-dot {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  border: none;
  cursor: pointer;
  padding: 0;
  transition: all 0.2s cubic-bezier(0.175, 0.885, 0.32, 1.275);
  position: relative;
  overflow: hidden;
}

.color-dot:hover {
  transform: scale(1.15);
}

.color-dot.active {
  transform: scale(1.1);
  box-shadow:
    0 0 0 3px var(--user-bg-base, #07111f),
    0 0 0 5px var(--user-primary, #38a7ff),
    0 0 15px var(--user-glow);
  z-index: 2;
}

.theme-dot {
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.12);
}

.theme-dot.active {
  box-shadow:
    inset 0 0 0 1px rgba(255, 255, 255, 0.12),
    0 0 0 3px var(--user-bg-base, #07111f),
    0 0 0 5px var(--user-primary, #38a7ff),
    0 0 15px var(--user-glow);
}

.themes-list {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin: 8px 0;
}

.theme-chip {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  border-radius: 999px;
  border: 1px solid var(--user-border, #1b344b);
  background: var(--user-surface-light, #0e2032);
  color: var(--user-text-dim, #728ba2);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s cubic-bezier(0.175, 0.885, 0.32, 1.275);
}

.theme-chip:hover {
  transform: translateY(-2px);
  border-color: var(--user-primary, #38a7ff);
  color: #eef7ff;
}

.theme-chip.active {
  color: #eef7ff;
  border-color: var(--user-primary, #38a7ff);
  background: var(--user-surface, #0b1a2a);
  box-shadow:
    0 0 0 1px var(--user-primary, #38a7ff),
    0 6px 18px var(--user-glow);
}

.theme-swatch {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  flex: none;
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.12);
}

.reset-button-alt {
  background: var(--user-surface-light, #0e2032);
  border: 1px solid var(--user-border, #1b344b);
  color: var(--user-text-dim, #728ba2);
  padding: 9px 15px;
  border-radius: 10px;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 8px;
  width: fit-content;
  transition: all 0.2s;
  margin-top: 6px;
}

.reset-button-alt:hover {
  background: var(--user-icon-bg, #11385b);
  color: var(--user-accent, #5ebcff);
  border-color: var(--user-primary, #38a7ff);
}

.lang-note {
  font-size: 12px;
  color: var(--user-text-dim, #728ba2);
  line-height: 1.4;
}
</style>
