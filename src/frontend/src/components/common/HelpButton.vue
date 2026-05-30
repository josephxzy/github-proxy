<template>
  <div
    class="fixed z-[10002] select-none"
    :style="{
      right: position.x + 'px',
      top: position.y + 'px',
      cursor: isDragging ? 'grabbing' : 'grab'
    }"
    @mousedown="startDrag"
    @touchstart="startDragTouch"
  >
    <!-- 帮助按钮（收起状态） -->
    <div
      v-if="!isOpen"
      @click="togglePanel"
      class="w-12 h-12 sm:w-14 sm:h-14 bg-blue-600 hover:bg-blue-700 active:bg-blue-800 text-white rounded-full shadow-lg flex items-center justify-center transition-all hover:scale-110 active:scale-95"
    >
      <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <circle cx="12" cy="12" r="10"></circle>
        <path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"></path>
        <line x1="12" y1="17" x2="12.01" y2="17"></line>
      </svg>
    </div>

    <!-- 帮助面板（展开状态） -->
    <div
      v-else
      class="bg-white dark:bg-gray-800 rounded-xl shadow-2xl border border-gray-200 dark:border-gray-700 w-[calc(100vw-32px)] sm:w-80 max-h-[80vh] sm:max-h-[70vh] overflow-hidden fixed sm:relative"
      :style="isOpen && isMobile ? { right: '16px', top: 'auto', bottom: '16px', left: '16px' } : {}"
    >
      <!-- 面板头部 -->
      <div class="flex items-center justify-between p-3 sm:p-4 border-b border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900">
        <h3 class="font-semibold text-sm sm:text-base text-gray-900 dark:text-white flex items-center gap-2">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="text-blue-600">
            <circle cx="12" cy="12" r="10"></circle>
            <path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"></path>
            <line x1="12" y1="17" x2="12.01" y2="17"></line>
          </svg>
          搜索语法指南
        </h3>
        <button
          @click="togglePanel"
          class="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 transition-colors p-1 -m-1"
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <line x1="18" y1="6" x2="6" y2="18"></line>
            <line x1="6" y1="6" x2="18" y2="18"></line>
          </svg>
        </button>
      </div>

      <!-- 面板内容 -->
      <div class="p-3 sm:p-4 overflow-y-auto max-h-[calc(80vh-52px)] sm:max-h-[calc(70vh-60px)] space-y-3 sm:space-y-4 text-xs sm:text-sm">
        <!-- 全局搜索 -->
        <div class="space-y-2">
          <div class="font-semibold text-gray-900 dark:text-white flex items-center gap-2">
            <span class="w-2 h-2 bg-green-500 rounded-full"></span>
            全局关键词搜索
          </div>
          <div class="bg-gray-50 dark:bg-gray-900 rounded-lg p-3 font-mono text-xs text-gray-700 dark:text-gray-300">
            facebook<br>
            react vue<br>
            python web scraping
          </div>
          <p class="text-gray-600 dark:text-gray-400 text-xs">在所有字段中搜索，结果最全面</p>
        </div>

        <!-- 用户仓库搜索 -->
        <div class="space-y-2">
          <div class="font-semibold text-gray-900 dark:text-white flex items-center gap-2">
            <span class="w-2 h-2 bg-blue-500 rounded-full"></span>
            用户仓库搜索
          </div>
          <div class="bg-gray-50 dark:bg-gray-900 rounded-lg p-3 font-mono text-xs text-gray-700 dark:text-gray-300">
            user/repo<br>
            facebook/react<br>
            vuejs/vue
          </div>
          <p class="text-gray-600 dark:text-gray-400 text-xs">搜索指定用户的特定仓库</p>
        </div>

        <!-- 高级语法 -->
        <div class="space-y-2">
          <div class="font-semibold text-gray-900 dark:text-white flex items-center gap-2">
            <span class="w-2 h-2 bg-purple-500 rounded-full"></span>
            高级限定符
          </div>
          <div class="space-y-2">
            <div class="bg-gray-50 dark:bg-gray-900 rounded-lg p-3">
              <div class="font-mono text-xs text-blue-600 dark:text-blue-400 mb-1">user:用户名</div>
              <p class="text-xs text-gray-600 dark:text-gray-400">搜索该用户的所有仓库</p>
            </div>
            <div class="bg-gray-50 dark:bg-gray-900 rounded-lg p-3">
              <div class="font-mono text-xs text-blue-600 dark:text-blue-400 mb-1">repo:仓库名</div>
              <p class="text-xs text-gray-600 dark:text-gray-400">按仓库名精确搜索</p>
            </div>
            <div class="bg-gray-50 dark:bg-gray-900 rounded-lg p-3">
              <div class="font-mono text-xs text-blue-600 dark:text-blue-400 mb-1">desc:关键词</div>
              <p class="text-xs text-gray-600 dark:text-gray-400">在描述中搜索</p>
            </div>
            <div class="bg-gray-50 dark:bg-gray-900 rounded-lg p-3">
              <div class="font-mono text-xs text-blue-600 dark:text-blue-400 mb-1">readme:关键词</div>
              <p class="text-xs text-gray-600 dark:text-gray-400">在 README 中搜索</p>
            </div>
          </div>
        </div>

        <!-- 链接下载 -->
        <div class="space-y-2">
          <div class="font-semibold text-gray-900 dark:text-white flex items-center gap-2">
            <span class="w-2 h-2 bg-orange-500 rounded-full"></span>
            链接下载
          </div>
          <div class="bg-gray-50 dark:bg-gray-900 rounded-lg p-3 font-mono text-xs text-gray-700 dark:text-gray-300">
            https://github.com/user/repo<br>
            → 下载 ZIP 压缩包<br><br>
            https://.../file.ext<br>
            → 下载文件
          </div>
          <p class="text-gray-600 dark:text-gray-400 text-xs">粘贴 GitHub 链接直接下载</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'

