<script setup>
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import {
  ArrowDownToLine,
  Copy,
  Download,
  Pause,
  Play,
  ScrollText,
  Search,
  Trash2
} from '../icons'
import { useAuthToken } from '../composables/useAuthToken'
import { useConfirmModal } from '../composables/useConfirmModal'
import { useI18n } from '../i18n'

const { authHeaders } = useAuthToken()
const { openConfirm } = useConfirmModal()
const { t, has } = useI18n()

const props = defineProps({
  notify: { type: Function, default: () => {} },
  logsSeq: { type: Number, default: 0 },
  active: { type: Boolean, default: false }
})

const MAX_ENTRIES = 3000
const MAX_RENDERED = 800

const LEVELS = [
  { id: 'error', labelKey: 'logs.levelError' },
  { id: 'warn', labelKey: 'logs.levelWarn' },
  { id: 'success', labelKey: 'logs.levelSuccess' },
  { id: 'info', labelKey: 'logs.levelInfo' },
  { id: 'debug', labelKey: 'logs.levelDebug' }
]

const entries = ref([])
const lastId = ref(-1)
const logFile = ref('')
const search = ref('')
const category = ref('all')
const activeLevels = ref(
  LEVELS.filter((l) => l.id !== 'debug').map((l) => l.id)
)
const follow = ref(true)
const unseen = ref(0)
const error = ref('')
const listEl = ref(null)

let timer = null
let disposed = false
let fetching = false

const api = async (url, options = {}) => {
  const response = await fetch(url, {
    ...options,
    headers: { ...(options.headers || {}), ...authHeaders() }
  })
  const data = await response.json().catch(() => ({}))
  if (!response.ok)
    throw new Error(data.detail || data.error || t('common.error'))
  return data
}

const fetchDelta = async (reset = false) => {
  if (fetching || disposed) return
  fetching = true
  try {
    if (reset) {
      entries.value = []
      lastId.value = -1
      unseen.value = 0
    }
    const data = await api(`/api/logs?since=${lastId.value}&limit=800`)
    if (
      !reset &&
      typeof data.last_id === 'number' &&
      data.last_id < lastId.value
    ) {
      fetching = false
      await fetchDelta(true)
      return
    }
    const incoming = Array.isArray(data.entries)
      ? data.entries
      : Array.isArray(data.logs)
        ? data.logs
        : []
    if (incoming.length) {
      entries.value.push(...incoming)
      if (entries.value.length > MAX_ENTRIES) {
        entries.value.splice(0, entries.value.length - MAX_ENTRIES)
      }
    }
    if (typeof data.last_id === 'number') lastId.value = data.last_id
    logFile.value = data.file || ''
    error.value = ''
  } catch (err) {
    error.value = err.message || t('logs.readError')
  } finally {
    fetching = false
  }
}

// --- Filtrado ---

const categories = computed(() => {
  const seen = new Set()
  entries.value.forEach((e) => {
    if (e.category) seen.add(e.category)
  })
  return Array.from(seen).sort()
})

const categoryLabel = (cat) =>
  has('logs.categories.' + cat) ? t('logs.categories.' + cat) : cat

const filtered = computed(() => {
  const term = search.value.trim().toLowerCase()
  const levels = activeLevels.value
  const cat = category.value
  return entries.value.filter((e) => {
    const lvl = (e.level || 'info').toLowerCase()
    if (!levels.includes(lvl)) return false
    if (cat !== 'all' && e.category !== cat) return false
    if (!term) return true
    return (
      (e.message || '').toLowerCase().includes(term) ||
      (e.detail || e.details || '').toLowerCase().includes(term) ||
      (e.category || '').toLowerCase().includes(term)
    )
  })
})

const visible = computed(() => {
  const list = filtered.value
  return list.length > MAX_RENDERED
    ? list.slice(list.length - MAX_RENDERED)
    : list
})

