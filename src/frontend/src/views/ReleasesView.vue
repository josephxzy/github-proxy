<template>
  <main class="flex-1 flex items-start justify-center py-8 px-4 sm:px-6 lg:px-8 bg-gradient-to-br from-gray-100 to-gray-200 dark:from-gray-900 dark:to-gray-950 transition-colors duration-300">
    <div class="w-full max-w-[1000px] mx-auto">
      <div class="pt-6">
        <button @click="$emit('back')" class="inline-flex items-center gap-2 text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300 font-medium transition-colors mb-6">
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="m15 18-6-6 6-6"></path>
          </svg>
          {{ fromView === 'search' ? '返回搜索结果' : '返回首页' }}
        </button>

        <div v-if="loadingReleases" class="text-center py-12">
          <svg class="animate-spin h-8 w-8 text-blue-600 mx-auto" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
          <p class="mt-4 text-gray-600 dark:text-gray-400">加载Releases列表中...</p>
        </div>

        <div v-else-if="releasesError" class="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-xl p-6 text-center">
          <svg class="h-12 w-12 text-red-500 mx-auto mb-3" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <p class="text-red-700 dark:text-red-400">{{ releasesError }}</p>
        </div>

        <div v-else-if="releasesData.length === 0" class="text-center py-8">
          <p class="text-gray-600 dark:text-gray-400">该仓库没有Releases</p>
        </div>

        <div v-else class="space-y-6">
          <div v-for="release in releasesData" :key="release.id" class="bg-white/60 dark:bg-gray-800/60 backdrop-blur-sm border border-gray-200/50 dark:border-gray-700/50 rounded-xl p-4 sm:p-6">
            <div class="flex flex-col gap-3 mb-4">
              <div class="flex-1">
                <div class="flex flex-wrap items-center gap-2 mb-2">
                  <span class="px-3 py-1 bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-400 text-sm font-semibold rounded-full">
                    {{ release.tag_name }}
                  </span>
                  <span v-if="release.prerelease" class="px-3 py-1 bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-400 text-sm font-semibold rounded-full">
                    Pre-release
                  </span>
                  <span v-if="release.draft" class="px-3 py-1 bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 text-sm font-semibold rounded-full">
                    Draft
                  </span>
                  <span class="text-xs text-gray-500 dark:text-gray-400 ml-auto">共 {{ totalReleases }} 个版本，{{ totalPages }} 页</span>
                </div>
                <h3 class="text-lg sm:text-xl font-semibold text-gray-900 dark:text-white mb-2">{{ release.name || release.tag_name }}</h3>
                <p class="text-sm text-gray-500 dark:text-gray-400">
                  发布时间: {{ formatDate(release.published_at) }}
                </p>
              </div>
            </div>

            <div v-if="release.body" class="prose dark:prose-invert max-w-none mb-4 p-4 bg-gray-50/50 dark:bg-gray-900/30 rounded-lg text-sm text-gray-700 dark:text-gray-300 markdown-body" v-html="renderMarkdown(release.body)"></div>

            <div v-if="release.assets && release.assets.length > 0" class="border-t border-gray-200 dark:border-gray-700 pt-4">
              <h4 class="text-sm font-semibold text-gray-700 dark:text-gray-300 mb-3">下载资源 ({{ release.assets.length }})</h4>
              <div class="space-y-2">
                <div v-for="asset in release.assets" :key="asset.id" class="flex flex-col sm:flex-row sm:items-center justify-between p-3 bg-gray-50/50 dark:bg-gray-900/30 rounded-lg hover:bg-gray-100/70 dark:hover:bg-gray-700/50 transition-colors gap-3">
                  <div class="flex-1 min-w-0">
                    <p class="text-sm font-medium text-gray-900 dark:text-white truncate">{{ asset.name }}</p>
                    <p class="text-xs text-gray-500 dark:text-gray-400">{{ formatSize(asset.size) }} · 下载 {{ asset.download_count }} 次</p>
                  </div>
                  <div class="flex items-center gap-2 w-full sm:w-auto">
                    <button
                      @click="copyAssetUrl(asset.browser_download_url)"
                      class="flex-1 sm:flex-none px-3 sm:px-4 py-2 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 text-sm font-medium rounded-lg transition-colors"
                      :class="{ 'bg-green-50 dark:bg-green-900/20 border-green-300 dark:border-green-700 text-green-700 dark:text-green-400': copiedAssetUrl === asset.browser_download_url }"
                    >
                      <span v-if="copiedAssetUrl === asset.browser_download_url">已复制</span>
                      <span v-else>复制链接</span>
                    </button>
                    <button
                      @click="downloadAsset(asset.browser_download_url)"
                      class="flex-1 sm:flex-none px-3 sm:px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium rounded-lg transition-colors"
                    >
                      下载
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div class="flex items-center justify-center gap-1 mt-6 pt-6 border-t border-gray-200 dark:border-gray-700">
            <button
              @click="goToPage(currentPage - 1)"
              :disabled="currentPage <= 1 || loadingReleases"
              class="px-3 py-1.5 border border-gray-300 dark:border-gray-700 rounded-md text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors">
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="m15 18-6-6 6-6"></path>
              </svg>
            </button>

            <template v-for="page in visiblePages" :key="page">
              <span v-if="page === '...'" class="px-3 py-1.5 text-sm text-gray-500 dark:text-gray-400">…</span>
              <button
                v-else
                @click="goToPage(page)"
                :class="[
                  'px-3 py-1.5 border rounded-md text-sm font-medium transition-colors',
                  currentPage === page
                    ? 'bg-blue-600 text-white border-blue-600 dark:border-blue-600'
                    : 'border-gray-300 dark:border-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700'
                ]">
                {{ page }}
              </button>
            </template>

            <button
              @click="goToPage(currentPage + 1)"
              :disabled="currentPage >= totalPages || loadingReleases"
              class="px-3 py-1.5 border border-gray-300 dark:border-gray-700 rounded-md text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors">
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="m9 18 6-6-6-6"></path>
              </svg>
            </button>
          </div>
        </div>
      </div>
    </div>
  </main>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { renderMarkdown } from '../markdown.js'

