import { Browser } from '@wailsio/runtime'
import { isWailsRuntime } from './useAuthToken'

export function openExternal(url) {
  if (!url) return
  if (isWailsRuntime()) {
    Browser.OpenURL(url)
  } else {
    window.open(url, '_blank')
  }
}
