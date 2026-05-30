<template>
  <div class="w-full relative z-[10001] mb-8">
    <div class="flex flex-col sm:flex-row gap-3 sm:items-start">
      <div class="flex-1 relative">
        <input
          type="text"
          :value="modelValue"
          @input="onInput($event)"
          @compositionstart="isComposing = true"
          @compositionend="onCompositionEnd($event)"
          @keydown="onKeyDown($event)"
          @keyup.enter="$emit('submit')"
          :placeholder="placeholder"
          class="w-full px-4 py-3 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 border rounded-lg focus:outline-none focus:ring-2 transition-colors duration-300 placeholder:text-gray-500 dark:placeholder:text-gray-400 h-[48px] border-gray-300 dark:border-gray-700 focus:ring-blue-500" />
      </div>
      <button
        @click="$emit('submit')"
        :disabled="!isValid"
        :class="[
          'px-6 py-3 text-white rounded-lg font-medium transition-colors whitespace-nowrap shrink-0 h-[48px]',
          isValid
            ? 'bg-blue-600 hover:bg-blue-700'
            : 'bg-gray-400 cursor-not-allowed opacity-50'
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
