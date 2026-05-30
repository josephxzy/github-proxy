<template>
  <main class="flex-1 flex items-start justify-center py-8 px-4 sm:px-6 lg:px-8 bg-gradient-to-br from-gray-100 to-gray-200 dark:from-gray-900 dark:to-gray-950 transition-colors duration-300">
    <div class="w-full max-w-[1000px] mx-auto">
      <div class="pt-6">
        <button @click="$emit('back')" class="inline-flex items-center gap-2 text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300 font-medium transition-colors mb-6">
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="m15 18-6-6 6-6"></path>
          </svg>
          返回首页
        </button>

        <!-- 搜索选项栏 -->
        <div v-if="!loadingSearch && !searchError && searchResults.length > 0" class="bg-white/80 dark:bg-gray-800/80 backdrop-blur-sm border border-gray-200/50 dark:border-gray-700/50 rounded-xl p-4 mb-6 space-y-3">
          <!-- 排序和范围选项 -->
          <div class="flex flex-wrap items-center gap-3">
            <!-- 排序选项 -->
            <div class="flex items-center gap-2">
              <label class="text-sm font-medium text-gray-700 dark:text-gray-300">排序:</label>
              <select
                v-model="sortBy"
                @change="handleSortChange"
                class="px-3 py-1.5 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-lg text-sm text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="">最佳匹配</option>
                <option value="stars">星标数</option>
                <option value="forks">Fork 数</option>
                <option value="updated">最近更新</option>
              </select>
            </div>

            <!-- 搜索范围 -->
            <div class="flex items-center gap-2">
              <label class="text-sm font-medium text-gray-700 dark:text-gray-300">范围:</label>
              <select
                v-model="searchScope"
                @change="handleScopeChange"
                class="px-3 py-1.5 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-lg text-sm text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="all">全部</option>
              <option value="name">名称</option>
              <option value="description">描述</option>
              <option value="readme">README</option>
              </select>
            </div>

            <!-- 结果统计 -->
            <div class="w-full sm:ml-auto text-sm text-gray-500 dark:text-gray-400">
              共找到 <span class="font-semibold text-blue-600 dark:text-blue-400">{{ totalResults }}</span> 个仓库
              <span v-if="totalResults > 0" class="ml-2">(第 {{ currentPage }}/{{ totalPages }} 页)</span>
            </div>
          </div>
        </div>

        <div v-if="loadingSearch" class="text-center py-12">
          <svg class="animate-spin h-8 w-8 text-blue-600 mx-auto" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
          <p class="mt-4 text-gray-600 dark:text-gray-400">搜索仓库中...</p>
        </div>

        <div v-else-if="searchError" class="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-xl p-6 text-center">
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

          <div v-for="repo in searchResults" :key="repo.id" class="bg-white/60 dark:bg-gray-800/60 backdrop-blur-sm border border-gray-200/50 dark:border-gray-700/50 rounded-xl p-4 sm:p-6 hover:shadow-md transition-shadow">
            <div class="flex flex-col sm:flex-row sm:items-start gap-3 sm:gap-0">
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-3 mb-2">
                  <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="text-gray-700 dark:text-gray-300 flex-shrink-0">
                    <path d="M9 19c-5 1.5-5-2.5-7-3m14 6v-3.87a3.37 3.37 0 0 0-.94-2.61c3.14-.35 6.44-1.54 6.44-7A5.44 5.44 0 0 0 20 4.77 5.07 5.07 0 0 0 19.91 1S18.73.65 16 2.48a13.38 13.38 0 0 0-7 0C6.27.65 5.09 1 5.09 1A5.07 5.07 0 0 0 5 4.77a5.44 5.44 0 0 0-1.5 3.78c0 5.42 3.3 6.61 6.44 7A3.37 3.37 0 0 0 9 18.13V22"></path>
                  </svg>
                  <a :href="repo.html_url" target="_blank" rel="noopener noreferrer" class="text-lg sm:text-xl font-semibold text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300 truncate">
                    {{ repo.full_name }}
                  </a>
                </div>
                <p class="text-sm text-gray-600 dark:text-gray-400 mb-3 line-clamp-2">{{ repo.description || '暂无描述' }}</p>
                <div class="flex flex-wrap items-center gap-2 sm:gap-4 text-xs text-gray-500 dark:text-gray-400">
                  <span v-if="repo.language" class="flex items-center gap-1">
                    <span class="w-3 h-3 rounded-full bg-blue-500 flex-shrink-0"></span>
                    {{ repo.language }}
                  </span>
                  <span class="flex items-center gap-1">
                    <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
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
              </div>
              <div class="flex flex-row sm:flex-col gap-2 sm:ml-4 w-full sm:w-auto">
                <button
                  @click="downloadRepoZip(repo)"
                  :disabled="!selectedNode"
                  class="flex-1 sm:flex-none px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed text-white text-sm font-medium rounded-lg transition-colors whitespace-nowrap"
                >
                  下载 ZIP
                </button>
                <button
                  @click="viewReleases(repo)"
                  class="flex-1 sm:flex-none px-4 py-2 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 text-sm font-medium rounded-lg transition-colors whitespace-nowrap"
                >
                  Releases
                </button>
              </div>
            </div>
          </div>

          <!-- 分页控制 - 统一样式 -->
          <div v-if="totalPages > 1" class="flex items-center justify-center gap-1 mt-6 pt-6 border-t border-gray-200 dark:border-gray-700">
            <button
              @click="prevPage"
              :disabled="currentPage === 1 || loadingSearch"
              class="px-3 py-1.5 border border-gray-300 dark:border-gray-700 rounded-md text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
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
              @click="nextPage"
              :disabled="currentPage === totalPages || loadingSearch"
              class="px-3 py-1.5 border border-gray-300 dark:border-gray-700 rounded-md text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
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
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'

