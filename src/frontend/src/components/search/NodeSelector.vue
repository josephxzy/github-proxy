<template>
  <div class="relative z-[9999]">
    <div v-if="isSharedMode" class="mb-3 flex items-center gap-2">
      <span class="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-medium bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400">
        <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M8 12h8"/><path d="M12 8v8"/></svg>
        共享模式 - 已连接到节点调度中心
      </span>
    </div>
    <div v-if="networkSpeed" class="mb-3 flex flex-wrap items-center gap-2 sm:gap-4 text-xs">
      <span class="inline-flex items-center gap-1 px-2.5 py-1 rounded-md bg-blue-50 dark:bg-blue-900/20 text-blue-600 dark:text-blue-400">
        <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 19V5"/><polyline points="5,12 12,5 19,12"/></svg>
        {{ formatNetSpeed(networkSpeed.uploadSpeed) }}
      </span>
      <span class="inline-flex items-center gap-1 px-2.5 py-1 rounded-md bg-purple-50 dark:bg-purple-900/20 text-purple-600 dark:text-purple-400">
        <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 5v14"/><polyline points="19,12 12,19 5,12"/></svg>
        {{ formatNetSpeed(networkSpeed.downloadSpeed) }}
      </span>
      <span v-if="networkSpeed.interfaceName" class="text-gray-400 dark:text-gray-500">{{ networkSpeed.interfaceName }}</span>
    </div>
    <div class="flex flex-col md:flex-row md:items-center gap-4 mb-6">
      <span class="hidden md:block text-gray-700 dark:text-gray-300 font-medium whitespace-nowrap">节点选择：</span>
      <div class="relative w-full md:flex-1 md:max-w-xl">
        <button
          type="button"
          @click="toggleList"
          :disabled="isLoadingNodes"
          class="w-full px-4 py-3 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded-lg focus:outline-none transition-all text-left disabled:opacity-50 disabled:cursor-not-allowed h-[48px] flex items-center justify-between">
          <div class="flex items-center gap-2 text-gray-500 dark:text-gray-400 w-full">
            <svg v-if="isLoadingNodes" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="w-5 h-5 animate-spin" aria-hidden="true">
              <path d="M21 12a9 9 0 1 1-6.219-8.56"></path>
            </svg>
            <span>{{ selectedNode ? selectedNode.name : (isLoadingNodes ? '加载节点列表中...' : '选择节点') }}</span>
          </div>
          <svg v-if="!isLoadingNodes" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="text-gray-500 dark:text-gray-400 flex-shrink-0" :class="{ 'rotate-180': showList }">
            <path d="m6 9 6 6 6-6"></path>
          </svg>
        </button>
        <div v-if="showList && !isLoadingNodes" class="absolute z-10 mt-1 w-full bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded-lg shadow-lg max-h-80 overflow-auto">
          <div v-for="node in displayedNodes" :key="node.id"
            @click="selectNode(node)"
            class="px-4 py-3 hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer flex items-center justify-between transition-colors"
            :class="{ 'bg-blue-50 dark:bg-blue-900/20': selectedNode?.id === node.id }">
            <div class="flex flex-col">
              <span class="text-gray-900 dark:text-gray-100 font-medium">{{ node.name }}</span>
              <div class="flex items-center gap-2 mt-0.5">
                <span v-if="node.isLocal" class="text-xs text-blue-500">推荐 · 本地节点</span>
                <span v-if="node.isNew" class="text-xs text-green-500">新节点</span>
                <span v-if="node.isBusy" class="text-xs text-orange-500">高负载中</span>
              </div>
            </div>
            <div class="text-right text-xs text-gray-500 dark:text-gray-400 whitespace-nowrap ml-2">
              <span v-if="node.score !== null && !node.isLocal">评分: {{ node.score }}</span>
              <span v-if="node.latency !== null">{{ typeof node.latency === 'number' ? node.latency + 'ms' : node.latency }}</span>
              <span v-if="node.speed"> | {{ formatSpeed(node.speed) }}</span>
            </div>
          </div>
          <button
            v-if="nodes.length > 5"
            @click="toggleShowAll"
            class="w-full px-4 py-2 text-sm text-blue-600 dark:text-blue-400 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors border-t border-gray-200 dark:border-gray-700">
            {{ showAll ? '收起' : `显示全部 ${nodes.length} 个节点` }}
          </button>
        </div>
      </div>
      <button
        type="button"
        @click="$emit('speed-test')"
        :disabled="isLoadingNodes || nodes.length === 0"
        class="w-full md:w-auto flex items-center justify-center gap-2 px-4 py-3 rounded-lg border transition-all whitespace-nowrap bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-100 border-gray-300 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700 disabled:opacity-50 disabled:cursor-not-allowed h-[48px]"
        title="开始节点测速">
        <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="w-4 h-4" aria-hidden="true">
          <path d="m12 14 4-4"></path>
          <path d="M3.34 19a10 10 0 1 1 17.32 0"></path>
        </svg>
        <span class="text-sm font-medium">节点测速</span>
      </button>
      <button
        type="button"
        @click="$emit('toggle-releases')"
        class="w-full md:w-auto flex items-center justify-center md:justify-start gap-2 px-4 py-3 rounded-lg border transition-all whitespace-nowrap h-[48px] bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-100 border-gray-300 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700">
        <div class="w-10 h-6 rounded-full transition-all relative" :class="isReleasesMode ? 'bg-blue-600' : 'bg-gray-300 dark:bg-gray-600'">
          <div class="absolute top-0.5 w-5 h-5 bg-white rounded-full transition-transform duration-300" :class="isReleasesMode ? 'translate-x-4' : 'translate-x-0.5'"></div>
        </div>
        <span class="text-sm font-medium">获取Releases列表</span>
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'

