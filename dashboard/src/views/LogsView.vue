<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import { useAuthToken } from '../composables/useAuthToken'
import { useI18n } from '../i18n'
import { RefreshCw, Trash2 } from '../icons'

const { authHeaders } = useAuthToken()
const { t } = useI18n()

const logs = ref([])
let timer = null

const fetchLogs = async () => {
  try {
    const res = await fetch('/api/logs', { headers: authHeaders() })
    if (res.ok) {
      const data = await res.json()
      logs.value = data.logs || []
    }
  } catch (err) {
    console.error('Failed to fetch logs:', err)
  }
}

const clearLogsView = () => {
  logs.value = []
}

onMounted(() => {
  fetchLogs()
  timer = setInterval(fetchLogs, 3000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<template>
  <div class="panel logs-view">
    <div class="panel-heading">
      <h2>{{ t('logs.title') }}</h2>
      <div class="header-actions">
        <button class="btn-icon" @click="fetchLogs">
          <RefreshCw :size="16" />
        </button>
        <button class="btn-icon" @click="clearLogsView">
          <Trash2 :size="16" />
        </button>
      </div>
    </div>

    <div v-if="logs.length === 0" class="empty-state">
      <p>{{ t('logs.empty') }}</p>
    </div>

    <div v-else class="logs-console">
      <div
        v-for="entry in logs"
        :key="entry.id"
        class="log-line"
        :class="entry.level.toLowerCase()"
      >
        <span class="log-time">{{ entry.timestamp }}</span>
        <span class="log-level">{{ entry.level }}</span>
        <span class="log-cat">[{{ entry.category }}]</span>
        <span class="log-msg">{{ entry.message }} {{ entry.details }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.logs-view {
  padding: 24px;
}

.btn-icon {
  background: var(--user-surface-light, #0e2032);
  border: 1px solid var(--user-border-light, #234765);
  color: var(--user-text-dim, #728ba2);
  width: 32px;
  height: 32px;
  border-radius: 8px;
  display: grid;
  place-items: center;
  cursor: pointer;
}

.btn-icon:hover {
  color: #fff;
  background: var(--user-border, #1b344b);
}

.logs-console {
  background: #050b14;
  border: 1px solid var(--user-border, #1b344b);
  border-radius: 12px;
  padding: 14px;
  font-family: monospace;
  font-size: 12px;
  max-height: 500px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.log-line {
  display: flex;
  gap: 10px;
  line-height: 1.4;
}

.log-time {
  color: #5f7890;
}

.log-level {
  font-weight: 700;
  width: 60px;
}

.log-line.info .log-level {
  color: #38a7ff;
}

.log-line.warn .log-level {
  color: #ffc764;
}

.log-line.error .log-level {
  color: #ff4d4d;
}

.log-line.success .log-level {
  color: #39db9a;
}

.log-cat {
  color: #94a8bc;
}

.log-msg {
  color: #dbe7f5;
}
</style>
