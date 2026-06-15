<template>
  <main class="flex-1 py-8 px-4 sm:px-6 lg:px-8 bg-gray-50 dark:bg-gray-950 transition-colors duration-300">
    <!-- 主页：搜索入口 -->
    <div v-if="currentView === 'home'" class="w-full max-w-[1000px] mx-auto">
      <div class="pt-10">
        <div class="text-center w-full mb-12">
          <h1 class="font-bold mb-4 text-6xl text-gray-900 dark:text-white transition-colors duration-300">Github <span class="text-blue-600">Proxy</span></h1>
          <p class="text-gray-600 dark:text-gray-400 text-base">支持 API、Git Clone、Releases、Archive、Gist、Raw 等资源加速下载，提升 GitHub 文件下载体验。</p>
        </div>

        <!-- 搜索框组件 -->
        <SearchBox
          v-model="githubUrl"
          :is-releases-mode="isReleasesMode"
          @submit="handleAction"
        />

        <!-- Token 弹窗 -->
        <Teleport to="body">
          <div v-if="showTokenModal" class="fixed inset-0 z-[10002] flex items-center justify-center bg-black/50" @click.self="closeTokenModal">
            <div class="bg-white dark:bg-gray-800 rounded-xl shadow-2xl w-full max-w-md mx-4 p-6 relative">
              <button @click="closeTokenModal" class="absolute top-3 right-3 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors">
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>

              <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">设置 GitHub Token</h3>

              <div class="relative">
                <input
                  :type="tokenVisible ? 'text' : 'password'"
                  v-model="tokenDraft"
                  placeholder="ghp_xxxxxxxxxxxxxxxxxxxx"
                  class="w-full px-3 py-2 pr-10 text-sm bg-gray-50 dark:bg-gray-900 text-gray-900 dark:text-gray-100 border rounded-lg focus:outline-none focus:ring-2 transition-colors"
                  :class="tokenError ? 'border-red-500 focus:ring-red-500' : 'border-gray-300 dark:border-gray-700 focus:ring-blue-500'"
                  @input="validateTokenDraft" />
                <button
                  @click="tokenVisible = !tokenVisible"
                  class="absolute right-2 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                  :title="tokenVisible ? '隐藏' : '显示'">
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path v-if="!tokenVisible" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" /><path v-if="!tokenVisible" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                    <path v-else stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
                  </svg>
                </button>
              </div>

              <p v-if="tokenError" class="mt-1 text-xs text-red-500">{{ tokenError }}</p>

              <div class="flex items-center gap-2 mt-4">
                <button
                  @click="handleTokenClear"
                  class="flex-1 px-4 py-2 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 text-sm font-medium rounded-lg transition-colors">
                  清空
                </button>
                <button
                  @click="handleTokenConfirm"
                  :disabled="!!tokenError"
                  class="flex-1 px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-400 disabled:cursor-not-allowed text-white text-sm font-medium rounded-lg transition-colors">
                  确认
                </button>
              </div>
            </div>
          </div>
        </Teleport>

        <!-- 功能开关栏 -->
        <div class="flex flex-col md:flex-row md:items-center md:justify-center gap-4 mb-6">
          <button
            type="button"
            @click="toggleTokenMode"
            class="w-full md:w-auto flex items-center justify-center gap-2 px-4 py-3 rounded-lg border transition-all whitespace-nowrap h-[48px] bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700"
            :class="token ? 'text-gray-900 dark:text-gray-100' : 'text-gray-700 dark:text-gray-100'">
            <div class="w-10 h-6 rounded-full transition-all relative" :class="token ? 'bg-blue-600' : 'bg-gray-300 dark:bg-gray-600'">
              <div class="absolute top-0.5 w-5 h-5 bg-white rounded-full transition-transform duration-300" :class="token ? 'translate-x-4' : 'translate-x-0.5'"></div>
            </div>
            <span class="text-sm font-medium">私有仓库</span>
          </button>
          <button
            type="button"
            @click="toggleReleasesMode"
            class="w-full md:w-auto flex items-center justify-center md:justify-start gap-2 px-4 py-3 rounded-lg border transition-all whitespace-nowrap h-[48px] bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-100 border-gray-300 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700">
            <div class="w-10 h-6 rounded-full transition-all relative" :class="isReleasesMode ? 'bg-blue-600' : 'bg-gray-300 dark:bg-gray-600'">
              <div class="absolute top-0.5 w-5 h-5 bg-white rounded-full transition-transform duration-300" :class="isReleasesMode ? 'translate-x-4' : 'translate-x-0.5'"></div>
            </div>
            <span class="text-sm font-medium">版本列表</span>
          </button>
        </div>
      </div>

      <!-- 帮助按钮组件 -->
      <HelpButton v-if="currentView === 'home'" />
    </div>

    <!-- Releases 列表页面 -->
    <ReleasesView
      v-else-if="currentView === 'releases'"
      :repoUrl="currentRepoUrl"
      :proxyHost="proxyHost"
      :fromView="previousView"
      :token="token"
      @back="goBack"
    />

    <!-- 搜索结果页面 -->
    <SearchResultsView
      v-else-if="currentView === 'search'"
      :searchQuery="searchQuery"
      :proxyHost="proxyHost"
      :token="token"
      @back="goHome"
      @view-releases="handleViewReleasesFromSearch"
    />
  </main>
</template>

