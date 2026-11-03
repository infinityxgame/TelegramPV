<script setup>
import { computed } from 'vue'
import { AlertCircle, HelpCircle, X } from '../icons'
import { useI18n } from '../i18n'

const { t } = useI18n()

const props = defineProps({
  show: Boolean,
  title: { type: String, default: '' },
  message: { type: String, default: '' },
  confirmText: { type: String, default: '' },
  cancelText: { type: String, default: undefined },
  type: { type: String, default: 'primary' }
})

const confirmLabel = computed(() => props.confirmText || t('common.confirm'))
const cancelLabel = computed(() =>
  props.cancelText === undefined ? t('common.cancel') : props.cancelText
)

const emit = defineEmits(['confirm', 'cancel'])
</script>

<template>
  <Teleport to="body">
    <Transition name="fade">
      <div v-if="show" class="modal-overlay" @click.self="emit('cancel')">
        <div class="modal-content" :class="type">
          <div class="modal-header">
            <div class="header-icon" :class="type">
              <AlertCircle v-if="type === 'danger'" :size="20" />
              <HelpCircle v-else :size="20" />
            </div>
            <h3>{{ title }}</h3>
            <button class="close-btn" @click="emit('cancel')">
              <X :size="18" />
            </button>
          </div>
          <div class="modal-body">
            <p>{{ message }}</p>
          </div>
          <div class="modal-footer">
            <button
              v-if="cancelLabel"
              class="cancel-btn"
              @click="emit('cancel')"
            >
              {{ cancelLabel }}
            </button>
            <button class="confirm-btn" :class="type" @click="emit('confirm')">
              {{ confirmLabel }}
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: rgba(2, 8, 19, 0.75);
  backdrop-filter: blur(8px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10000;
}

.modal-content {
  background: var(--user-surface, #0b1a2a);
  border: 1px solid var(--user-border-light, #234765);
  border-radius: 18px;
  width: min(420px, 90%);
  box-shadow:
    0 25px 50px -12px rgba(0, 0, 0, 0.5),
    0 0 35px var(--user-glow, rgba(56, 167, 255, 0.15));
  overflow: hidden;
}

.modal-content.danger {
  border-color: rgba(232, 136, 136, 0.4);
  box-shadow:
    0 25px 50px -12px rgba(0, 0, 0, 0.5),
    0 0 45px rgba(232, 136, 136, 0.25);
  animation: modalShake 0.5s cubic-bezier(0.36, 0.07, 0.19, 0.97) both;
}

@keyframes modalShake {
  10%,
  90% {
    transform: translate3d(-1px, 0, 0);
  }
  20%,
  80% {
    transform: translate3d(2px, 0, 0);
  }
  30%,
  50%,
  70% {
    transform: translate3d(-4px, 0, 0);
  }
  40%,
  60% {
    transform: translate3d(4px, 0, 0);
  }
}

.modal-header {
  padding: 20px 24px;
  display: flex;
  align-items: center;
  gap: 12px;
  border-bottom: 1px solid var(--user-border, #1b344b);
  position: relative;
}

.header-icon {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  display: grid;
  place-items: center;
}

.header-icon.primary {
  background: var(--user-icon-bg, #11385b);
  color: var(--user-accent, #5ebcff);
}
.header-icon.danger {
  background: #48252c;
  color: #ffadad;
}

.modal-header h3 {
  margin: 0;
  font-size: 16px;
  color: #f1f7ff;
}

.close-btn {
  margin-left: auto;
  background: transparent;
  border: 0;
  color: var(--user-text-dim, #728ba2);
  cursor: pointer;
  padding: 4px;
}

.close-btn:hover {
  color: var(--user-accent, #5ebcff);
}

.modal-body {
  padding: 24px;
  color: var(--user-text-dim, #728ba2);
  font-size: 14px;
  line-height: 1.6;
}

.modal-body p {
  margin: 0;
  white-space: pre-line;
}

.modal-footer {
  padding: 16px 24px;
  background: var(--user-bg-base, #07111f);
  border-top: 1px solid var(--user-border, #1b344b);
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

button {
  padding: 10px 18px;
  border-radius: 10px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.cancel-btn {
  background: var(--user-surface-light, #0e2032);
  border: 1px solid var(--user-border-light, #234765);
  color: var(--user-text-dim, #728ba2);
}

.cancel-btn:hover {
  background: var(--user-border, #1b344b);
  color: #fff;
}

.confirm-btn {
  border: 0;
}

.confirm-btn.primary {
  background: var(--user-primary, #38a7ff);
  color: var(--user-bg-base, #07111f);
}
.confirm-btn.primary:hover {
  opacity: 0.9;
  box-shadow: 0 4px 15px var(--user-glow, rgba(56, 167, 255, 0.15));
}

.confirm-btn.danger {
  background: #e88888;
  color: #041322;
}
.confirm-btn.danger:hover {
  background: #ffadad;
  box-shadow: 0 4px 15px rgba(232, 136, 136, 0.3);
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.25s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
