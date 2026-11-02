<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import ChatItem from '../components/ChatItem.vue'
import { useAuthToken } from '../composables/useAuthToken'
import { useConfirmModal } from '../composables/useConfirmModal'
import { useI18n } from '../i18n'
import { Download, Layers, Loader2, RefreshCw, Search, Trash2 } from '../icons'

const props = defineProps({
  category: {
    type: String,
    default: 'all'
  }
})

const { authHeaders } = useAuthToken()
const { t } = useI18n()
const { confirm } = useConfirmModal()

const chats = ref([])
const counts = ref({ all: 0, groups: 0, channels: 0, bots: 0, users: 0 })
const searchQuery = ref('')
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
    }
  } catch (err) {
    console.error('Failed batch deletion:', err)
  }
}

const exportTXT = () => {
  window.open(`/api/chats/export?category=${props.category}`, '_blank')
}

const toggleSelect = (id) => {
  if (selectedIDs.value.has(id)) {
    selectedIDs.value.delete(id)
  } else {
    selectedIDs.value.add(id)
  }
}

const toggleSelectAll = () => {
  if (selectedIDs.value.size === chats.value.length) {
    selectedIDs.value.clear()
  } else {
    selectedIDs.value = new Set(chats.value.map((c) => c.id))
  }
}

const allSelected = computed(
  () => chats.value.length > 0 && selectedIDs.value.size === chats.value.length
)

watch(() => props.category, fetchChats)

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

    <!-- Batch Selection Bar -->
    <div v-if="chats.length > 0" class="selection-bar">
      <button class="btn-select-all" @click="toggleSelectAll">
        <span>{{
          allSelected ? t('chats.deselectAll') : t('chats.selectAll')
        }}</span>
      </button>

      <span v-if="selectedIDs.size > 0" class="selected-count">
        {{ selectedIDs.size }} / {{ chats.length }}
      </span>

      <button
        v-if="selectedIDs.size > 0"
        class="btn-batch-delete"
        @click="deleteSelectedBatch"
      >
        <Trash2 :size="16" />
        <span>{{ t('chats.deleteSelected', selectedIDs.size) }}</span>
      </button>
    </div>

    <!-- Loading State -->
    <div v-if="loading" class="chats-loading">
      <Loader2 class="spin" :size="32" />
      <p>{{ t('common.loading') }}</p>
    </div>

    <!-- Empty State -->
    <div v-else-if="chats.length === 0" class="empty-state">
      <Layers :size="48" />
      <p>{{ t('chats.emptyTitle') }}</p>
      <small>{{ t('chats.emptyDesc') }}</small>
    </div>

    <!-- Chats List -->
    <div v-else class="chats-list">
      <ChatItem
        v-for="chat in chats"
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
  min-width: 280px;
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

.selection-bar {
  display: flex;
  align-items: center;
  gap: 16px;
  background: var(--user-surface, #0b1a2a);
  border: 1px solid var(--user-border, #1b344b);
  padding: 10px 16px;
  border-radius: 12px;
}

.btn-select-all {
  background: transparent;
  border: 0;
  color: var(--user-accent, #5ebcff);
  font-weight: 600;
  font-size: 13px;
  cursor: pointer;
}

.selected-count {
  font-size: 12px;
  color: var(--user-text-dim, #728ba2);
}

.btn-batch-delete {
  margin-left: auto;
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
