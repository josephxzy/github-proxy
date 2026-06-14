import { ref, watch } from 'vue'

const STORAGE_KEY = 'github_token'
const token = ref(localStorage.getItem(STORAGE_KEY) || '')

watch(token, (newVal) => {
  if (newVal) {
    localStorage.setItem(STORAGE_KEY, newVal)
  } else {
    localStorage.removeItem(STORAGE_KEY)
  }
})

export function useToken() {
  return {
    token,
    clearToken() {
      token.value = ''
    }
  }
}
