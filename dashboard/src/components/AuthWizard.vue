<script setup>
import { computed, ref, watch } from 'vue'
import { useAuthToken } from '../composables/useAuthToken'
import { openExternal } from '../composables/useExternalLink'
import { useI18n } from '../i18n'
import {
  AlertCircle,
  ArrowLeft,
  ArrowRight,
  CheckCircle2,
  KeyRound,
  Loader2,
  Phone,
  ShieldCheck
} from '../icons'

const props = defineProps({
  authStatus: {
    type: Object,
    required: true
  }
})

const emit = defineEmits(['auth-success'])

const { authHeaders } = useAuthToken()
const { t } = useI18n()

const serverStep = computed(() => {
  if (
    !props.authStatus.has_credentials ||
    props.authStatus.state === 'UNCONFIGURED'
  )
    return 1
  if (
    props.authStatus.state === 'NOT_LOGGED_IN' ||
    props.authStatus.state === 'NEED_PHONE'
  )
    return 2
  if (props.authStatus.state === 'WAITING_CODE') return 3
  if (props.authStatus.state === 'WAITING_2FA') return 4
  if (props.authStatus.state === 'LOGGED_IN') return 5
  return 1
})

const stepOverride = ref(null)
const currentStep = computed(() => stepOverride.value ?? serverStep.value)

const apiId = ref('')
const apiHash = ref('')
const phoneNumber = ref(props.authStatus.phone || '')
const code = ref('')
const password = ref('')
const phoneCodeHash = ref('')

watch(
  () => props.authStatus,
  (val) => {
    if (val && val.phone && !phoneNumber.value) {
      phoneNumber.value = val.phone
    }
  },
  { immediate: true }
)

const loading = ref(false)
const errorMessage = ref('')

watch(currentStep, () => {
  errorMessage.value = ''
})

const apiCall = async (url, body) => {
  loading.value = true
  errorMessage.value = ''
  try {
    const res = await fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...authHeaders()
      },
      body: JSON.stringify(body)
    })

    const text = await res.text()
    if (!text || text.trim() === '') {
      throw new Error(t('auth.errNoResponse'))
    }

    let data
    try {
      data = JSON.parse(text)
    } catch {
      throw new Error(t('auth.errInvalidResponse'))
    }

    if (!res.ok) {
      throw new Error(data.error || data.detail || t('auth.errProcess'))
    }

    stepOverride.value = null
    return data
  } catch (err) {
    errorMessage.value = err.message
    throw err
  } finally {
    loading.value = false
  }
}

const saveCredentials = async () => {
  const trimmedApiId = apiId.value.trim()
  const trimmedApiHash = apiHash.value.trim()
  if (!trimmedApiId || !trimmedApiHash) {
    errorMessage.value = t('auth.errFillBoth')
    return
  }
  if (!/^\d+$/.test(trimmedApiId)) {
    errorMessage.value = t('auth.errApiIdFormat')
    return
  }
  if (!/^[a-fA-F0-9]{32}$/.test(trimmedApiHash)) {
    errorMessage.value = t('auth.errApiHashFormat')
    return
  }

  try {
    await apiCall('/api/auth/credentials', {
      api_id: trimmedApiId,
      api_hash: trimmedApiHash
    })
    stepOverride.value = 2
  } catch {
    // Manejado en apiCall
  }
}

const sendCode = async () => {
  const trimmedPhone = phoneNumber.value.trim()
  if (!trimmedPhone) {
    errorMessage.value = t('auth.step2Desc')
    return
  }

  try {
    const data = await apiCall('/api/auth/send-code', {
      phone: trimmedPhone
    })
    if (data.phone_code_hash) {
      phoneCodeHash.value = data.phone_code_hash
    }
    stepOverride.value = 3
  } catch {
    // Manejado en apiCall
  }
}

const verifyCode = async () => {
  const trimmedCode = code.value.trim()
  if (!trimmedCode) return

  try {
    const data = await apiCall('/api/auth/sign-in', {
      phone: phoneNumber.value.trim(),
      code: trimmedCode,
      phone_code_hash: phoneCodeHash.value
    })

    if (data.state === 'WAITING_2FA') {
      stepOverride.value = 4
    } else if (data.state === 'LOGGED_IN' || data.success) {
      emit('auth-success')
    }
  } catch {
    // Manejado en apiCall
  }
}

const submit2FA = async () => {
  const trimmedPassword = password.value.trim()
  if (!trimmedPassword) return

  try {
    const data = await apiCall('/api/auth/2fa', {
      password: trimmedPassword
    })

    if (data.state === 'LOGGED_IN' || data.success) {
      emit('auth-success')
    }
  } catch {
    // Manejado en apiCall
  }
}

