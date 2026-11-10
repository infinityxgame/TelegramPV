import { reactive } from 'vue'
import { t } from '../i18n'

const modal = reactive({
  show: false,
  title: '',
  message: '',
  confirmText: '',
  cancelText: '',
  type: 'primary',
  action: null,
  cancelAction: null
})

export function useConfirmModal() {
  const openConfirm = (config) => {
    modal.title = config.title
    modal.message = config.message
    modal.confirmText = config.confirmText
    modal.cancelText =
      config.cancelText !== undefined ? config.cancelText : t('common.cancel')
    modal.type = config.type || 'primary'
    modal.action = config.action
    modal.cancelAction = config.cancelAction
    modal.show = true
  }

  const confirm = (config) => {
    return new Promise((resolve) => {
      openConfirm({
        ...config,
        action: () => resolve(true),
        cancelAction: () => resolve(false)
      })
    })
  }

  const handleConfirm = () => {
    if (modal.action) modal.action()
    modal.show = false
  }

  const handleCancel = () => {
    if (modal.cancelAction) modal.cancelAction()
    modal.show = false
  }

  return { modal, openConfirm, confirm, handleConfirm, handleCancel }
}
