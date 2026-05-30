<template>
  <main class="flex-1 py-8 px-4 sm:px-6 lg:px-8 bg-gray-50 dark:bg-gray-950 transition-colors duration-300">
    <div class="w-full max-w-[800px] mx-auto">
      <div class="pt-10">
        <div class="text-center w-full mb-12">
          <h1 class="font-bold mb-4 text-5xl text-gray-900 dark:text-white transition-colors duration-300">联系与支持</h1>
          <p class="text-gray-600 dark:text-gray-400 text-base">如有任何问题、建议或反馈，欢迎通过以下方式联系我们。</p>
        </div>



        <div class="space-y-6">
          <div class="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg p-8">
            <h2 class="text-2xl font-semibold text-gray-900 dark:text-white mb-6">服务信息</h2>

            <div v-if="loading" class="text-center py-8">
              <div class="inline-block animate-spin rounded-full h-8 w-8 border-4 border-blue-600 border-t-transparent">
              </div>
              <p class="mt-3 text-sm text-gray-500 dark:text-gray-400">加载中...</p>
            </div>

            <div v-else-if="error"
              class="p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
              <div class="flex items-center gap-2">
                <svg class="w-5 h-5 text-red-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
                </svg>
                <span class="text-sm text-red-600 dark:text-red-400">获取状态失败</span>
              </div>
            </div>

            <div v-else class="space-y-4">
              <div class="p-4 bg-gray-50 dark:bg-gray-900/50 rounded-lg border border-gray-200 dark:border-gray-700">
                <div class="flex items-center justify-between">
                  <div class="flex items-center gap-3">
                    <span class="w-2.5 h-2.5 rounded-full bg-green-500 animate-pulse"></span>
                    <span class="text-lg font-semibold text-gray-900 dark:text-white">{{ serviceName }}</span>
                  </div>
                  <div class="text-sm text-gray-500 dark:text-gray-400">
                    构建: {{ buildTime }}
                  </div>
                </div>
              </div>

              <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div class="p-4 bg-gray-50 dark:bg-gray-900/50 rounded-lg border border-gray-200 dark:border-gray-700">
                  <div class="flex items-center gap-2 mb-2">
                    <svg class="w-5 h-5 text-gray-500 dark:text-gray-400" fill="none" viewBox="0 0 24 24"
                      stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                        d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z">
                      </path>
                    </svg>
                    <span class="text-sm font-medium text-gray-700 dark:text-gray-300">启动时间</span>
                  </div>
                  <div class="text-sm text-gray-900 dark:text-white">{{ startTimeFormatted }}</div>
                </div>

                <div class="p-4 bg-gray-50 dark:bg-gray-900/50 rounded-lg border border-gray-200 dark:border-gray-700">
                  <div class="flex items-center gap-2 mb-2">
                    <svg class="w-5 h-5 text-gray-500 dark:text-gray-400" fill="none" viewBox="0 0 24 24"
                      stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                        d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"></path>
                    </svg>
                    <span class="text-sm font-medium text-gray-700 dark:text-gray-300">运行时间</span>
                  </div>
                  <div class="text-sm text-gray-900 dark:text-white">{{ uptime }}</div>
                </div>
              </div>
            </div>
          </div>

          <div class="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg p-8">
            <h2 class="text-2xl font-semibold text-gray-900 dark:text-white mb-6">联系方式</h2>

            <div class="space-y-4">

              <div class="flex items-start gap-4 p-4 bg-gray-50 dark:bg-gray-900/50 rounded-lg">
                <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none"
                  stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
                  class="w-6 h-6 text-blue-600 flex-shrink-0 mt-0.5">
                  <rect width="20" height="16" x="2" y="4" rx="2"></rect>
                  <path d="m22 7-8.97 5.7a1.94 1.94 0 0 1-2.06 0L2 7"></path>
                </svg>
                <div class="w-full">
                  <h3 class="font-medium text-gray-900 dark:text-white mb-1">电子邮箱</h3>
                  <p class="text-sm text-gray-600 dark:text-gray-400 mb-3">如有任何问题或建议，欢迎发送邮件联系我们。</p>
                  <div class="flex gap-2">
                    <button @click="copyEmail"
                      class="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-medium transition-colors">
                      复制邮箱
                    </button>
                    <a :href="`mailto:${email}`"
                      class="px-4 py-2 bg-gray-200 dark:bg-gray-700 hover:bg-gray-300 dark:hover:bg-gray-600 text-gray-900 dark:text-white rounded-lg text-sm font-medium transition-colors">
                      发送邮件
                    </a>
                  </div>
                </div>
              </div>

              <div class="flex items-start gap-4 p-4 bg-gray-50 dark:bg-gray-900/50 rounded-lg">
                <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none"
                  stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
                  class="w-6 h-6 text-blue-600 flex-shrink-0 mt-0.5">
                  <path d="M7.9 20A9 9 0 1 0 4 16.1L2 22Z"></path>
                </svg>
                <div>
                  <h3 class="font-medium text-gray-900 dark:text-white mb-1">问题反馈</h3>
                  <p class="text-sm text-gray-600 dark:text-gray-400 mb-2">如果您有任何使用上的问题或改进建议，欢迎提交 Issue。</p>
                  <a href="https://github.com/AlistairBlake/github-proxy/issues" target="_blank" rel="noopener noreferrer"
                    class="text-sm text-blue-600 dark:text-blue-400 hover:underline">
                    Github Issues &rarr;
                  </a>
                </div>
              </div>


            </div>
          </div>

        </div>


      </div>
    </div>
  </main>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'

const email = ref('AlistairBlake@example.com')
const serverInfo = ref({})
const loading = ref(true)
const error = ref(false)

const fetchServerInfo = async () => {
  try {
    loading.value = true
    error.value = false
    const response = await fetch('/ready')
    const data = await response.json()
    serverInfo.value = data
  } catch (err) {
    error.value = true
  } finally {
    loading.value = false
  }
}

const serviceName = computed(() => {
  const service = serverInfo.value.service ?? ''
  const version = serverInfo.value.version ?? ''
  return `${service} (${version})`
})

const buildTime = computed(() => {
  if (!serverInfo.value.build_time || serverInfo.value.build_time === 'unknown') return '未知'
  return serverInfo.value.build_time
})

const startTimeFormatted = computed(() => {
  if (!serverInfo.value.start_time_unix) return '未知'
  const date = new Date(serverInfo.value.start_time_unix * 1000)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false
  })
})

const uptime = computed(() => {
  if (serverInfo.value.uptime) {
    return serverInfo.value.uptime
  }
  return '未知'
})

const copyEmail = async () => {
  try {
    await navigator.clipboard.writeText(email.value)
    alert('邮箱地址已复制到剪贴板')
  } catch (err) {
    const textArea = document.createElement('textarea')
    textArea.value = email.value
    textArea.style.position = 'fixed'
    textArea.style.left = '-9999px'
    document.body.appendChild(textArea)
    textArea.select()
    document.execCommand('copy')
    document.body.removeChild(textArea)
    alert('邮箱地址已复制到剪贴板')
  }
}

onMounted(() => {
  fetchServerInfo()
})
</script>