const goBack = () => {
  if (currentStep.value > 1) {
    stepOverride.value = currentStep.value - 1
  }
}
</script>

<template>
  <div class="auth-wrapper">
    <div class="auth-card">
      <div class="auth-header">
        <h2>{{ t('auth.title') }}</h2>
        <div class="step-indicator">
          <span :class="{ active: currentStep === 1 }">1</span>
          <span :class="{ active: currentStep === 2 }">2</span>
          <span :class="{ active: currentStep === 3 }">3</span>
          <span :class="{ active: currentStep === 4 }">4</span>
        </div>
      </div>

      <!-- Error Message -->
      <div v-if="errorMessage" class="error-banner">
        <AlertCircle :size="18" />
        <span>{{ errorMessage }}</span>
      </div>

      <!-- Step 1: API Credentials -->
      <div v-if="currentStep === 1" class="step-content">
        <div class="step-icon"><KeyRound :size="32" /></div>
        <h3>{{ t('auth.step1Title') }}</h3>
        <p>{{ t('auth.step1Desc') }}</p>

        <div class="my-telegram-banner">
          <span>{{ t('auth.credsBannerA') }}</span>
          <a
            href="#"
            class="inline-link"
            @click.prevent="openExternal('https://my.telegram.org')"
          >
            {{ t('auth.credsBannerB') }}
          </a>
        </div>

        <div class="form-group">
          <label>{{ t('auth.apiIdLabel') }}</label>
          <input
            v-model="apiId"
            type="text"
            placeholder="e.g. 1234567"
            :disabled="loading"
          />
        </div>

        <div class="form-group">
          <label>{{ t('auth.apiHashLabel') }}</label>
          <input
            v-model="apiHash"
            type="text"
            placeholder="e.g. 0123456789abcdef0123456789abcdef"
            :disabled="loading"
          />
        </div>

        <button
          class="primary-button action-btn"
          :disabled="loading"
          @click="saveCredentials"
        >
          <Loader2 v-if="loading" class="spin" :size="18" />
          <span v-else>{{ t('auth.submit') }}</span>
          <ArrowRight v-if="!loading" :size="18" />
        </button>
      </div>

      <!-- Step 2: Phone Number -->
      <div v-else-if="currentStep === 2" class="step-content">
        <div class="step-icon"><Phone :size="32" /></div>
        <h3>{{ t('auth.step2Title') }}</h3>
        <p>{{ t('auth.step2Desc') }}</p>

        <div class="form-group">
          <label>{{ t('auth.phoneLabel') }}</label>
          <input
            v-model="phoneNumber"
            type="text"
            placeholder="+1234567890"
            :disabled="loading"
            @keyup.enter="sendCode"
          />
        </div>

        <div class="btn-group">
          <button class="secondary-button" :disabled="loading" @click="goBack">
            <ArrowLeft :size="18" />
            <span>{{ t('auth.back') }}</span>
          </button>
          <button
            class="primary-button action-btn"
            :disabled="loading"
            @click="sendCode"
          >
            <Loader2 v-if="loading" class="spin" :size="18" />
            <span v-else>{{ t('auth.sendCode') }}</span>
            <ArrowRight v-if="!loading" :size="18" />
          </button>
        </div>
      </div>

      <!-- Step 3: Verification Code -->
      <div v-else-if="currentStep === 3" class="step-content">
        <div class="step-icon"><CheckCircle2 :size="32" /></div>
        <h3>{{ t('auth.step3Title') }}</h3>
        <p>{{ t('auth.step3Desc') }}</p>

        <div class="form-group">
          <label>{{ t('auth.codeLabel') }}</label>
          <input
            v-model="code"
            type="text"
            placeholder="12345"
            maxlength="5"
            :disabled="loading"
            @keyup.enter="verifyCode"
          />
        </div>

        <div class="btn-group">
          <button class="secondary-button" :disabled="loading" @click="goBack">
            <ArrowLeft :size="18" />
            <span>{{ t('auth.back') }}</span>
          </button>
          <button
            class="primary-button action-btn"
            :disabled="loading"
            @click="verifyCode"
          >
            <Loader2 v-if="loading" class="spin" :size="18" />
            <span v-else>{{ t('auth.verifyCode') }}</span>
            <ArrowRight v-if="!loading" :size="18" />
          </button>
        </div>
      </div>

      <!-- Step 4: 2FA Password -->
      <div v-else-if="currentStep === 4" class="step-content">
        <div class="step-icon"><ShieldCheck :size="32" /></div>
        <h3>{{ t('auth.step4Title') }}</h3>
        <p>{{ t('auth.step4Desc') }}</p>

        <div class="form-group">
          <label>{{ t('auth.passwordLabel') }}</label>
          <input
            v-model="password"
            type="password"
            placeholder="Password"
            :disabled="loading"
            @keyup.enter="submit2FA"
          />
        </div>

        <div class="btn-group">
          <button class="secondary-button" :disabled="loading" @click="goBack">
            <ArrowLeft :size="18" />
            <span>{{ t('auth.back') }}</span>
          </button>
          <button
            class="primary-button action-btn"
            :disabled="loading"
            @click="submit2FA"
          >
            <Loader2 v-if="loading" class="spin" :size="18" />
            <span v-else>{{ t('auth.unlock2FA') }}</span>
            <ArrowRight v-if="!loading" :size="18" />
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.auth-wrapper {
  display: grid;
  place-items: center;
  min-height: 80vh;
  padding: 20px;
}