const props = defineProps({
  repoUrl: String,
  selectedNode: Object,
  getNodeUrl: Function,
  fromView: {
    type: String,
    default: 'home'
  }
})

const emit = defineEmits(['back'])

const releasesData = ref([])
const loadingReleases = ref(false)
const releasesError = ref('')
const copiedAssetUrl = ref('')

const currentPage = ref(1)
const perPage = 10
const totalReleases = ref(0)
const totalPages = ref(0)

onMounted(() => {
  if (props.repoUrl) {
    fetchReleases()
  }
})

watch(() => props.repoUrl, (newUrl, oldUrl) => {
  if (newUrl && newUrl !== oldUrl) {
    currentPage.value = 1
    totalReleases.value = 0
    totalPages.value = 0
    releasesData.value = []
    fetchReleases()
  }
})

const fetchReleases = async (page = 1) => {
  if (!props.repoUrl || !props.selectedNode) return

  loadingReleases.value = true
  releasesError.value = ''

  try {
    let owner, repo
    const url = props.repoUrl.trim()

    if (url.startsWith('https://github.com/')) {
      const urlParts = url.split('/')
      owner = urlParts[3]
      repo = urlParts[4]?.replace('.git', '')
    } else {
      const parts = url.split('/')
      if (parts.length >= 2) {
        owner = parts[0]
        repo = parts[1]?.replace('.git', '')
      }
    }

    if (!owner || !repo) {
      throw new Error('无效的仓库URL')
    }

    const apiPath = `repos/${owner}/${repo}/releases?per_page=${perPage}&page=${page}`
    const proxyUrl = '/' + `https://api.github.com/${apiPath}`

    const response = await fetch(proxyUrl, { cache: 'no-store' })

    if (!response.ok) {
      if (response.status === 403) {
        throw new Error('访问被拒绝，请检查仓库是否公开')
      } else if (response.status === 404) {
        throw new Error('仓库不存在或没有Releases')
      }
      throw new Error(`请求失败: ${response.status} ${response.statusText}`)
    }

    const linkHeader = response.headers.get('Link')
    if (linkHeader) {
      const lastMatch = linkHeader.match(/page=(\d+)>;\s*rel="last"/)
      if (lastMatch) {
        totalPages.value = parseInt(lastMatch[1], 10)
      }
    }

    const data = await response.json()

    if (Array.isArray(data)) {
      releasesData.value = data
      if (totalPages.value) {
        const isLastPage = currentPage.value === totalPages.value
        if (isLastPage) {
          totalReleases.value = (totalPages.value - 1) * perPage + data.length
        } else {
          totalReleases.value = (totalPages.value - 1) * perPage + perPage
        }
      } else {
        totalPages.value = currentPage.value
        totalReleases.value = (currentPage.value - 1) * perPage + data.length
      }
    } else {
      throw new Error(data.message || '获取Releases失败')
    }
  } catch (error) {
    console.error('获取Releases失败:', error)
    releasesError.value = error.message || '获取Releases失败，请稍后重试'
  } finally {
    loadingReleases.value = false
  }
}

const visiblePages = computed(() => {
  const pages = []
  const maxVisible = 7
  const total = totalPages.value || 1

  if (total <= maxVisible) {
    for (let i = 1; i <= total; i++) pages.push(i)
    return pages
  }

  pages.push(1)
  if (currentPage.value > 4) pages.push('...')

  const start = Math.max(2, currentPage.value - 2)
  const end = Math.min(total - 1, currentPage.value + 2)

  for (let i = start; i <= end; i++) pages.push(i)

  if (currentPage.value < total - 3) pages.push('...')
  pages.push(total)

  return pages
})

const goToPage = (page) => {
  if (page < 1 || page > totalPages.value || loadingReleases.value) return
  currentPage.value = page
  fetchReleases(page)
}

const copyAssetUrl = (url) => {
  if (!props.selectedNode) return
  const proxyUrl = props.getNodeUrl(props.selectedNode) + '/' + url
  navigator.clipboard.writeText(proxyUrl).then(() => {
    copiedAssetUrl.value = url
    setTimeout(() => {
      copiedAssetUrl.value = ''
    }, 2000)
  })
}

const downloadAsset = (url) => {
  if (!props.selectedNode) return
  const proxyUrl = props.getNodeUrl(props.selectedNode) + '/' + url
  window.open(proxyUrl, '_blank')
}

const formatDate = (dateString) => {
  if (!dateString) return '未知'
  const date = new Date(dateString)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

const formatSize = (bytes) => {
  if (!bytes) return '未知大小'
  const units = ['B', 'KB', 'MB', 'GB']
  let unitIndex = 0
  let size = bytes

  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024
    unitIndex++
  }

  return `${size.toFixed(2)} ${units[unitIndex]}`
}
</script>
