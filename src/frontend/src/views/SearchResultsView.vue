<template>
  <main class="flex-1 flex items-start justify-center py-8 px-4 sm:px-6 lg:px-8 bg-gray-50 dark:bg-gray-950 transition-colors duration-300">
    <div class="w-full max-w-[1000px] mx-auto">
      <div class="pt-6">
        <button
          @click="$emit('back')"
          title="返回首页"
          class="fixed right-4 sm:right-6 top-1/2 -translate-y-1/2 z-[9000] w-11 h-11 flex items-center justify-center rounded-full bg-white dark:bg-gray-900 ring-1 ring-gray-200 dark:ring-gray-700 shadow-lg hover:shadow-xl hover:ring-gray-300 dark:hover:ring-gray-600 text-gray-700 dark:text-gray-300 hover:text-blue-600 dark:hover:text-blue-400 active:scale-95 transition-all"
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M19 12H5"></path>
            <path d="m12 19-7-7 7-7"></path>
          </svg>
        </button>

        <div v-if="loadingSearch" class="text-center py-12">
          <svg class="animate-spin h-8 w-8 text-blue-600 mx-auto" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
          <p class="mt-4 text-gray-600 dark:text-gray-400">搜索仓库中...</p>
        </div>

        <div v-else-if="searchError" class="bg-red-50 dark:bg-red-900/20 ring-1 ring-red-200 dark:ring-red-800 rounded-2xl p-6 text-center animate-fade-in">
          <svg class="h-12 w-12 text-red-500 mx-auto mb-3" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <p class="text-red-700 dark:text-red-400">{{ searchError }}</p>
        </div>

        <div v-else-if="searchResults.length === 0" class="text-center py-8">
          <p class="text-gray-600 dark:text-gray-400">未找到匹配的仓库</p>
        </div>

        <div v-else class="space-y-4">
          <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2 mb-4">
            <h2 class="text-lg sm:text-xl font-bold text-gray-900 dark:text-white">仓库列表</h2>
            <span class="text-sm text-gray-500 dark:text-gray-400">
              显示 {{ searchResults.length }} / {{ totalResults }} 个仓库
              <span v-if="totalPages > 1" class="ml-1">(第 {{ currentPage }}/{{ totalPages }} 页)</span>
            </span>
          </div>

          <div v-for="repo in searchResults" :key="repo.id" class="bg-white dark:bg-gray-900 ring-1 ring-gray-200 dark:ring-gray-800 rounded-2xl p-4 sm:p-5 shadow-sm hover:shadow-lg hover:ring-gray-300 dark:hover:ring-gray-700 hover:-translate-y-0.5 transition-all animate-fade-in-up">
            <div class="flex items-center gap-3 mb-2">
              <div class="w-8 h-8 rounded-lg bg-gray-100 dark:bg-gray-800 ring-1 ring-gray-200/70 dark:ring-gray-700 flex items-center justify-center flex-shrink-0">
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="text-gray-700 dark:text-gray-300">
                  <path d="M9 19c-5 1.5-5-2.5-7-3m14 6v-3.87a3.37 3.37 0 0 0-.94-2.61c3.14-.35 6.44-1.54 6.44-7A5.44 5.44 0 0 0 20 4.77 5.07 5.07 0 0 0 19.91 1S18.73.65 16 2.48a13.38 13.38 0 0 0-7 0C6.27.65 5.09 1 5.09 1A5.07 5.07 0 0 0 5 4.77a5.44 5.44 0 0 0-1.5 3.78c0 5.42 3.3 6.61 6.44 7A3.37 3.37 0 0 0 9 18.13V22"></path>
                </svg>
              </div>
              <a :href="repo.html_url" target="_blank" rel="noopener noreferrer" class="text-lg sm:text-xl font-semibold text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300 hover:underline underline-offset-4 truncate transition-colors">
                {{ repo.full_name }}
              </a>
            </div>
            <p class="text-sm text-gray-600 dark:text-gray-400 mb-2.5 line-clamp-1">{{ repo.description || '暂无描述' }}</p>
            <div class="flex flex-wrap items-center gap-x-2 sm:gap-x-4 gap-y-2">
              <div class="flex flex-wrap items-center gap-x-2 sm:gap-x-4 gap-y-1 text-xs text-gray-500 dark:text-gray-400">
                <span v-if="repo.language" class="flex items-center gap-1.5">
                  <span class="w-3 h-3 rounded-full flex-shrink-0 ring-1 ring-black/5 dark:ring-white/10" :style="{ backgroundColor: languageColor(repo.language) }"></span>
                  {{ repo.language }}
                </span>
                <span class="flex items-center gap-1">
                  <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="currentColor" stroke="currentColor" stroke-width="2" class="text-amber-400">
                    <polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 12 17.77 7 14.14 2 9.27 8.91 8.26 12 2"></polygon>
                  </svg>
                  {{ formatStarCount(repo.stargazers_count) }}
                </span>
                <span class="flex items-center gap-1">
                  <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="12" cy="18" r="3"></circle>
                    <circle cx="6" cy="6" r="3"></circle>
                    <circle cx="18" cy="6" r="3"></circle>
                    <line x1="8.7" y1="14.7" x2="15.3" y2="9.3"></line>
                    <line x1="15.3" y1="14.7" x2="8.7" y2="9.3"></line>
                  </svg>
                  {{ repo.forks_count }}
                </span>
                <span>最后提交 {{ formatDate(repo.pushed_at) }}</span>
              </div>
              <div class="flex items-center gap-1.5 sm:gap-2 ml-auto">
                <button
                  @click="downloadRepoZip(repo)"
                  class="inline-flex items-center justify-center gap-1 px-2.5 sm:px-3 py-1 sm:py-1.5 bg-blue-600 hover:bg-blue-700 text-white text-xs font-medium rounded-lg active:scale-[0.98] transition-all whitespace-nowrap"
                >
                  <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
                    <polyline points="7 10 12 15 17 10"></polyline>
                    <line x1="12" y1="15" x2="12" y2="3"></line>
                  </svg>
                  ZIP
                </button>
                <button
                  @click="viewReleases(repo)"
                  class="px-2.5 sm:px-3 py-1 sm:py-1.5 ring-1 ring-gray-200 dark:ring-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-800 hover:ring-gray-300 dark:hover:ring-gray-600 text-xs font-medium rounded-lg transition-all whitespace-nowrap"
                >
                  Releases
                </button>
                <button
                  @click="showReadme(repo)"
                  class="px-2.5 sm:px-3 py-1 sm:py-1.5 ring-1 ring-gray-200 dark:ring-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-800 hover:ring-gray-300 dark:hover:ring-gray-600 text-xs font-medium rounded-lg transition-all whitespace-nowrap"
                >
                  ReadMe
                </button>
              </div>
            </div>
          </div>

          <!-- 分页控制 - 统一样式 -->
          <div v-if="totalPages > 1" class="flex items-center justify-center gap-1.5 mt-6 pt-6 border-t border-gray-200/70 dark:border-gray-800">
            <button
              @click="prevPage"
              :disabled="currentPage === 1 || loadingSearch"
              class="p-2 ring-1 ring-gray-200 dark:ring-gray-700 rounded-lg text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-900 hover:ring-gray-300 dark:hover:ring-gray-600 disabled:opacity-40 disabled:cursor-not-allowed transition-all"
            >
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="m15 18-6-6 6-6"></path>
              </svg>
            </button>

            <template v-for="page in visiblePages" :key="page">
              <span v-if="page === '...'" class="px-3 py-1.5 text-sm text-gray-400 dark:text-gray-500">…</span>
              <button
                v-else
                @click="goToPage(page)"
                :class="[
                  'min-w-[36px] px-3 py-1.5 rounded-lg text-sm font-medium transition-all',
                  currentPage === page
                    ? 'bg-blue-600 text-white'
                    : 'ring-1 ring-gray-200 dark:ring-gray-700 bg-white dark:bg-gray-900 text-gray-700 dark:text-gray-300 hover:ring-gray-300 dark:hover:ring-gray-600'
                ]"
              >
                {{ page }}
              </button>
            </template>

            <button
              @click="nextPage"
              :disabled="currentPage === totalPages || loadingSearch"
              class="p-2 ring-1 ring-gray-200 dark:ring-gray-700 rounded-lg text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-900 hover:ring-gray-300 dark:hover:ring-gray-600 disabled:opacity-40 disabled:cursor-not-allowed transition-all"
            >
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="m9 18 6-6-6-6"></path>
              </svg>
            </button>
          </div>
        </div>
      </div>
    </div>
  </main>

  <Teleport to="body">
    <div v-if="showReadmeModal" class="fixed inset-0 z-[10002] flex items-center justify-center bg-black/50 backdrop-blur-sm animate-fade-in" @click.self="closeReadmeModal">
      <div class="bg-white dark:bg-gray-800 rounded-2xl shadow-2xl ring-1 ring-gray-900/5 dark:ring-white/10 w-full max-w-4xl mx-2 sm:mx-4 max-h-[85vh] flex flex-col relative animate-scale-in">
        <button @click="closeReadmeModal" class="absolute top-3 right-3 p-1.5 rounded-lg text-gray-400 hover:text-gray-600 hover:bg-gray-100 dark:hover:text-gray-300 dark:hover:bg-gray-700 transition-colors z-10">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>

        <div class="px-4 sm:px-6 pt-5 pb-3 border-b border-gray-100 dark:border-gray-700 flex items-center gap-3">
          <div class="w-9 h-9 rounded-xl bg-blue-50 dark:bg-blue-500/10 ring-1 ring-blue-100 dark:ring-blue-500/20 flex items-center justify-center flex-shrink-0">
            <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="text-blue-600 dark:text-blue-400">
              <path d="M4 19.5v-15A2.5 2.5 0 0 1 6.5 2H20v20H6.5a2.5 2.5 0 0 1 0-5H20"></path>
            </svg>
          </div>
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white pr-8 truncate">{{ readmeRepoName }}</h3>
        </div>

        <div v-if="readmeLoading" class="flex items-center justify-center py-16">
          <svg class="animate-spin h-8 w-8 text-blue-600 mx-auto" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
        </div>

        <div v-else-if="readmeError" class="p-6 text-center">
          <p class="text-red-500 dark:text-red-400">{{ readmeError }}</p>
        </div>

        <div v-else class="overflow-y-auto overflow-x-hidden p-4 sm:p-6 markdown-body text-gray-900 dark:text-gray-100" v-html="readmeContent"></div>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { renderMarkdown } from '../markdown.js'