.auth-card {
  background: var(--user-surface, #0b1a2a);
  border: 1px solid var(--user-border, #1b344b);
  border-radius: 20px;
  width: 100%;
  max-width: 480px;
  padding: 32px;
  box-shadow: 0 20px 50px rgba(0, 0, 0, 0.4);
}

.auth-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
  border-bottom: 1px solid var(--user-border, #1b344b);
  padding-bottom: 16px;
}

.auth-header h2 {
  margin: 0;
  font-size: 20px;
  color: #f3f8ff;
  font-family: 'Space Grotesk', sans-serif;
}

.step-indicator {
  display: flex;
  gap: 8px;
}

.step-indicator span {
  width: 26px;
  height: 26px;
  border-radius: 50%;
  background: var(--user-surface-light, #0e2032);
  color: var(--user-text-dim, #728ba2);
  font-size: 12px;
  font-weight: 700;
  display: grid;
  place-items: center;

  transition: all 0.2s ease;
}

.step-indicator span.active {
  background: var(--user-primary, #38a7ff);
  color: var(--user-bg-base, #07111f);
  box-shadow: 0 0 12px var(--user-glow);
}

.error-banner {
  background: rgba(255, 77, 77, 0.1);
  border: 1px solid rgba(255, 77, 77, 0.3);
  color: #ffadad;
  padding: 12px 16px;
  border-radius: 10px;
  margin-bottom: 20px;
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 13px;
}

.step-content {
  display: flex;
  flex-direction: column;
}

.step-icon {
  width: 56px;
  height: 56px;
  border-radius: 16px;
  background: var(--user-icon-bg, #11385b);
  color: var(--user-primary, #38a7ff);
  display: grid;
  place-items: center;
  margin-bottom: 16px;
}

.step-content h3 {
  margin: 0 0 6px;
  color: #f1f7ff;
  font-size: 18px;
  font-family: 'Space Grotesk', sans-serif;
}

.step-content p {
  margin: 0 0 20px;
  color: var(--user-text-dim, #728ba2);
  font-size: 13px;
  line-height: 1.5;
}

.my-telegram-banner {
  background: var(--user-surface-light, #0e2032);
  border: 1px solid var(--user-border-light, #234765);
  padding: 10px 14px;
  border-radius: 10px;
  font-size: 12px;
  margin-bottom: 18px;

  color: #dbe7f5;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 18px;
}

.form-group label {
  font-size: 12px;
  font-weight: 600;
  color: #dbe7f5;
}

.form-group input {
  background: var(--user-bg-base, #07111f);
  border: 1px solid var(--user-border-light, #234765);
  color: #dbe7f5;
  border-radius: 10px;
  padding: 12px 14px;
  outline: none;
  font-size: 14px;
}

.form-group input:focus {
  border-color: var(--user-primary, #38a7ff);
  box-shadow: 0 0 0 3px var(--user-glow);
}

.action-btn {
  width: 100%;
  justify-content: center;
  padding: 12px;
  margin-top: 8px;
}

.btn-group {
  display: flex;
  gap: 12px;
  margin-top: 8px;
}

.secondary-button {
  border: 1px solid var(--user-border-light, #234765);
  background: var(--user-surface-light, #0e2032);
  color: #dbe7f5;
  padding: 12px 18px;
  border-radius: 10px;
  font-weight: 600;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 8px;
}

.secondary-button:hover:not(:disabled) {
  background: var(--user-border, #1b344b);
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