const hiddenCount = computed(() =>
  Math.max(0, filtered.value.length - visible.value.length)
)
const errorCount = computed(
  () =>
    entries.value.filter((e) => (e.level || '').toLowerCase() === 'error')
      .length
)
const warnCount = computed(
  () =>
    entries.value.filter((e) => (e.level || '').toLowerCase() === 'warn').length
)

const toggleLevel = (id) => {
  const current = activeLevels.value
  activeLevels.value = current.includes(id)
    ? current.filter((l) => l !== id)
    : [...current, id]
}

// --- Auto-scroll ---

const scrollToBottom = () => {
  const el = listEl.value
  if (!el) return
  el.scrollTop = el.scrollHeight
  unseen.value = 0
}

const onScroll = () => {
  const el = listEl.value
  if (!el) return
  const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 40
  if (atBottom) {
    unseen.value = 0
  } else if (follow.value) {
    follow.value = false
  }
}

const toggleFollow = () => {
  follow.value = !follow.value
  if (follow.value) nextTick(scrollToBottom)
}

watch(
  () => filtered.value.length,
  (count, previous) => {
    if (!props.active) return
    if (follow.value) {
      nextTick(scrollToBottom)
    } else if (count > (previous || 0)) {
      unseen.value += count - (previous || 0)
    }
  }
)

watch(
  () => props.logsSeq,
  (seq) => {
    if (seq !== lastId.value) fetchDelta()
  }
)

watch(
  () => props.active,
  (isActive) => {
    if (isActive) {
      fetchDelta()
      if (follow.value) nextTick(scrollToBottom)
    }
  }
)

// --- Acciones ---

const buildText = () =>
  filtered.value
    .map(
      (e) =>
        `${formatTime(e.time || e.timestamp)} ${levelLabel(e.level).padEnd(7)} [${categoryLabel(e.category)}] ${e.message}${e.detail || e.details ? ' — ' + (e.detail || e.details) : ''}`
    )
    .join('\n')

const copyAll = async () => {
  const text = buildText()
  if (!text) {
    props.notify(t('logs.nothingToCopy'))
    return
  }
  try {
    await navigator.clipboard.writeText(text)
    props.notify(t('logs.copied'))
  } catch {
    const area = document.createElement('textarea')
    area.value = text
    area.style.position = 'fixed'
    area.style.opacity = '0'
    document.body.appendChild(area)
    area.select()
    try {
      document.execCommand('copy')
      props.notify(t('logs.copied'))
    } catch {
      error.value = t('logs.copyError')
    }
    document.body.removeChild(area)
  }
}

const exportFile = async () => {
  try {
    const response = await fetch('/api/logs/export', {
      headers: { ...authHeaders() }
    })
    if (!response.ok) throw new Error(t('logs.exportError'))
    const blob = await response.blob()
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    const stamp = new Date().toISOString().slice(0, 19).replace(/[:T]/g, '-')
    link.href = url
    link.download = `telegrampv-${stamp}.log`
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    setTimeout(() => URL.revokeObjectURL(url), 2000)
    props.notify(t('logs.exported'))
  } catch (err) {
    error.value = err.message || t('logs.exportError')
  }
}

const clearLogs = () => {
  openConfirm({
    title: t('logs.clearModalTitle'),
    message: t('logs.clearModalText'),
    confirmText: t('logs.clearModalConfirm'),
    type: 'danger',
    action: async () => {
      try {
        const data = await api('/api/logs/clear', { method: 'POST' })
        entries.value = []
        unseen.value = 0
        if (typeof data.last_id === 'number') lastId.value = data.last_id
        await fetchDelta()
      } catch (err) {
        error.value = err.message || t('logs.clearError')
      }
    }
  })
}

// --- Formato ---

const formatTime = (value) => {
  if (typeof value === 'number') {
    const date = new Date(value * 1000)
    if (!Number.isNaN(date.getTime())) {
      const pad = (n, size = 2) => String(n).padStart(size, '0')
      return `${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}.${pad(date.getMilliseconds(), 3)}`
    }
  }
  if (typeof value === 'string' && value) return value
  return '--:--:--'
}

