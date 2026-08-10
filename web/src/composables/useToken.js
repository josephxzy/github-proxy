import { ref, watch } from 'vue'

const STORAGE_KEY = 'github_token'
const token = ref(localStorage.getItem(STORAGE_KEY) || '')
const tokenInvalid = ref(false)

watch(token, (newVal) => {
  if (newVal) {
    localStorage.setItem(STORAGE_KEY, newVal)
  } else {
    localStorage.removeItem(STORAGE_KEY)
  }
  tokenInvalid.value = false
})

export function useToken() {
  return {
    token,
    tokenInvalid,
    clearToken() {
      token.value = ''
    },
    markTokenInvalid() {
      tokenInvalid.value = true
    }
  }
}
