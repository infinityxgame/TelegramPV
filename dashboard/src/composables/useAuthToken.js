import { ref } from 'vue'

const token = ref('')
const ready = ref(false)

export const isWailsRuntime = () => !!window.runtime

const STORAGE_KEY = 'telegrampv_api_token'

export const hasStoredToken = () => {
  if (isWailsRuntime()) return true
  try {
    return !!localStorage.getItem(STORAGE_KEY)
  } catch {
    return false
  }
}

export function useAuthToken() {
  const initToken = async () => {
    if (isWailsRuntime()) {
      token.value = 'wails-runtime'
    } else {
      try {
        token.value = localStorage.getItem(STORAGE_KEY) || ''
      } catch {
        token.value = ''
      }
    }
    ready.value = true
    return token.value
  }

  const setToken = (newToken) => {
    token.value = newToken
    try {
      localStorage.setItem(STORAGE_KEY, newToken)
    } catch (e) {
      console.error('Failed to save token to localStorage:', e)
    }
  }

  const clearToken = () => {
    token.value = ''
    try {
      localStorage.removeItem(STORAGE_KEY)
    } catch (e) {
      console.error('Failed to clear token from localStorage:', e)
    }
  }

  const authHeaders = () => {
    if (!token.value) return {}
    return {
      Authorization: `Bearer ${token.value}`
    }
  }

  return {
    token,
    ready,
    initToken,
    setToken,
    clearToken,
    authHeaders,
    isWailsRuntime
  }
}