const levelLabel = (level = '') => {
  const lvl = level.toLowerCase()
  const key = 'logs.levelTag' + lvl.charAt(0).toUpperCase() + lvl.slice(1)
  return has(key) ? t(key) : t('logs.levelTagInfo')
}

onMounted(async () => {
  disposed = false
  await fetchDelta()
  await nextTick()
  scrollToBottom()
  timer = setInterval(() => {
    fetchDelta()
  }, 3000)
})

onUnmounted(() => {
  disposed = true
  clearInterval(timer)
})
</script>

<template>
  <section class="logs-view">
    <section class="logs-hero">
      <div class="logs-hero-info">
        <span class="hero-kicker">
          <ScrollText :size="13" /> {{ t('logs.kicker') }}
        </span>
        <h2>{{ t('logs.title') }}</h2>
        <p>
          {{ t('logs.text') }}
          <span v-if="logFile" class="log-path" :title="logFile">
            {{ t('logs.storedAt', { file: logFile }) }}
          </span>
        </p>
      </div>
      <div class="logs-counters">
        <div class="counter">
          <strong>{{ entries.length }}</strong>
          <span>{{ t('logs.entries') }}</span>
        </div>
        <div class="counter warn" :class="{ muted: !warnCount }">
          <strong>{{ warnCount }}</strong>
          <span>{{ t('logs.warnings') }}</span>
        </div>
        <div class="counter error" :class="{ muted: !errorCount }">
          <strong>{{ errorCount }}</strong>
          <span>{{ t('logs.errors') }}</span>
        </div>
      </div>
    </section>

    <section class="panel logs-panel">
      <div class="logs-toolbar">
        <div class="level-chips">
          <button
            v-for="level in LEVELS"
            :key="level.id"
            class="level-chip"
            :class="[level.id, { active: activeLevels.includes(level.id) }]"
            type="button"
            @click="toggleLevel(level.id)"
          >
            {{ t(level.labelKey) }}
          </button>
        </div>

        <div class="toolbar-right">
          <select
            v-model="category"
            class="cat-select"
            :aria-label="t('logs.filterSourceAria')"
          >
            <option value="all">{{ t('logs.allSources') }}</option>
            <option v-for="cat in categories" :key="cat" :value="cat">
              {{ categoryLabel(cat) }}
            </option>
          </select>

          <label class="search-box">
            <Search :size="14" />
            <input
              v-model="search"
              type="search"
              :placeholder="t('logs.searchPlaceholder')"
            />
          </label>

          <button
            class="tool-button"
            type="button"
            :class="{ on: follow }"
            :title="follow ? t('logs.followTitleOn') : t('logs.followTitleOff')"
            @click="toggleFollow"
          >
            <Pause v-if="follow" :size="14" />
            <Play v-else :size="14" />
            {{ follow ? t('logs.live') : t('logs.paused') }}
          </button>
          <button
            class="tool-button"
            type="button"
            :title="t('logs.copyTitle')"
            @click="copyAll"
          >
            <Copy :size="14" />
          </button>
          <button
            class="tool-button"
            type="button"
            :title="t('logs.exportTitle')"
            @click="exportFile"
          >
            <Download :size="14" />
          </button>
          <button
            class="tool-button danger"
            type="button"
            :title="t('logs.clearTitle')"
            @click="clearLogs"
          >
            <Trash2 :size="14" />
          </button>
        </div>
      </div>

      <div v-if="error" class="logs-error">{{ error }}</div>

      <div ref="listEl" class="logs-list" @scroll="onScroll">
        <div v-if="!filtered.length" class="empty-state">
          <ScrollText :size="28" />
          <p>
            {{ entries.length ? t('logs.emptyFiltered') : t('logs.emptyNone') }}
          </p>
          <small>{{
            entries.length
              ? t('logs.emptyFilteredHint')
              : t('logs.emptyNoneHint')
          }}</small>
        </div>

        <p v-else-if="hiddenCount" class="logs-truncated">
          {{
            t('logs.truncated', {
              shown: visible.length,
              total: filtered.length
            })
          }}
        </p>

        <article
          v-for="entry in visible"
          :key="entry.id"
          class="log-row"
          :class="(entry.level || 'info').toLowerCase()"
        >
          <span class="log-time">{{
            formatTime(entry.time || entry.timestamp)
          }}</span>
          <span
            class="log-level"
            :class="(entry.level || 'info').toLowerCase()"
          >
            {{ levelLabel(entry.level) }}
          </span>
          <span class="log-category">{{ categoryLabel(entry.category) }}</span>
          <span class="log-body">
            <span class="log-message">{{ entry.message }}</span>
            <small v-if="entry.detail || entry.details" class="log-detail">
              {{ entry.detail || entry.details }}
            </small>
          </span>
        </article>
      </div>

      <button
        v-if="unseen"
        class="jump-button"
        type="button"
        @click="((follow = true), scrollToBottom())"
      >
        <ArrowDownToLine :size="14" />
        {{
          unseen === 1
            ? t('logs.newEntry', { n: unseen })
            : t('logs.newEntries', { n: unseen })
        }}
      </button>
    </section>
  </section>