<script setup>
import { ref, computed } from 'vue'
import ReleasesView from './ReleasesView.vue'
import SearchResultsView from './SearchResultsView.vue'
import SearchBox from '../components/search/SearchBox.vue'
import HelpButton from '../components/common/HelpButton.vue'
import { useToken } from '../composables/useToken'

const { token, clearToken } = useToken()
const showTokenModal = ref(false)
const tokenVisible = ref(false)
const tokenDraft = ref(token.value)
const tokenError = ref('')

const proxyHost = window.location.origin

const toggleTokenMode = () => {
  tokenDraft.value = token.value
  tokenError.value = ''
  validateTokenDraft()
  tokenVisible.value = false
  showTokenModal.value = true
}

const closeTokenModal = () => {
  showTokenModal.value = false
  if (!tokenDraft.value.trim()) {
    tokenError.value = ''
  }
}

const validateTokenDraft = () => {
  const val = tokenDraft.value.trim()
  if (!val) {
    tokenError.value = ''
    return
  }
  if (!/^gh[poaurs]_[A-Za-z0-9_]+$/.test(val) && !/^github_pat_/.test(val)) {
    tokenError.value = '格式不正确，GitHub Token 通常以 ghp_、gho_、github_pat_ 等开头'
  } else {
    tokenError.value = ''
  }
}

const handleTokenClear = () => {
  tokenDraft.value = ''
  tokenError.value = ''
}

const handleTokenConfirm = () => {
  token.value = tokenDraft.value.trim()
  showTokenModal.value = false
}

const githubUrl = ref('')
const isReleasesMode = ref(false)

// 页面路由状态
const currentView = ref('home')
const previousView = ref('home')
const currentRepoUrl = ref('')
const searchQuery = ref('')
const navigationHistory = ref([])

// 获取输入类型（用于 handleAction 判断）
const inputType = computed(() => {
  const url = githubUrl.value.trim()

  if (!url) return 'empty'

  if (isReleasesMode.value) return 'releases'

  if (url.startsWith('http://') || url.startsWith('https://')) {
    const repoPattern = /^https?:\/\/github\.com\/[^/]+\/[^/]+(?:\.git)?$/
    if (repoPattern.test(url)) return 'repo'
    return 'download'
  }

  return 'search'
})

// 页面导航方法
const navigateTo = (view, options = {}) => {
  if (currentView.value !== 'home') {
    navigationHistory.value.push({
      view: currentView.value,
      repoUrl: currentRepoUrl.value,
      searchQuery: searchQuery.value
    })
  }

  previousView.value = currentView.value
  currentView.value = view

  if (options.repoUrl !== undefined) {
    currentRepoUrl.value = options.repoUrl
  }
  if (options.searchQuery !== undefined) {
    searchQuery.value = options.searchQuery
  }
}

const goBack = () => {
  if (navigationHistory.value.length > 0) {
    const previousState = navigationHistory.value.pop()
    previousView.value = previousState.view === 'home' ? 'home' : currentView.value
    currentView.value = previousState.view
    currentRepoUrl.value = previousState.repoUrl || ''
    searchQuery.value = previousState.searchQuery || ''
  } else {
    goHome()
  }
}

const goHome = () => {
  currentView.value = 'home'
  previousView.value = 'home'
  currentRepoUrl.value = ''
  searchQuery.value = ''
  navigationHistory.value = []
}

const handleAction = () => {
  const url = githubUrl.value.trim()
  if (!url) return

  switch (inputType.value) {
    case 'releases':
      navigateTo('releases', { repoUrl: url })
      break
    case 'repo':
      downloadRepoZip()
      break
    case 'download':
      downloadFile()
      break
    case 'search':
      navigateTo('search', { searchQuery: url })
      break
    default:
      if (isReleasesMode.value) {
        navigateTo('releases', { repoUrl: url })
      } else {
        if (url.startsWith('http')) {
          downloadFile()
        } else {
          navigateTo('search', { searchQuery: url })
        }
      }
  }
}

const downloadFile = () => {
  let baseUrl = proxyHost + '/' + githubUrl.value.trim()
  if (token.value) {
    baseUrl += (baseUrl.includes('?') ? '&' : '?') + 'token=' + token.value
  }
  window.open(baseUrl, '_blank')
}

const downloadRepoZip = async () => {
  if (!githubUrl.value) return

  const repoUrl = githubUrl.value.replace('.git', '').trim()
  const parts = repoUrl.split('/').filter(p => p)

  if (parts.length < 4) return

  const owner = parts[2]
  const repo = parts[3]

  if (!owner || !repo) return

  let branch = 'main'
  try {
    const resp = await fetch(`/api/repo/${owner}/${repo}/branch`)
    if (resp.ok) {
      const data = await resp.json()
      branch = data.branch || 'main'
    }
  } catch (e) {
    console.error('Failed to fetch default branch, using main:', e)
  }

  const zipUrl = `https://github.com/${owner}/${repo}/archive/refs/heads/${branch}.zip`
  let proxyUrl = proxyHost + '/' + zipUrl + '?fast=1'
  if (token.value) {
    proxyUrl += '&token=' + token.value
  }
  window.open(proxyUrl, '_blank')
}

const handleViewReleasesFromSearch = (repoUrl) => {
  navigateTo('releases', { repoUrl: repoUrl })
}

const toggleReleasesMode = () => {
  isReleasesMode.value = !isReleasesMode.value
}
</script>
