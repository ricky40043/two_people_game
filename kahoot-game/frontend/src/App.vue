<template>
  <div id="app" class="min-h-screen bg-gradient-to-br from-purple-600 via-blue-600 to-blue-700">
    <!-- 全域導航 -->
    <nav v-if="showNavigation" class="fixed top-0 left-0 right-0 z-50 bg-black/20 backdrop-blur-sm">
      <div class="container mx-auto px-4 py-3">
        <div class="flex items-center justify-between">
          <router-link to="/" class="text-white font-bold text-xl flex items-center space-x-2">
            <span class="text-2xl">🎮</span>
            <span>Ricky 遊戲小舖</span>
          </router-link>
          
          <div class="flex items-center space-x-4">
            <!-- 連線狀態指示器 -->
            <div class="flex items-center space-x-2">
              <div 
                :class="[
                  'w-3 h-3 rounded-full',
                  socketStore.isConnected ? 'bg-green-400 animate-pulse' : 'bg-red-400'
                ]"
              ></div>
              <span class="text-white text-sm">
                {{ socketStore.isConnected ? '已連線' : '未連線' }}
              </span>
            </div>
            
            <!-- 房間資訊 -->
            <div v-if="gameStore.currentRoom" class="text-white text-sm">
              房間: {{ gameStore.currentRoom.id }}
            </div>
          </div>
        </div>
      </div>
    </nav>

    <!-- 主要內容 -->
    <main :class="{ 'pt-16': showNavigation }">
      <router-view />
    </main>

    <!-- 全域載入動畫 -->
    <div 
      v-if="isLoading" 
      class="fixed inset-0 bg-black/50 backdrop-blur-sm z-50 flex items-center justify-center"
    >
      <div class="bg-white rounded-2xl p-8 flex flex-col items-center space-y-4">
        <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-purple-600"></div>
        <p class="text-gray-700 font-medium">{{ loadingText }}</p>
      </div>
    </div>

    <!-- 全域錯誤提示 -->
    <div 
      v-if="errorMessage" 
      class="fixed top-20 right-4 bg-red-500 text-white px-6 py-4 rounded-lg shadow-lg z-50 max-w-sm"
    >
      <div class="flex items-center justify-between">
        <div class="flex items-center space-x-2">
          <span class="text-xl">❌</span>
          <span>{{ errorMessage }}</span>
        </div>
        <button @click="clearError" class="text-white hover:text-gray-200">
          ✕
        </button>
      </div>
    </div>

    <!-- 全域成功提示 -->
    <div 
      v-if="successMessage" 
      class="fixed top-20 right-4 bg-green-500 text-white px-6 py-4 rounded-lg shadow-lg z-50 max-w-sm"
    >
      <div class="flex items-center justify-between">
        <div class="flex items-center space-x-2">
          <span class="text-xl">✅</span>
          <span>{{ successMessage }}</span>
        </div>
        <button @click="clearSuccess" class="text-white hover:text-gray-200">
          ✕
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { useGameStore } from '@/stores/game'
import { useSocketStore } from '@/stores/socket'
import { useUIStore } from '@/stores/ui'

const route = useRoute()
const gameStore = useGameStore()
const socketStore = useSocketStore()
const uiStore = useUIStore()

// 計算屬性
const showNavigation = computed(() => {
  // 在全屏遊戲頁面隱藏導航
  return !['game-host', 'game-player'].includes(route.name as string)
})

const isLoading = computed(() => uiStore.isLoading)
const loadingText = computed(() => uiStore.loadingText)
const errorMessage = computed(() => uiStore.errorMessage)
const successMessage = computed(() => uiStore.successMessage)

// 方法
const clearError = () => uiStore.clearError()
const clearSuccess = () => uiStore.clearSuccess()

// 生命週期
onMounted(() => {
  // 初始化 WebSocket 連線
  socketStore.connect()
})

onUnmounted(() => {
  // 清理 WebSocket 連線
  socketStore.disconnect()
})
</script>

<style>
/* 全域樣式已在 style.css 中定義 */
</style>