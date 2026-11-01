import { reactive } from 'vue'

const state = reactive({
  isOpen: false,
  title: '',
  message: '',
  confirmText: '',
  cancelText: '',
  onConfirm: null,
  onCancel: null
})

export function useConfirmModal() {
  const confirm = (options) => {
    return new Promise((resolve) => {
      state.title = options.title || 'Confirm'
      state.message = options.message || 'Are you sure?'
      state.confirmText = options.confirmText || 'Confirm'
      state.cancelText = options.cancelText || 'Cancel'

      state.onConfirm = () => {
        state.isOpen = false
        resolve(true)
      }

      state.onCancel = () => {
        state.isOpen = false
        resolve(false)
      }

      state.isOpen = true
    })
  }

  const close = () => {
    state.isOpen = false
  }

  return {
    state,
    confirm,
    close
  }
}