const isOpen = ref(false)
const isDragging = ref(false)
const isMobile = ref(false)
const position = ref({ x: 0, y: 0 })
const dragOffset = ref({ x: 0, y: 0 })

// 检测是否为移动设备
const checkIsMobile = () => {
  isMobile.value = window.innerWidth < 640 || 'ontouchstart' in window
}

onMounted(() => {
  if (typeof window !== 'undefined') {
    checkIsMobile()

    // 根据设备类型设置不同的默认位置
    if (isMobile.value) {
      // 手机：右下角位置，避免遮挡主要内容
      position.value = {
        x: Math.round(window.innerWidth * 0.05),
        y: Math.round(window.innerHeight * 0.7)
      }
    } else {
      // 桌面：右上角位置
      position.value = {
        x: 10,
        y: Math.round(window.innerHeight * 0.16)
      }
    }

    // 监听窗口大小变化
    window.addEventListener('resize', checkIsMobile)
  }
})

onUnmounted(() => {
  if (typeof window !== 'undefined') {
    window.removeEventListener('resize', checkIsMobile)
  }
})

const togglePanel = () => {
  isOpen.value = !isOpen.value
}

const startDrag = (e) => {
  if (isOpen.value) return

  isDragging.value = true
  dragOffset.value = {
    x: e.clientX,
    y: e.clientY
  }

  document.addEventListener('mousemove', onDrag)
  document.addEventListener('mouseup', stopDrag)
}

const startDragTouch = (e) => {
  if (isOpen.value) return

  isDragging.value = true
  const touch = e.touches[0]
  dragOffset.value = {
    x: touch.clientX,
    y: touch.clientY
  }

  document.addEventListener('touchmove', onDragTouch, { passive: false })
  document.addEventListener('touchend', stopDrag)
}

const onDrag = (e) => {
  if (!isDragging.value) return
  e.preventDefault()

  // 计算鼠标移动的偏移量（当前坐标 - 上次坐标）
  const deltaX = e.clientX - dragOffset.value.x
  const deltaY = e.clientY - dragOffset.value.y

  // 更新位置
  // 注意：因为使用 right 定位，x轴方向需要反向计算
  position.value = {
    x: Math.max(0, Math.min(window.innerWidth - 48, position.value.x - deltaX)),
    y: Math.max(0, Math.min(window.innerHeight - 48, position.value.y + deltaY))
  }

  // 记录当前位置作为下次计算的基准
  dragOffset.value = {
    x: e.clientX,
    y: e.clientY
  }
}

const onDragTouch = (e) => {
  if (!isDragging.value) return
  e.preventDefault()

  const touch = e.touches[0]
  // 计算触摸点移动的偏移量（当前坐标 - 上次坐标）
  const deltaX = touch.clientX - dragOffset.value.x
  const deltaY = touch.clientY - dragOffset.value.y

  // 更新位置（x轴反向因为 right 定位）
  position.value = {
    x: Math.max(0, Math.min(window.innerWidth - 48, position.value.x - deltaX)),
    y: Math.max(0, Math.min(window.innerHeight - 48, position.value.y + deltaY))
  }

  dragOffset.value = {
    x: touch.clientX,
    y: touch.clientY
  }
}

const stopDrag = () => {
  isDragging.value = false
  document.removeEventListener('mousemove', onDrag)
  document.removeEventListener('mouseup', stopDrag)
  document.removeEventListener('touchmove', onDragTouch)
  document.removeEventListener('touchend', stopDrag)
}
</script>