import { useToken } from '../composables/useToken'

const { markTokenInvalid } = useToken()

const props = defineProps({
  searchQuery: String,
  proxyHost: String,
  token: {
    type: String,
    default: ''
  }
})

const emit = defineEmits(['back', 'view-releases'])

const searchResults = ref([])
const loadingSearch = ref(false)
const searchError = ref('')
const totalResults = ref(0)
const currentPage = ref(1)
const perPage = 30

const showReadmeModal = ref(false)
const readmeLoading = ref(false)
const readmeError = ref('')
const readmeContent = ref('')
const readmeRepoName = ref('')

// 计算总页数
const totalPages = computed(() => {
  if (totalResults.value === 0) return 0
  return Math.ceil(totalResults.value / perPage)
})

// 可见页码（带省略号的完整分页）
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

onMounted(() => {
  if (props.searchQuery) {
    searchRepositories()
  }
})

// 构建查询字符串 - 支持 GitHub 完整搜索语法
const buildQueryString = (query) => {
  let baseQuery = query.trim()

  if (!baseQuery) return ''

  // 1. 检查是否已经是完整的 GitHub 搜索语法（包含限定符）
  // 支持的限定符：user:, org:, repo:, topic:, language:, stars:, forks:, created:, pushed:, desc:, description:, readme:
  const hasAdvancedSyntax = /(?:user|org|repo|topic|language|stars|forks|created|pushed|desc|description|readme):/.test(baseQuery)

  // 如果已经包含高级语法，处理 repo: 的特殊情况
  if (hasAdvancedSyntax) {
    baseQuery = baseQuery.replace(/repo:([^/\s]+)(?=\s|$)/g, (_, repoName) => {
      return `${repoName} in:name`
    })
    return baseQuery
  }

  // 2. 处理 user/repo 格式（例如：facebook/react）
  if (baseQuery.includes('/') && !baseQuery.startsWith('http')) {
    const parts = baseQuery.split('/').filter(p => p.trim())
    if (parts.length === 2) {
      // 转换为：user:facebook react
      baseQuery = `user:${parts[0]} ${parts[1]}`
      return baseQuery
    }
  }

  // 3. 其他情况：作为全局关键词搜索（保持原样）
  // 用户输入 "facebook" 或 "facebook web" 都直接作为全局搜索
  // GitHub API 会在所有字段中搜索这些关键词

  return baseQuery
}