const props = defineProps({
  searchQuery: String,
  selectedNode: Object,
  getNodeUrl: Function
})

const emit = defineEmits(['back', 'view-releases'])

const searchResults = ref([])
const loadingSearch = ref(false)
const searchError = ref('')
const totalResults = ref(0)
const currentPage = ref(1)
const perPage = 30

// 排序选项
const sortBy = ref('')
const sortOrder = ref('desc')

// 搜索范围选项
const searchScope = ref('all')

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

  // 4. 根据搜索范围添加限定符（仅在非特殊查询时）
  // 注意：不要对用户名或仓库名自动添加 in: 限定符
  if (searchScope.value !== 'all' && !baseQuery.includes('user:') && !baseQuery.includes('org:')) {
    const scopeMap = {
      'name': 'in:name',
      'description': 'in:description',
      'readme': 'in:readme'
    }
    baseQuery = `${baseQuery} ${scopeMap[searchScope.value]}`
  }

  return baseQuery
}

const searchRepositories = async (page = 1) => {
  if (!props.searchQuery.trim() || !props.selectedNode) return

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

    // 添加排序参数（仅当选择了排序方式时）
    if (sortBy.value) {
      apiPath += `&sort=${sortBy.value}&order=${sortOrder.value}`
    }

    const proxyUrl = '/' + `https://api.github.com/${apiPath}`
    const response = await fetch(proxyUrl, { cache: 'no-store' })

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

// 排序变更处理
const handleSortChange = () => {
  currentPage.value = 1
  searchRepositories(1)
}

// 搜索范围变更处理
const handleScopeChange = () => {
  currentPage.value = 1
  searchRepositories(1)
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
  if (!props.selectedNode || !repo.html_url) return

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

  // 使用快速模式下载 (fast=1)：跳过预检，提升响应速度
  const zipUrl = `https://github.com/${owner}/${repoName}/archive/refs/heads/${branch}.zip`
  const proxyUrl = props.getNodeUrl(props.selectedNode) + '/' + zipUrl + '?fast=1'
  window.open(proxyUrl, '_blank')
}

const viewReleases = (repo) => {
  emit('view-releases', repo.html_url)
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
</script>
