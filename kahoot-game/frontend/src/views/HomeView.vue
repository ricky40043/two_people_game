<template>
  <div class="home-view min-h-screen flex items-center justify-center p-4">
    <div class="max-w-md w-full">
      <!-- 標題區域 -->
      <div class="text-center mb-12 fade-in">
        <h1 class="text-6xl mb-4">🎮</h1>
        <h2 class="text-4xl font-bold text-white mb-4">
          風格
        </h2>
        <h3 class="text-2xl font-semibold text-white/90 mb-2">
          多人遊戲
        </h3>
        <p class="text-white/70 text-lg">
          即時問答 • 主角輪替 • QR Code 加入
        </p>
      </div>

      <!-- 功能卡片 -->
      <div class="space-y-4 slide-in">
        <!-- 創建房間 -->
        <div class="card card-body group hover:scale-102 transition-transform duration-300">
          <div class="flex items-center justify-between">
            <div class="flex items-center space-x-4">
              <div class="w-12 h-12 bg-green-500 rounded-xl flex items-center justify-center text-2xl">
                🏠
              </div>
              <div>
                <h3 class="text-white font-semibold text-lg">創建房間</h3>
                <p class="text-white/70 text-sm">成為主持人，開始新遊戲</p>
              </div>
            </div>
            <router-link 
              to="/create"
              class="btn btn-success group-hover:scale-105 transition-transform"
            >
              創建
            </router-link>
          </div>
        </div>

        <!-- 加入房間 -->
        <div class="card card-body group hover:scale-102 transition-transform duration-300">
          <div class="flex items-center justify-between">
            <div class="flex items-center space-x-4">
              <div class="w-12 h-12 bg-blue-500 rounded-xl flex items-center justify-center text-2xl">
                🚪
              </div>
              <div>
                <h3 class="text-white font-semibold text-lg">加入房間</h3>
                <p class="text-white/70 text-sm">輸入房間代碼參與遊戲</p>
              </div>
            </div>
            <router-link 
              to="/join"
              class="btn btn-primary group-hover:scale-105 transition-transform"
            >
              加入
            </router-link>
          </div>
        </div>

        <!-- 快速加入 -->
        <div class="card card-body group hover:scale-102 transition-transform duration-300">
          <div class="flex items-center justify-between">
            <div class="flex items-center space-x-4">
              <div class="w-12 h-12 bg-purple-500 rounded-xl flex items-center justify-center text-2xl">
                📱
              </div>
              <div>
                <h3 class="text-white font-semibold text-lg">掃描加入</h3>
                <p class="text-white/70 text-sm">使用相機掃描 QR Code</p>
              </div>
            </div>
            <button 
              @click="startQRScanner"
              :disabled="!canUseCamera"
              class="btn btn-outline group-hover:scale-105 transition-transform"
              :class="{ 'opacity-50 cursor-not-allowed': !canUseCamera }"
            >
              掃描
            </button>
          </div>
        </div>
      </div>

      <!-- 統計資訊 -->
      <div class="mt-12 text-center fade-in" style="animation-delay: 0.5s">
        <div class="grid grid-cols-3 gap-4">
          <div class="card card-body">
            <div class="text-2xl font-bold text-white">{{ stats.activeRooms }}</div>
            <div class="text-white/70 text-sm">活躍房間</div>
          </div>
          <div class="card card-body">
            <div class="text-2xl font-bold text-white">{{ stats.onlinePlayers }}</div>
            <div class="text-white/70 text-sm">線上玩家</div>
          </div>
          <div class="card card-body">
            <div class="text-2xl font-bold text-white">{{ stats.totalGames }}</div>
            <div class="text-white/70 text-sm">總遊戲數</div>
          </div>
        </div>
      </div>

      <!-- 版本資訊 -->
      <div class="mt-8 text-center">
        <p class="text-white/50 text-sm">
          v1.0.0 • 由 Vue 3 + TypeScript 強力驅動
        </p>
      </div>
    </div>

    <!-- QR 掃描器模態框 -->
    <div 
      v-if="showQRScanner" 
      class="fixed inset-0 bg-black/80 backdrop-blur-sm z-50 flex items-center justify-center p-4"
      @click.self="closeQRScanner"
    >
      <div class="bg-white rounded-2xl p-6 max-w-sm w-full">
        <div class="text-center mb-4">
          <h3 class="text-xl font-bold text-gray-800 mb-2">掃描 QR Code</h3>
          <p class="text-gray-600 text-sm">將相機對準房間的 QR Code</p>
        </div>
        
        <div class="bg-gray-100 rounded-xl h-64 flex items-center justify-center mb-4">
          <div class="text-center">
            <div class="text-4xl mb-2">📷</div>
            <div class="text-gray-500">相機預覽區域</div>
            <div class="text-gray-400 text-sm mt-2">（功能開發中）</div>
          </div>
        </div>
        
        <div class="flex space-x-4">
          <button 
            @click="closeQRScanner"
            class="flex-1 btn bg-gray-500 hover:bg-gray-600"
          >
            取消
          </button>
          <router-link 
            to="/join"
            @click="closeQRScanner"
            class="flex-1 btn btn-primary text-center"
          >
            手動輸入
          </router-link>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useSocketStore } from '@/stores/socket'
import { useUIStore } from '@/stores/ui'

const socketStore = useSocketStore()
const uiStore = useUIStore()

// 響應式數據
const showQRScanner = ref(false)
const stats = ref({
  activeRooms: 0,
  onlinePlayers: 0,
  totalGames: 0
})

// 計算屬性
const canUseCamera = computed(() => {
  return 'mediaDevices' in navigator && 'getUserMedia' in navigator.mediaDevices
})

// 方法
const startQRScanner = () => {
  if (!canUseCamera.value) {
    uiStore.showError('您的設備不支援相機功能')
    return
  }
  showQRScanner.value = true
}

const closeQRScanner = () => {
  showQRScanner.value = false
}

const fetchStats = async () => {
  try {
    // TODO: 實際 API 調用
    // 模擬數據
    stats.value = {
      activeRooms: Math.floor(Math.random() * 20) + 1,
      onlinePlayers: Math.floor(Math.random() * 100) + 10,
      totalGames: Math.floor(Math.random() * 1000) + 100
    }
  } catch (error) {
    console.error('獲取統計資訊失敗:', error)
  }
}

// 生命週期
onMounted(() => {
  fetchStats()
  
  // 每30秒更新一次統計
  setInterval(fetchStats, 30000)
})
</script>

<style scoped>
@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes slideIn {
  from {
    opacity: 0;
    transform: translateX(-20px);
  }
  to {
    opacity: 1;
    transform: translateX(0);
  }
}

.fade-in {
  animation: fadeIn 0.6s ease-out;
}

.slide-in {
  animation: slideIn 0.6s ease-out;
}
</style>