const searchRepositories = async (page = 1) => {
  if (!props.searchQuery.trim()) return

  loadingSearch.value = true
  searchError.value = ''
  if (page === 1) {
    searchResults.value = []
  }

  try {
    const query = buildQueryString(props.searchQuery)

    if (!query) {
      throw new Error('搜索关键词不能为空')
    }

    // 构建API参数
    let apiPath = `search/repositories?q=${encodeURIComponent(query)}&per_page=${perPage}&page=${page}`

    const proxyUrl = '/' + `https://api.github.com/${apiPath}`
    const fetchOptions = { cache: 'no-store' }
    if (props.token) {
      fetchOptions.headers = { 'X-GitHub-Token': props.token }
    }
    const response = await fetch(proxyUrl, fetchOptions)

    if (response.headers.get('X-Token-Status') === 'invalid') {
      markTokenInvalid()
    }

    if (!response.ok) {
      let errorMsg = `搜索失败: ${response.status} ${response.statusText}`

      if (response.status === 422) {
        const errorData = await response.json().catch(() => ({}))
        if (errorData.message) {
          errorMsg = `搜索参数错误: ${errorData.message}`
        } else {
          errorMsg = '搜索参数错误，请检查搜索关键词是否有效'
        }
      } else if (response.status === 403) {
        errorMsg = 'API 请求频率超限，请稍后重试'
      } else if (response.status === 429) {
        errorMsg = '请求过于频繁，请稍后再试'
      }

      throw new Error(errorMsg)
    }

    const data = await response.json()

    if (data.items && Array.isArray(data.items)) {
      searchResults.value = data.items.map(item => ({
        id: item.id,
        full_name: item.full_name,
        description: item.description,
        html_url: item.html_url,
        stargazers_count: item.stargazers_count,
        forks_count: item.forks_count,
        language: item.language,
        pushed_at: item.pushed_at || item.updated_at
      }))

      totalResults.value = data.total_count || 0
      currentPage.value = page

      if (searchResults.value.length === 0) {
        searchError.value = '未找到匹配的仓库'
      }
    } else {
      throw new Error(data.message || '搜索失败')
    }
  } catch (error) {
    console.error('搜索仓库失败:', error)
    searchError.value = error.message || '搜索失败，请稍后重试'
  } finally {
    loadingSearch.value = false
  }
}

