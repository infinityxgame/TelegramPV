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
  const openConfirm = (options) => {
    return new Promise((resolve) => {
      state.title = options.title || 'Confirm'
      state.message = options.message || 'Are you sure?'
      state.confirmText = options.confirmText || 'Confirm'
      state.cancelText = options.cancelText || 'Cancel'

      state.onConfirm = () => {
        state.isOpen = false
        if (options.action) options.action()
        resolve(true)
      }

      state.onCancel = () => {
        state.isOpen = false
        if (options.cancelAction) options.cancelAction()
        resolve(false)
      }

      state.isOpen = true
    })
  }

  const confirm = openConfirm

  const close = () => {
    state.isOpen = false
  }

  return {
    state,
    confirm,
    openConfirm,
    close
  }
}
