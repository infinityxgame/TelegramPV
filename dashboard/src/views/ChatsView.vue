<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import ChatItem from '../components/ChatItem.vue'
import { useAuthToken } from '../composables/useAuthToken'
import { useConfirmModal } from '../composables/useConfirmModal'
import { useI18n } from '../i18n'
import {
  CheckSquare,
  Download,
  Layers,
  Loader2,
  RefreshCw,
  Search,
  Square,
  Trash2
} from '../icons'

const props = defineProps({
  category: {
    type: String,
    default: 'all'
  }
})

const emit = defineEmits(['update-counts'])

const { authHeaders } = useAuthToken()
const { t } = useI18n()
const { confirm } = useConfirmModal()

const chats = ref([])
const counts = ref({ all: 0, groups: 0, channels: 0, bots: 0, users: 0 })
const searchQuery = ref('')
const sortOrder = ref('az')
const privacyFilter = ref('all')
const roleFilter = ref('')
const loading = ref(false)
const syncing = ref(false)
const selectedIDs = ref(new Set())

const fetchChats = async () => {
  loading.value = true
  try {
    const url = `/api/chats?category=${props.category}&search=${encodeURIComponent(searchQuery.value)}`
    const res = await fetch(url, { headers: authHeaders() })
    if (res.ok) {
      const data = await res.json()
      chats.value = data.chats || []
      if (data.counts) {
        counts.value = data.counts
        emit('update-counts', data.counts)
      }
    }
  } catch (err) {
    console.error('Failed to fetch chats:', err)
  } finally {
    loading.value = false
  }
}

const syncChats = async () => {
  syncing.value = true
  try {
    const res = await fetch('/api/chats/sync', {
      method: 'POST',
      headers: authHeaders()
    })
    if (res.ok) {
      const data = await res.json()
      if (data.counts) {
        counts.value = data.counts
        emit('update-counts', data.counts)
      }
      await fetchChats()
    }
  } catch (err) {
    console.error('Failed to sync chats:', err)
  } finally {
    syncing.value = false
  }
}

const leaveChat = async (chat) => {
  const confirmed = await confirm({
    title: t('chats.confirmDeleteTitle'),
    message: t('chats.confirmDeleteMsg', chat.title),
    confirmText: t('chats.leaveAction'),
    cancelText: t('common.cancel')
  })

  if (!confirmed) return

  try {
    const res = await fetch('/api/chats/leave', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...authHeaders()
      },
      body: JSON.stringify({ chat_id: chat.id })
    })

    if (res.ok) {
      chats.value = chats.value.filter((c) => c.id !== chat.id)
      selectedIDs.value.delete(chat.id)
      await fetchChats()
    }
  } catch (err) {
    console.error('Failed to leave chat:', err)
  }
}

const deleteSelectedBatch = async () => {
  if (selectedIDs.value.size === 0) return

  const count = selectedIDs.value.size
  const confirmed = await confirm({
    title: t('chats.confirmBatchDeleteTitle'),
    message: t('chats.confirmBatchDeleteMsg', count),
    confirmText: t('chats.leaveAction'),
    cancelText: t('common.cancel')
  })

  if (!confirmed) return

  try {
    const ids = Array.from(selectedIDs.value)
    const res = await fetch('/api/chats/leave', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...authHeaders()
      },
      body: JSON.stringify({ chat_ids: ids })
    })

    if (res.ok) {
      chats.value = chats.value.filter((c) => !selectedIDs.value.has(c.id))
      selectedIDs.value.clear()
      await fetchChats()
    }
  } catch (err) {
    console.error('Failed batch deletion:', err)
  }
}

const exportTXT = async () => {
  try {
    await fetch(`/api/chats/export?category=${props.category}&format=json`, {
      headers: authHeaders()
    })
  } catch (err) {
    console.error('Failed to export TXT:', err)
  }
}

const toggleSelect = (id) => {
  if (selectedIDs.value.has(id)) {
    selectedIDs.value.delete(id)
  } else {
    selectedIDs.value.add(id)
  }
}

