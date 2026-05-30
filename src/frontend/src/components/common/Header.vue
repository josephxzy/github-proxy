<template>
  <header class="bg-white dark:bg-gray-900 transition-colors duration-300">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
      <div class="flex items-center justify-between h-16">
        <div class="flex items-center space-x-2">
          <div
            class="w-10 h-10 rounded-lg flex items-center justify-center transition-colors duration-300 bg-white dark:bg-gray-800 shadow-sm">
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"
              class="w-8 h-8 transition-colors duration-300 fill-gray-900 dark:fill-white" aria-hidden="true">
              <path
                d="M7.75 11c-.69 0-1.25.56-1.25 1.25v1.5a1.25 1.25 0 1 0 2.5 0v-1.5C9 11.56 8.44 11 7.75 11zm1.27 4.5a.469.469 0 0 1 .48-.5h5a.47.47 0 0 1 .48.5c-.116 1.316-.759 2.5-2.98 2.5s-2.864-1.184-2.98-2.5zm7.23-4.5c-.69 0-1.25.56-1.25 1.25v1.5a1.25 1.25 0 1 0 2.5 0v-1.5c0-.69-.56-1.25-1.25-1.25z">
              </path>
              <path fill-rule="evenodd"
                d="M21.255 3.82a1.725 1.725 0 0 0-2.141-1.195c-.557.16-1.406.44-2.264.866c-.78.386-1.647.93-2.293 1.677A18.442 18.442 0 0 0 12 5c-.93 0-1.784.059-2.569.17c-.645-.74-1.505-1.28-2.28-1.664a13.876 13.876 0 0 0-2.265-.866a1.725 1.725 0 0 0-2.141 1.196a23.645 23.645 0 0 0-.69 3.292c-.125.97-.191 2.07-.066 3.112C1.254 11.882 1 13.734 1 15.527C1 19.915 3.13 23 12 23c8.87 0 11-3.053 11-7.473c0-1.794-.255-3.647-.99-5.29c.127-1.046.06-2.15-.066-3.125a23.652 23.652 0 0 0-.689-3.292zM20.5 14c.5 3.5-1.5 6.5-8.5 6.5s-9-3-8.5-6.5c.583-4 3-6 8.5-6s7.928 2 8.5 6z">
              </path>
            </svg>
          </div>
          <a href="/" class="text-xl font-semibold hover:opacity-80 transition-colors duration-300 text-gray-900 dark:text-white">Github Proxy</a>
        </div>
        <div class="flex items-center space-x-8">
          <button v-if="isContact" @click="$emit('goHome')"
            class="w-6 h-6 flex items-center justify-center cursor-pointer hover:opacity-80 transition-colors duration-300"
            title="返回首页">
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none"
              stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
              class="text-gray-600 dark:text-gray-400 transition-colors duration-300">
              <path d="m3 9 9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"></path>
              <polyline points="9,22 9,12 15,12 15,22"></polyline>
            </svg>
          </button>
          <button v-else @click="$emit('goContact')"
            class="w-6 h-6 flex items-center justify-center cursor-pointer hover:opacity-80 transition-colors duration-300"
            title="反馈及建议">
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none"
              stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
              class="text-gray-600 dark:text-gray-400 transition-colors duration-300">
              <path d="M7.9 20A9 9 0 1 0 4 16.1L2 22Z"></path>
            </svg>
          </button>
          <button @click="$emit('toggleDark')"
            class="w-6 h-6 flex items-center justify-center cursor-pointer hover:opacity-80 transition-colors duration-300"
            :title="isDark ? '切换到亮色模式' : '切换到暗黑模式'">
            <svg v-if="isDark" xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24"
              fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
              class="text-yellow-400 transition-colors duration-300">
              <circle cx="12" cy="12" r="4"></circle>
              <path d="M12 2v2"></path>
              <path d="M12 20v2"></path>
              <path d="m4.93 4.93 1.41 1.41"></path>
              <path d="m17.66 17.66 1.41 1.41"></path>
              <path d="M2 12h2"></path>
              <path d="M20 12h2"></path>
              <path d="m6.34 17.66-1.41 1.41"></path>
              <path d="m19.07 4.93-1.41 1.41"></path>
            </svg>
            <svg v-else xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none"
              stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
              class="text-gray-600 transition-colors duration-300">
              <path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9Z"></path>
            </svg>
          </button>
        </div>
      </div>
    </div>
  </header>
</template>

<script setup>
defineProps({
  isDark: {
    type: Boolean,
    required: true
  },
  isContact: {
    type: Boolean,
    default: false
  }
})

defineEmits(['toggleDark', 'goContact', 'goHome'])
</script>