// 跳转到指定页
const goToPage = (page) => {
  if (page < 1 || page > totalPages.value || loadingSearch.value) return
  currentPage.value = page
  searchRepositories(page)
}

// 上一页
const prevPage = () => {
  goToPage(currentPage.value - 1)
}

// 下一页
const nextPage = () => {
  goToPage(currentPage.value + 1)
}

const downloadRepoZip = async (repo) => {
  if (!repo.html_url) return

  const parts = repo.html_url.split('/').filter(p => p)
  if (parts.length < 4) return

  const owner = parts[2]
  const repoName = parts[3]

  if (!owner || !repoName) return

  let branch = 'main'
  try {
    const resp = await fetch(`/api/repo/${owner}/${repoName}/branch`)
    if (resp.ok) {
      const data = await resp.json()
      branch = data.branch || 'main'
    }
  } catch (e) {
    console.error('Failed to fetch default branch, using main:', e)
  }

  const zipUrl = `https://github.com/${owner}/${repoName}/archive/refs/heads/${branch}.zip`
  let proxyUrl = props.proxyHost + '/' + zipUrl
  if (props.token) {
    proxyUrl += '?token=' + props.token
  }
  window.open(proxyUrl, '_blank')
}

const viewReleases = (repo) => {
  emit('view-releases', repo.html_url)
}

const parseRepoFromUrl = (url) => {
  const parts = url.split('/').filter(p => p)
  if (parts.length < 4) return null
  return { owner: parts[2], repo: parts[3].replace('.git', '') }
}