const displayedChats = computed(() => {
  let list = chats.value ? [...chats.value] : []

  // Filtro de privacidad para Grupos y Canales
  if (['all', 'groups', 'channels'].includes(props.category)) {
    if (privacyFilter.value === 'public') {
      list = list.filter((c) => Boolean(c.username && c.username.trim()))
    } else if (privacyFilter.value === 'private') {
      list = list.filter((c) => !c.username || !c.username.trim())
    }
  }

  // Filtro Combinado de Rol y Propiedad
  if (roleFilter.value === 'creator') {
    list = list.filter((c) => Boolean(c.is_creator))
  } else if (roleFilter.value === 'admin') {
    list = list.filter((c) => Boolean(c.is_admin || c.is_creator))
  }

  // Ordenamiento alfabético A-Z / Z-A
  if (sortOrder.value === 'az') {
    list.sort((a, b) =>
      (a.title || '').localeCompare(b.title || '', undefined, {
        sensitivity: 'base',
        numeric: true
      })
    )
  } else if (sortOrder.value === 'za') {
    list.sort((a, b) =>
      (b.title || '').localeCompare(a.title || '', undefined, {
        sensitivity: 'base',
        numeric: true
      })
    )
  }

  return list
})

const toggleSelectAll = () => {
  if (allSelected.value) {
    const displayedSet = new Set(displayedChats.value.map((c) => c.id))
    selectedIDs.value = new Set(
      Array.from(selectedIDs.value).filter((id) => !displayedSet.has(id))
    )
  } else {
    const newSet = new Set(selectedIDs.value)
    displayedChats.value.forEach((c) => newSet.add(c.id))
    selectedIDs.value = newSet
  }
}

const allSelected = computed(() => {
  if (displayedChats.value.length === 0) return false
  return displayedChats.value.every((c) => selectedIDs.value.has(c.id))
})

const toggleRoleFilter = (role) => {
  if (roleFilter.value === role) {
    roleFilter.value = ''
  } else {
    roleFilter.value = role
  }
}

watch(
  () => props.category,
  () => {
    privacyFilter.value = 'all'
    roleFilter.value = ''
    selectedIDs.value.clear()
    fetchChats()
  }
)

let searchTimeout = null
watch(searchQuery, () => {
  clearTimeout(searchTimeout)
  searchTimeout = setTimeout(fetchChats, 300)
})

onMounted(fetchChats)
</script>

