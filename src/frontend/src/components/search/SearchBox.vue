<template>
  <div class="w-full relative z-[10001] mb-8">
    <div class="flex flex-col sm:flex-row gap-3 sm:items-start">
      <div class="flex-1 relative">
        <div class="absolute left-4 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500 pointer-events-none transition-colors">
          <svg v-if="isUrl" xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none"
            stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"></path>
            <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"></path>
          </svg>
          <svg v-else xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none"
            stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="11" cy="11" r="8"></circle>
            <path d="m21 21-4.3-4.3"></path>
          </svg>
        </div>
        <input
          type="text"
          :value="modelValue"
          @input="onInput($event)"
          @compositionstart="isComposing = true"
          @compositionend="onCompositionEnd($event)"
          @keydown="onKeyDown($event)"
          @keyup.enter="$emit('submit')"
          :placeholder="placeholder"
          class="w-full pl-11 pr-4 py-3 bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 rounded-xl shadow-sm ring-1 ring-gray-200 dark:ring-gray-700 focus:outline-none focus:ring-2 focus:ring-blue-500 transition-shadow duration-300 hover:ring-gray-300 dark:hover:ring-gray-600 placeholder:text-gray-400 dark:placeholder:text-gray-500 h-[48px]" />
      </div>
      <button
        @click="$emit('submit')"
        :disabled="!isValid"
        :class="[
          'px-6 py-3 rounded-xl font-medium transition-all whitespace-nowrap shrink-0 h-[48px]',
          isValid
            ? 'text-white bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 shadow-md shadow-blue-600/25 hover:shadow-lg hover:shadow-blue-600/30 active:scale-[0.98]'
            : 'text-gray-400 dark:text-gray-500 bg-gray-100 dark:bg-gray-800 ring-1 ring-gray-200 dark:ring-gray-700 cursor-not-allowed'
        ]">
        {{ buttonText }}
      </button>
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'

const props = defineProps({
  modelValue: {
    type: String,
    default: ''
  },
  isReleasesMode: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['update:modelValue', 'submit'])

const isComposing = ref(false)

const replaceColons = (val) => val.replace(/：/g, ':')

const onInput = (e) => {
  if (!isComposing.value) {
    emit('update:modelValue', replaceColons(e.target.value))
  }
}

const onCompositionEnd = (e) => {
  isComposing.value = false
  const newValue = replaceColons(e.target.value)
  emit('update:modelValue', newValue)
}

const onKeyDown = (e) => {
  if (e.key === 'Enter' && isComposing.value) {
    e.preventDefault()
  }
}

const isUrl = computed(() => {
  const v = props.modelValue.trim()
  return v.startsWith('http://') || v.startsWith('https://')
})

const inputType = computed(() => {
  const url = props.modelValue.trim()

  if (!url) return 'empty'

  if (props.isReleasesMode) return 'releases'

  if (url.startsWith('http://') || url.startsWith('https://')) {
    const repoPattern = /^https?:\/\/github\.com\/[^/]+\/[^/]+(?:\.git)?$/
    if (repoPattern.test(url)) return 'repo'
    return 'download'
  }

  return 'search'
})

const buttonText = computed(() => {
  switch (inputType.value) {
    case 'releases':
      return '查看'
    case 'repo':
      return '下载 ZIP'
    case 'download':
      return '下载'
    case 'search':
      return '搜索'
    default:
      return props.isReleasesMode ? '查看' : '搜索'
  }
})

const placeholder = computed(() => {
  if (props.isReleasesMode) {
    return '输入 Github 仓库链接 (例如: https://github.com/owner/repo)'
  }
  return '搜索关键词、user/repo、user:name，或粘贴 GitHub 链接'
})

const isValid = computed(() => {
  if (!props.modelValue.trim()) return false

  if (props.isReleasesMode) {
    const url = props.modelValue.trim()
    if (url.startsWith('https://github.com/')) {
      return url.split('/').length >= 5
    }
    const shortPattern = /^[a-zA-Z0-9_.-]+\/[a-zA-Z0-9_.-]+$/
    return shortPattern.test(url)
  }

  const url = props.modelValue.trim()

  if (url.startsWith('http://') || url.startsWith('https://')) {
    return url.startsWith('https://github.com') ||
           url.startsWith('https://raw.githubusercontent.com') ||
           url.startsWith('https://api.github.com')
  }

  return url.length > 0
})
</script>