const props = defineProps({
  nodes: {
    type: Array,
    default: () => []
  },
  selectedNode: {
    type: Object,
    default: null
  },
  isLoadingNodes: {
    type: Boolean,
    default: false
  },
  isSharedMode: {
    type: Boolean,
    default: false
  },
  isReleasesMode: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['select-node', 'speed-test', 'toggle-releases'])

const showList = ref(false)
const showAll = ref(false)
const networkSpeed = ref(null)
let pollTimer = null

const fetchNetworkSpeed = async () => {
  try {
    const res = await fetch('/api/network/stats', { cache: 'no-store' })
    if (res.ok) {
      const data = await res.json()
      if (data.uploadSpeed > 0 || data.downloadSpeed > 0) {
        networkSpeed.value = data
      }
    }
  } catch {}
}
onMounted(() => {
  fetchNetworkSpeed()
  pollTimer = setInterval(fetchNetworkSpeed, 1000)
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})

const displayedNodes = computed(() => {
  if (showAll.value || props.nodes.length <= 5) {
    return props.nodes
  }
  const localNode = props.nodes.find(n => n.isLocal)
  const externalNodes = props.nodes.filter(n => !n.isLocal)
  const result = localNode ? [localNode] : []
  return [...result, ...externalNodes.slice(0, 4)]
})

const toggleList = () => {
  if (!props.isLoadingNodes) {
    showList.value = !showList.value
  }
}

const toggleShowAll = () => {
  showAll.value = !showAll.value
}

const selectNode = (node) => {
  emit('select-node', node)
  showList.value = false
}

const formatSpeed = (speed) => {
  if (speed === null || speed === undefined) return ''
  if (speed >= 1024) return (speed / 1024).toFixed(1) + ' MB/s'
  return speed.toFixed(0) + ' KB/s'
}

const formatNetSpeed = (bytesPerSec) => {
  if (!bytesPerSec || bytesPerSec <= 0) return '0 B/s'
  const units = ['B/s', 'KB/s', 'MB/s', 'GB/s']
  let i = 0
  let value = bytesPerSec
  while (value >= 1024 && i < units.length - 1) {
    value /= 1024
    i++
  }
  if (i === 0) return `${value.toFixed(0)} ${units[i]}`
  return `${value.toFixed(1)} ${units[i]}`
}
</script>