<template>
  <div class="chats-view">
    <!-- Header Toolbar -->
    <div class="chats-toolbar">
      <div class="search-box">
        <Search :size="18" class="search-icon" />
        <input
          v-model="searchQuery"
          type="text"
          :placeholder="t('chats.searchPlaceholder')"
        />
      </div>

      <div class="sort-box">
        <select v-model="sortOrder" class="sort-select">
          <option value="az">{{ t('chats.sortAZ') }}</option>
          <option value="za">{{ t('chats.sortZA') }}</option>
          <option value="recent">{{ t('chats.sortRecent') }}</option>
        </select>
      </div>

      <div class="toolbar-actions">
        <button class="btn-sync" :disabled="syncing" @click="syncChats">
          <Loader2 v-if="syncing" class="spin" :size="16" />
          <RefreshCw v-else :size="16" />
          <span>{{
            syncing ? t('chats.syncing') : t('chats.syncButton')
          }}</span>
        </button>

        <button class="btn-export" @click="exportTXT">
          <Download :size="16" />
          <span>{{ t('chats.exportButton') }}</span>
        </button>
      </div>
    </div>

    <!-- Filter & Selection Bar -->
    <div class="filter-selection-bar">
      <!-- Filtros de Privacidad (Todos, Públicos, Privados) -->
      <div
        v-if="['all', 'groups', 'channels'].includes(category)"
        class="privacy-filters"
      >
        <button
          class="filter-pill"
          :class="{ active: privacyFilter === 'all' }"
          @click="privacyFilter = 'all'"
        >
          {{ t('chats.filterAll') }}
        </button>
        <button
          class="filter-pill"
          :class="{ active: privacyFilter === 'public' }"
          @click="privacyFilter = 'public'"
        >
          {{ t('chats.filterPublic') }}
        </button>
        <button
          class="filter-pill"
          :class="{ active: privacyFilter === 'private' }"
          @click="privacyFilter = 'private'"
        >
          {{ t('chats.filterPrivate') }}
        </button>
      </div>

      <!-- Filtros de Rol y Propiedad para Grupos y Canales (Switches conmutables) -->
      <div
        v-if="['all', 'groups', 'channels'].includes(category)"
        class="privacy-filters"
      >
        <label
          class="filter-switch-label"
          :class="{ active: roleFilter === 'creator' }"
          @click.prevent="toggleRoleFilter('creator')"
        >
          <span class="switch">
            <input type="checkbox" :checked="roleFilter === 'creator'" />
            <span></span>
          </span>
          <span>{{ t('chats.filterRoleOwner') }}</span>
        </label>

        <label
          class="filter-switch-label"
          :class="{ active: roleFilter === 'admin' }"
          @click.prevent="toggleRoleFilter('admin')"
        >
          <span class="switch">
            <input type="checkbox" :checked="roleFilter === 'admin'" />
            <span></span>
          </span>
          <span>{{ t('chats.filterRoleAdmin') }}</span>
        </label>

        <span class="results-count">
          {{ displayedChats.length }} {{ t('chats.resultsCount') }}
        </span>
      </div>

      <!-- Filtro de Propiedad para Bots (Switch conmutable) -->
      <div v-else-if="category === 'bots'" class="privacy-filters">
        <label
          class="filter-switch-label"
          :class="{ active: roleFilter === 'creator' }"
          @click.prevent="toggleRoleFilter('creator')"
        >
          <span class="switch">
            <input type="checkbox" :checked="roleFilter === 'creator'" />
            <span></span>
          </span>
          <span>{{ t('chats.filterBotsOwner') }}</span>
        </label>

        <span class="results-count">
          {{ displayedChats.length }} {{ t('chats.resultsCount') }}
        </span>
      </div>

      <!-- Fallback contador -->
      <div
        v-else-if="!['all', 'groups', 'channels', 'bots'].includes(category)"
        class="privacy-filters"
      >
        <span class="results-count">
          {{ displayedChats.length }} {{ t('chats.resultsCount') }}
        </span>
      </div>

      <!-- Selección múltiple fijada a la derecha con acciones a la izquierda -->
      <div class="selection-controls">
        <span v-if="selectedIDs.size > 0" class="selected-count">
          {{ selectedIDs.size }} / {{ displayedChats.length }}
        </span>

        <button
          v-if="selectedIDs.size > 0"
          class="btn-batch-delete"
          @click="deleteSelectedBatch"
        >
          <Trash2 :size="16" />
          <span>{{ t('chats.deleteSelected', selectedIDs.size) }}</span>
        </button>

        <button
          v-if="displayedChats.length > 0"
          class="btn-select-all"
          @click="toggleSelectAll"
        >
          <CheckSquare v-if="allSelected" :size="16" />
          <Square v-else :size="16" />
          <span>{{
            allSelected ? t('chats.deselectAll') : t('chats.selectAll')
          }}</span>
        </button>
      </div>
    </div>

    <!-- Loading State -->
    <div v-if="loading" class="chats-loading">
      <Loader2 class="spin" :size="32" />
      <p>{{ t('common.loading') }}</p>
    </div>

    <!-- Empty State -->
    <div v-else-if="displayedChats.length === 0" class="empty-state">
      <Layers :size="48" />
      <p>{{ t('chats.emptyTitle') }}</p>
      <small>{{ t('chats.emptyDesc') }}</small>
    </div>

    <!-- Chats List -->
    <div v-else class="chats-list">
      <ChatItem
        v-for="chat in displayedChats"
        :key="chat.id"
        :chat="chat"
        :selected="selectedIDs.has(chat.id)"
        @toggle-select="toggleSelect"
        @leave="leaveChat"
      />
    </div>
  </div>
</template>

<style scoped>
.chats-view {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.chats-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
}

.search-box {
  position: relative;
  flex: 1;
  min-width: 260px;
}

.search-icon {
  position: absolute;
  left: 14px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--user-text-dim, #728ba2);
}

