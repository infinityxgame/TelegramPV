<script setup>
import { useConfirmModal } from '../composables/useConfirmModal'
import { AlertCircle, X } from '../icons'

const { state } = useConfirmModal()
</script>

<template>
  <div v-if="state.isOpen" class="modal-overlay" @click.self="state.onCancel">
    <div class="modal-content">
      <div class="modal-header">
        <div class="modal-title">
          <AlertCircle class="modal-icon" />
          <h3>{{ state.title }}</h3>
        </div>
        <button class="modal-close" @click="state.onCancel">
          <X :size="18" />
        </button>
      </div>
      <div class="modal-body">
        <p>{{ state.message }}</p>
      </div>
      <div class="modal-footer">
        <button class="btn-cancel" @click="state.onCancel">
          {{ state.cancelText }}
        </button>
        <button class="btn-confirm" @click="state.onConfirm">
          {{ state.confirmText }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.7);
  backdrop-filter: blur(6px);
  z-index: 99999;
  display: grid;
  place-items: center;
  padding: 16px;
}

.modal-content {
  background: var(--user-surface, #0b1a2a);
  border: 1px solid var(--user-border, #1b344b);
  border-radius: 16px;
  width: 100%;
  max-width: 440px;
  padding: 24px;
  box-shadow: 0 20px 50px rgba(0, 0, 0, 0.5);
  animation: modalIn 0.25s cubic-bezier(0.16, 1, 0.3, 1);
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 14px;
}

.modal-title {
  display: flex;
  align-items: center;
  gap: 10px;
  color: #f2f7fc;
}

.modal-title h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
}

.modal-icon {
  color: #ff4d4d;
}

.modal-close {
  background: transparent;
  border: 0;
  color: var(--user-text-dim, #728ba2);
  cursor: pointer;
  padding: 4px;
  border-radius: 6px;
}

.modal-close:hover {
  color: #fff;
  background: var(--user-surface-light, #0e2032);
}

.modal-body p {
  margin: 0;
  color: var(--user-text-dim, #728ba2);
  font-size: 14px;
  line-height: 1.5;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 24px;
}

.btn-cancel {
  background: var(--user-surface-light, #0e2032);
  border: 1px solid var(--user-border-light, #234765);
  color: #dbe7f5;
  padding: 10px 18px;
  border-radius: 10px;
  font-weight: 600;
  font-size: 13px;
  cursor: pointer;
}

.btn-cancel:hover {
  background: var(--user-border, #1b344b);
}

.btn-confirm {
  background: #ff4d4d;
  border: 0;
  color: #fff;
  padding: 10px 18px;
  border-radius: 10px;
  font-weight: 600;
  font-size: 13px;
  cursor: pointer;
}

.btn-confirm:hover {
  background: #ff3333;
  box-shadow: 0 4px 14px rgba(255, 77, 77, 0.4);
}

@keyframes modalIn {
  from {
    opacity: 0;
    transform: scale(0.94);
  }
  to {
    opacity: 1;
    transform: scale(1);
  }
}
</style>
