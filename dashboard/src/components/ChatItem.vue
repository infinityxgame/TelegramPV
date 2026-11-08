<script setup>
import { useI18n } from '../i18n'
import {
  Bot,
  CheckSquare,
  ExternalLink,
  Radio,
  Square,
  Trash2,
  User,
  Users
} from '../icons'

const props = defineProps({
  chat: {
    type: Object,
    required: true
  },
  selected: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['toggle-select', 'leave'])
const { t } = useI18n()

// Abrir el chat directamente en la aplicación de Telegram usando el protocolo tg://
const openInTelegram = () => {
  if (!props.chat) return
  let url
  if (props.chat.username) {
    url = `tg://resolve?domain=${props.chat.username}`
  } else {
    const rawId = String(props.chat.id)
    if (rawId.startsWith('-100')) {
      const cleanId = rawId.replace('-100', '')
      url = `tg://openmessage?chat_id=${cleanId}`
    } else if (rawId.startsWith('-')) {
      const cleanId = rawId.replace('-', '')
      url = `tg://openmessage?chat_id=${cleanId}`
    } else {
      url = `tg://openmessage?user_id=${rawId}`
    }
  }
  window.open(url, '_blank')
}
</script>

<template>
  <div class="chat-item-row" :class="{ 'is-selected': selected }">
    <button class="checkbox-btn" @click.stop="emit('toggle-select', chat.id)">
      <CheckSquare v-if="selected" class="check-icon checked" :size="18" />
      <Square v-else class="check-icon" :size="18" />
    </button>

    <div class="chat-type-icon" :class="chat.type">
      <Users v-if="chat.type === 'group'" :size="18" />
      <Radio v-else-if="chat.type === 'channel'" :size="18" />
      <Bot v-else-if="chat.type === 'bot'" :size="18" />
      <User v-else :size="18" />
    </div>

    <div class="chat-info">
      <div class="chat-title-row">
        <strong class="chat-title">{{ chat.title }}</strong>
        <span v-if="chat.username" class="chat-username"
          >@{{ chat.username }}</span
        >
      </div>
      <div class="chat-meta">
        <span class="chat-id-badge">{{ chat.id }}</span>
        <span class="chat-type-label">{{ t(`chats.${chat.type}`) }}</span>
      </div>
    </div>

    <div class="chat-actions">
      <button
        class="open-chat-btn"
        :title="t('chats.openAction')"
        @click.stop="openInTelegram"
      >
        <ExternalLink :size="16" />
      </button>

      <button
        class="delete-chat-btn"
        :title="t('chats.leaveAction')"
        @click.stop="emit('leave', chat)"
      >
        <Trash2 :size="16" />
      </button>
    </div>
  </div>
</template>

<style scoped>
.chat-item-row {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 12px 16px;
  background: var(--user-surface, #0b1a2a);
  border: 1px solid var(--user-border, #1b344b);
  border-radius: 12px;
  transition: all 0.2s ease;
}

.chat-item-row:hover {
  border-color: var(--user-border-light, #234765);
  background: var(--user-surface-light, #0e2032);
}

.chat-item-row.is-selected {
  border-color: var(--user-primary, #38a7ff);
  background: color-mix(in srgb, var(--user-primary), transparent 92%);
}

.checkbox-btn {
  background: transparent;
  border: 0;
  color: var(--user-text-dim, #728ba2);
  cursor: pointer;
  padding: 2px;
  display: grid;
  place-items: center;
}

.checkbox-btn:hover {
  color: var(--user-accent, #5ebcff);
}

.check-icon.checked {
  color: var(--user-primary, #38a7ff);
}

.chat-type-icon {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  display: grid;
  place-items: center;
  flex: none;
}

.chat-type-icon.group {
  background: #11385b;
  color: #38a7ff;
}

.chat-type-icon.channel {
  background: #26164d;
  color: #b69df2;
}

.chat-type-icon.bot {
  background: #47391b;
  color: #ffc764;
}

.chat-type-icon.user {
  background: #123d35;
  color: #5ce2b0;
}

.chat-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.chat-title-row {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.chat-title {
  color: #f2f7fc;
  font-size: 14px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.chat-username {
  color: var(--user-accent, #5ebcff);
  font-size: 12px;
  font-weight: 500;
}

.chat-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 11px;
}

.chat-id-badge {
  font-family: monospace;
  color: var(--user-text-dim, #728ba2);
}

.chat-type-label {
  color: var(--user-text-dim, #728ba2);
  text-transform: capitalize;
}

.chat-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.open-chat-btn {
  background: transparent;
  border: 0;
  color: var(--user-accent, #5ebcff);
  width: 32px;
  height: 32px;
  border-radius: 8px;
  cursor: pointer;
  display: grid;
  place-items: center;
  transition: all 0.2s ease;
}

.open-chat-btn:hover {
  background: var(--user-surface-light, #0e2032);
  color: var(--user-primary, #38a7ff);
}

.delete-chat-btn {
  background: transparent;
  border: 0;
  color: #d9828b;
  width: 32px;
  height: 32px;
  border-radius: 8px;
  cursor: pointer;
  display: grid;
  place-items: center;
  transition: all 0.2s ease;
}

.delete-chat-btn:hover {
  background: #48252c;
  color: #ffb0b0;
}
</style>