.search-box input {
  width: 100%;
  background: var(--user-surface, #0b1a2a);
  border: 1px solid var(--user-border, #1b344b);
  border-radius: 12px;
  padding: 12px 14px 12px 42px;
  color: #dbe7f5;
  font-size: 14px;
  outline: none;
}

.search-box input:focus {
  border-color: var(--user-primary, #38a7ff);
  box-shadow: 0 0 0 3px var(--user-glow);
}

.sort-box {
  position: relative;
  display: flex;
  align-items: center;
}

.sort-select {
  background-color: var(--user-surface, #0b1a2a);
  border: 1px solid var(--user-border, #1b344b);
  border-radius: 12px;
  padding: 12px 36px 12px 16px;
  color: #dbe7f5;
  font-size: 14px;
  font-weight: 500;
  outline: none;
  cursor: pointer;
  appearance: none;
  -webkit-appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='16' height='16' viewBox='0 0 24 24' fill='none' stroke='%23728ba2' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='m6 9 6 6 6-6'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 12px center;
}

.sort-select:focus {
  border-color: var(--user-primary, #38a7ff);
  box-shadow: 0 0 0 3px var(--user-glow);
}

.toolbar-actions {
  display: flex;
  gap: 10px;
}

.btn-sync,
.btn-export {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 16px;
  border-radius: 10px;
  font-weight: 600;
  font-size: 13px;
  cursor: pointer;
  border: 0;
}

.btn-sync {
  background: var(--user-gradient, #38a7ff);
  color: var(--user-bg-base, #07111f);
}

.btn-sync:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 6px 18px var(--user-glow);
}

.btn-export {
  background: var(--user-surface-light, #0e2032);
  border: 1px solid var(--user-border-light, #234765);
  color: #dbe7f5;
}

.btn-export:hover {
  background: var(--user-border, #1b344b);
}

.filter-selection-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
}

.privacy-filters {
  display: flex;
  align-items: center;
  gap: 8px;
}

.filter-pill {
  background: var(--user-surface, #0b1a2a);
  border: 1px solid var(--user-border, #1b344b);
  color: var(--user-text-dim, #728ba2);
  padding: 7px 16px;
  border-radius: 20px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
}

.filter-pill:hover {
  background: var(--user-surface-light, #0e2032);
  color: #dbe7f5;
  border-color: var(--user-border-light, #234765);
}

.filter-pill.active {
  background: var(--user-primary, #38a7ff);
  color: #ffffff;
  border-color: transparent;
  box-shadow: 0 4px 12px
    color-mix(in srgb, var(--user-primary), transparent 70%);
}

.filter-switch-label {
  display: inline-flex;
  align-items: center;
  gap: 9px;
  background: transparent;
  border: none;
  padding: 2px 4px;
  color: var(--user-text-dim, #728ba2);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
  user-select: none;
  -webkit-user-select: none;
}

.filter-switch-label:hover {
  color: #dbe7f5;
}

.filter-switch-label.active {
  color: #ffffff;
}

.filter-switch-label .switch span {
  width: 48px;
  height: 26px;
  background: var(--user-surface, #0b1a2a);
  border: 1px solid var(--user-border, #1b344b);
  border-radius: 20px;
  position: relative;
  cursor: pointer;
  transition: all 0.25s cubic-bezier(0.175, 0.885, 0.32, 1.275);
}

.filter-switch-label .switch span:after {
  content: '';
  position: absolute;
  width: 18px;
  height: 18px;
  top: 3px;
  left: 3px;
  background: #a5bad0;
  border-radius: 50%;
  transition: all 0.25s cubic-bezier(0.175, 0.885, 0.32, 1.275);
}

.filter-switch-label.active .switch span {
  background: var(--user-primary, #38a7ff);
  border-color: transparent;
  box-shadow: 0 4px 12px
    color-mix(in srgb, var(--user-primary), transparent 70%);
}

.filter-switch-label.active .switch span:after {
  left: 25px;
  background: #ffffff;
}

.results-count {
  font-size: 13px;
  color: var(--user-text-dim, #728ba2);
  margin-left: 8px;
  font-weight: 500;
}

.selection-controls {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 12px;
}

.btn-select-all {
  display: flex;
  align-items: center;
  gap: 8px;
  background: transparent;
  border: 0;
  color: var(--user-accent, #5ebcff);
  font-weight: 600;
  font-size: 13px;
  cursor: pointer;
  padding: 6px 10px;
  border-radius: 8px;
}

.btn-select-all:hover {
  background: var(--user-surface-light, #0e2032);
}

.selected-count {
  font-size: 12px;
  color: var(--user-text-dim, #728ba2);
}

.btn-batch-delete {
  display: flex;
  align-items: center;
  gap: 8px;
  background: #3d171d;
  border: 1px solid #732a36;
  color: #ffadad;
  padding: 6px 14px;
  border-radius: 8px;
  font-weight: 600;
  font-size: 12px;
  cursor: pointer;
}

.btn-batch-delete:hover {
  background: #48252c;
  color: #ffb0b0;
}

.chats-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.chats-loading {
  text-align: center;
  padding: 48px;
  color: var(--user-text-dim, #728ba2);
}

.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