const replaceGithubUrls = (html) => {
  return html.replace(/(href|src)="(https?:\/\/(?:github\.com|raw\.githubusercontent\.com|user-images\.githubusercontent\.com)\/[^"]*)"/g,
    (_, attr, url) => `${attr}="${props.proxyHost}/${url}"`)
}

const replaceRelativeUrls = (html, baseUrl) => {
  return html.replace(/(href|src)="(?!https?:\/\/|\/|#|mailto:|data:)([^"]*)"/g,
    (_, attr, path) => {
      let resolved = baseUrl
      let currentPath = path
      while (currentPath.startsWith('../')) {
        currentPath = currentPath.substring(3)
        resolved = resolved.substring(0, resolved.lastIndexOf('/', resolved.length - 2) + 1)
      }
      if (currentPath.startsWith('./')) {
        currentPath = currentPath.substring(2)
      }
      return `${attr}="${resolved}${currentPath}"`
    })
}

const decodeBase64 = (base64) => {
  const binary = atob(base64.replace(/\n/g, ''))
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i)
  }
  return new TextDecoder().decode(bytes)
}

const showReadme = async (repo) => {
  const info = parseRepoFromUrl(repo.html_url)
  if (!info) return

  readmeRepoName.value = repo.full_name
  showReadmeModal.value = true
  readmeLoading.value = true
  readmeError.value = ''
  readmeContent.value = ''

  try {
    const apiPath = `repos/${info.owner}/${info.repo}/readme`
    const proxyUrl = '/' + `https://api.github.com/${apiPath}`
    const fetchOptions = { cache: 'no-store' }
    if (props.token) {
      fetchOptions.headers = { 'X-GitHub-Token': props.token }
    }
    const response = await fetch(proxyUrl, fetchOptions)

    if (response.headers.get('X-Token-Status') === 'invalid') {
      markTokenInvalid()
    }

    if (!response.ok) {
      if (response.status === 404) {
        throw new Error('该仓库没有 README 文件')
      }
      throw new Error(`获取 README 失败: ${response.status}`)
    }

    const data = await response.json()
    let content = ''
    if (data.content && data.encoding === 'base64') {
      content = decodeBase64(data.content)
    } else if (typeof data === 'string') {
      content = data
    } else {
      throw new Error('无法解析 README 内容')
    }

    const rendered = renderMarkdown(content)
    let finalContent = rendered

    if (data.download_url) {
      const rawBaseUrl = data.download_url.substring(0, data.download_url.lastIndexOf('/') + 1)
      finalContent = replaceRelativeUrls(finalContent, rawBaseUrl)
    }

    readmeContent.value = replaceGithubUrls(finalContent)
  } catch (error) {
    readmeError.value = error.message || '获取 README 失败'
  } finally {
    readmeLoading.value = false
  }
}

const closeReadmeModal = () => {
  showReadmeModal.value = false
  readmeContent.value = ''
  readmeError.value = ''
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

const formatStarCount = (count) => {
  if (!count && count !== 0) return '0'
  if (count >= 1000000) {
    return (count / 1000000).toFixed(1) + 'M'
  }
  if (count >= 1000) {
    return (count / 1000).toFixed(1) + 'K'
  }
  return count.toString()
}

// GitHub 官方语言配色
const languageColors = {
  JavaScript: '#f1e05a',
  TypeScript: '#3178c6',
  Vue: '#41b883',
  Python: '#3572A5',
  Go: '#00ADD8',
  Rust: '#dea584',
  Java: '#b07219',
  'C++': '#f34b7d',
  C: '#555555',
  'C#': '#178600',
  PHP: '#4F5D95',
  Ruby: '#701516',
  Swift: '#F05138',
  Kotlin: '#A97BFF',
  Dart: '#00B4AB',
  Shell: '#89e051',
  HTML: '#e34c26',
  CSS: '#563d7c',
  Dockerfile: '#384d54',
  Makefile: '#427819',
  Lua: '#000080',
  R: '#198CE7',
  Scala: '#c22d40',
  Haskell: '#5e5086',
  Elixir: '#6e4a7e',
  Zig: '#ec915c',
  Assembly: '#6E4C13',
  Objective: '#438eff'
}

const languageColor = (lang) => languageColors[lang] || '#3b82f6'
</script>