</template>

<style scoped>
.logs-view {
  display: flex;
  flex-direction: column;
  gap: 18px;
  min-width: 0;
}
.logs-hero {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 20px;
  padding: 26px 30px;
  border: 1px solid var(--user-border-light, #234765);
  border-radius: 18px;
  background: linear-gradient(
    110deg,
    var(--user-surface-light, #0e2032),
    var(--user-surface, #0b1a2a) 70%
  );
}
.hero-kicker {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: var(--user-primary, #38a7ff);
}
.logs-hero h2 {
  font:
    600 23px 'Space Grotesk',
    system-ui,
    sans-serif;
  margin: 8px 0 5px;
  color: #f1f7ff;
}
.logs-hero p {
  margin: 0;
  color: var(--user-text-dim, #728ba2);
  font-size: 13px;
  line-height: 1.5;
}
.logs-hero-info {
  min-width: 0;
  flex: 1;
}

.log-path {
  display: block;
  font-size: 11px;
  opacity: 0.75;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 100%;
}
.logs-counters {
  display: flex;
  gap: 10px;
  flex-shrink: 0;
}
.counter {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  min-width: 64px;
  padding: 10px 12px;
  border-radius: 12px;
  background: var(--user-bg-base, #07111f);
  border: 1px solid var(--user-border, #1b344b);
}
.counter strong {
  font:
    700 18px 'Space Grotesk',
    system-ui,
    sans-serif;
  color: #eef7ff;
}
.counter span {
  font-size: 10px;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: var(--user-text-dim, #728ba2);
}
.counter.warn strong {
  color: #f2c14b;
}
.counter.error strong {
  color: #e58b91;
}
.counter.muted {
  opacity: 0.45;
}
.counter.muted strong {
  color: var(--user-text-dim, #728ba2);
}

.logs-panel {
  display: flex;
  flex-direction: column;
  gap: 14px;
  position: relative;
  min-width: 0;
}
.logs-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}
.level-chips {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}
.level-chip {
  border: 1px solid var(--user-border-light, #234765);
  background: var(--user-bg-base, #07111f);
  color: var(--user-text-dim, #728ba2);
  border-radius: 20px;
  padding: 5px 12px;
  font:
    700 10px 'DM Sans',
    system-ui,
    sans-serif;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  cursor: pointer;
  transition: all 0.2s;
}
.level-chip:hover {
  border-color: var(--user-primary, #38a7ff);
}
.level-chip.active {
  color: #eef7ff;
  background: var(--user-icon-bg, #11385b);
  border-color: var(--user-border-light, #234765);
}
.level-chip.active.error {
  color: #e58b91;
  border-color: #4a2b2d;
  background: #251415;
}
.level-chip.active.warn {
  color: #f2c14b;
  border-color: #4a3c1b;
  background: #221a08;
}
.level-chip.active.success {
  color: #76c859;
  border-color: #234a1b;
  background: #0d220d;
}
.level-chip.active.info {
  color: #5ebcff;
  border-color: #1b3e5b;
  background: #0a1926;
}
.level-chip.active.debug {
  color: #9aa7b4;
  border-color: #2c3540;
  background: #141a21;
}

.toolbar-right {
  display: flex;
  align-items: center;
  gap: 7px;
  flex-wrap: wrap;
  min-width: 0;
}
.cat-select {
  appearance: none;
  -webkit-appearance: none;
  -moz-appearance: none;
  background-color: var(--user-bg-base, #07111f);
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 24 24' fill='none' stroke='%238aa0b4' stroke-width='2.5' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='m6 9 6 6 6-6'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 10px center;
  border: 1px solid var(--user-border-light, #234765);
  color: #dbe7f5;
  border-radius: 9px;
  padding: 7px 30px 7px 10px;
  outline: none;
  font: inherit;
  font-size: 12px;
  max-width: 196px;
  cursor: pointer;
}
.cat-select:focus {
  border-color: var(--user-primary, #38a7ff);
  box-shadow: 0 0 0 3px var(--user-glow, rgba(56, 167, 255, 0.15));
}
.search-box {
  display: flex;
  align-items: center;
  gap: 6px;
  background: var(--user-bg-base, #07111f);
  border: 1px solid var(--user-border-light, #234765);
  border-radius: 9px;
  padding: 0 10px;
  color: var(--user-text-dim, #728ba2);
  min-width: 0;
}
.search-box:focus-within {
  border-color: var(--user-primary, #38a7ff);
  box-shadow: 0 0 0 3px var(--user-glow, rgba(56, 167, 255, 0.15));
}
.search-box input {
  background: transparent;
  border: 0;
  outline: none;
  color: #dbe7f5;
  font: inherit;
  font-size: 12px;
  padding: 8px 0;
  width: 165px;
  min-width: 0;
}
.tool-button {
  display: flex;
  align-items: center;
  gap: 5px;
  border: 1px solid var(--user-border-light, #234765);
  background: var(--user-bg-base, #07111f);
  color: #dbe7f5;
  border-radius: 9px;
  padding: 7px 10px;
  font-size: 11px;
  cursor: pointer;
  transition: all 0.2s;
}
.tool-button:hover {
  background: var(--user-icon-bg, #11385b);
  border-color: var(--user-primary, #38a7ff);
  color: var(--user-accent, #5ebcff);
}
.tool-button.on {
  color: var(--user-accent, #5ebcff);
  border-color: var(--user-primary, #38a7ff);
}
.tool-button.danger:hover {
  background: #251415;
  border-color: #4a2b2d;
  color: #e58b91;
}

.logs-error {
  border: 1px solid #4a2b2d;
  background: #251415;
  color: #e58b91;
  border-radius: 9px;
  padding: 9px 12px;
  font-size: 12px;
}

.logs-list {
  height: calc(100vh - 360px);
  min-height: 280px;
  overflow-y: auto;
  overflow-x: hidden;
  background: var(--user-bg-base, #07111f);
  border: 1px solid var(--user-border, #1b344b);
  border-radius: 12px;
  padding: 6px 0;
}
.logs-list::-webkit-scrollbar {
  width: 8px;
}
.logs-list::-webkit-scrollbar-thumb {
  background: var(--user-border-light, #234765);
  border-radius: 4px;
}
.logs-truncated {
  margin: 4px 14px 8px;
  font-size: 11px;
  color: var(--user-text-dim, #728ba2);
  text-align: center;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  color: var(--user-text-dim, #728ba2);
  text-align: center;
  gap: 8px;
}
.empty-state p {
  margin: 0;
  font-size: 14px;
  font-weight: 500;
  color: #dbe7f5;
}
.empty-state small {
  font-size: 12px;
  opacity: 0.75;
}

.log-row {
  position: relative;
  display: grid;
  grid-template-columns: 88px 54px 96px minmax(0, 1fr);
  gap: 10px;
  align-items: baseline;
  padding: 5px 14px 5px 16px;
  font:
    12px/1.5 'JetBrains Mono',
    'Consolas',
    monospace;
}
.log-row.error {
  background: rgba(192, 81, 89, 0.07);
}
.log-row.debug {
  opacity: 0.62;
}
.log-row:hover {
  background: var(--user-surface, #0b1a2a);
}

.log-row::before {
  content: '';
  position: absolute;
  left: 0;
  top: 5px;
  bottom: 5px;
  width: 2px;
  border-radius: 2px;
  background: transparent;
}
.log-row.error::before {
  background: #c05159;
}
.log-row.warn::before {
  background: #c8961f;
}
.log-row.success::before {
  background: #479f29;
}
.log-row.info::before {
  background: #38a7ff;
}
.log-row.debug::before {
  background: #5c6773;
}

.log-time {
  color: var(--user-text-dim, #728ba2);
  font-size: 11px;
  white-space: nowrap;
}
.log-level {
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.5px;
  color: var(--user-text-dim, #728ba2);
  white-space: nowrap;
}
.log-level.error {
  color: #e58b91;
}
.log-level.warn {
  color: #f2c14b;
}
.log-level.success {
  color: #76c859;
}
.log-level.info {
  color: #5ebcff;
}
.log-level.debug {
  color: #9aa7b4;
}
.log-category {
  font-size: 10px;
  color: var(--user-text-dim, #728ba2);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.log-body {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.log-message {
  color: #dbe7f5;
  word-break: break-word;
}
.log-detail {
  color: var(--user-text-dim, #728ba2);
  font-size: 11px;
  word-break: break-word;
}

.jump-button {
  position: absolute;
  left: 50%;
  transform: translateX(-50%);
  bottom: 32px;
  display: flex;
  align-items: center;
  gap: 6px;
  border: 1px solid var(--user-primary, #38a7ff);
  background: var(--user-icon-bg, #11385b);
  color: var(--user-accent, #5ebcff);
  border-radius: 20px;
  padding: 7px 14px;
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.35);
}

@media (max-width: 900px) {
  .logs-hero {
    flex-direction: column;
    align-items: flex-start;
    padding: 22px;
  }
  .logs-hero-info {
    width: 100%;
    min-width: 0;
  }
  .logs-counters {
    width: 100%;
  }
  .counter {
    flex: 1;
  }
  .logs-list {
    height: calc(100vh - 430px);
  }
  .log-row {
    grid-template-columns: 66px 48px minmax(0, 1fr);
  }
  .log-category {
    display: none;
  }
}
@media (max-width: 580px) {
  .logs-hero {
    padding: 16px;
    gap: 14px;
    box-sizing: border-box;
    width: 100%;
  }
  .logs-hero h2 {
    font-size: 18px;
    word-break: break-word;
  }
  .logs-hero p {
    font-size: 12px;
    word-break: break-word;
  }
  .logs-counters {
    width: 100%;
    gap: 6px;
  }
  .counter {
    flex: 1 1 0;
    min-width: 0;
    padding: 8px 4px;
  }
  .counter strong {
    font-size: 15px;
  }
  .counter span {
    font-size: 9px;
  }
  .logs-panel {
    padding: 14px;
  }
  .logs-toolbar {
    flex-direction: column;
    align-items: stretch;
    gap: 10px;
  }
  .level-chips {
    width: 100%;
    gap: 4px;
  }
  .level-chip {
    padding: 4px 8px;
    font-size: 9px;
    flex: 1 1 auto;
    text-align: center;
  }
  .toolbar-right {
    width: 100%;
    gap: 6px;
  }
  .search-box {
    flex: 1 1 100%;
    box-sizing: border-box;
  }
  .search-box input {
    width: 100%;
  }
  .cat-select {
    max-width: 100%;
    flex: 1 1 100%;
  }
  .tool-button {
    flex: 1 1 auto;
    justify-content: center;
    padding: 6px 8px;
    font-size: 10px;
  }
  .log-row {
    grid-template-columns: 55px minmax(0, 1fr);
    font-size: 11px;
    padding: 5px 8px;
    gap: 6px;
  }
  .log-level,
  .log-category {
    display: none;
  }
}
</style>
