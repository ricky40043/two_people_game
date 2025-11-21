<template>
  <div class="create-room-view min-h-screen flex items-center justify-center p-4">
    <div class="max-w-md w-full">
      <!-- 返回按鈕 -->
      <div class="mb-6">
        <router-link 
          to="/"
          class="inline-flex items-center text-white/70 hover:text-white transition-colors"
        >
          <svg class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"></path>
          </svg>
          返回主頁
        </router-link>
      </div>

      <!-- 表單卡片 -->
      <div class="card card-body fade-in">
        <div class="text-center mb-8">
          <div class="text-4xl mb-4">🏠</div>
          <h1 class="text-2xl font-bold text-white mb-2">創建房間</h1>
          <p class="text-white/70">設置您的遊戲參數，成為主持人</p>
        </div>

        <form @submit.prevent="createRoom" class="space-y-6">
          <!-- 主持人名稱 -->
          <div>
            <label class="block text-white/90 font-medium mb-2">
              主持人名稱 <span class="text-red-400">*</span>
            </label>
            <input
              v-model="form.hostName"
              type="text"
              class="input"
              placeholder="請輸入您的名稱"
              required
              maxlength="20"
              :disabled="isSubmitting"
            />
            <div class="text-white/60 text-sm mt-1">
              {{ form.hostName.length }}/20 個字符
            </div>
          </div>

          <!-- 題目數量 -->
          <div>
            <label class="block text-white/90 font-medium mb-2">
              題目數量
            </label>
            <div class="grid grid-cols-4 gap-2">
              <button
                v-for="count in questionCounts"
                :key="count"
                type="button"
                @click="form.totalQuestions = count"
                :class="[
                  'py-2 px-3 rounded-lg border-2 transition-all text-sm font-medium',
                  form.totalQuestions === count
                    ? 'border-white bg-white text-gray-800'
                    : 'border-white/30 text-white hover:border-white/60'
                ]"
                :disabled="isSubmitting"
              >
                {{ count }}
              </button>
            </div>
          </div>

          <!-- 答題時間 -->
          <div>
            <label class="block text-white/90 font-medium mb-2">
              每題答題時間
            </label>
            <div class="grid grid-cols-3 gap-2">
              <button
                v-for="time in answerTimes"
                :key="time.value"
                type="button"
                @click="form.questionTimeLimit = time.value"
                :class="[
                  'py-2 px-3 rounded-lg border-2 transition-all text-sm font-medium',
                  form.questionTimeLimit === time.value
                    ? 'border-white bg-white text-gray-800'
                    : 'border-white/30 text-white hover:border-white/60'
                ]"
                :disabled="isSubmitting"
              >
                {{ time.label }}
              </button>
            </div>
          </div>

          <!-- 遊戲預覽 -->
          <div class="bg-white/10 rounded-xl p-4 border border-white/20">
            <h3 class="text-white font-medium mb-3 flex items-center">
              <span class="text-lg mr-2">📋</span>
              遊戲預覽
            </h3>
            <div class="space-y-2 text-sm">
              <div class="flex justify-between text-white/80">
                <span>主持人：</span>
                <span>{{ form.hostName || '未設定' }}</span>
              </div>
              <div class="flex justify-between text-white/80">
                <span>題目數量：</span>
                <span>{{ form.totalQuestions }} 題</span>
              </div>
              <div class="flex justify-between text-white/80">
                <span>答題時間：</span>
                <span>{{ form.questionTimeLimit }} 秒/題</span>
              </div>
              <div class="flex justify-between text-white/80">
                <span>預計時長：</span>
                <span>{{ estimatedDuration }} 分鐘</span>
              </div>
            </div>
          </div>

          <!-- 提交按鈕 -->
          <button
            type="submit"
            :disabled="!canSubmit || isSubmitting"
            class="w-full btn btn-success text-lg py-4"
            :class="{ 'opacity-50 cursor-not-allowed': !canSubmit || isSubmitting }"
          >
            <span v-if="isSubmitting" class="flex items-center justify-center">
              <div class="loading-spinner mr-2"></div>
              創建中...
            </span>
            <span v-else class="flex items-center justify-center">
              <span class="text-xl mr-2">🚀</span>
              創建房間
            </span>
          </button>
        </form>

        <!-- 提示信息 -->
        <div class="mt-6 text-center">
          <div class="bg-blue-500/20 border border-blue-500/30 rounded-lg p-3">
            <div class="flex items-start">
              <span class="text-blue-400 text-lg mr-2 mt-0.5">💡</span>
              <div class="text-blue-100 text-sm text-left">
                <p class="font-medium mb-1">遊戲說明：</p>
                <ul class="space-y-1 text-xs">
                  <li>• 每題會輪流指定一位「主角」朗讀題目</li>
                  <li>• 答對獲得基礎分 + 速度加分</li>
                  <li>• 主角答對可獲得額外加分</li>
                  <li>• 玩家可通過 QR Code 快速加入</li>
                </ul>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useSocketStore } from '@/stores/socket'
import { useGameStore } from '@/stores/game'
import { useUIStore } from '@/stores/ui'
import { apiService } from '@/services/api'
import { logInfo, logError, captureError } from '@/utils/logger'

const router = useRouter()
const socketStore = useSocketStore()
const gameStore = useGameStore()
const uiStore = useUIStore()

// 表單數據
const form = ref({
  hostName: '',
  totalQuestions: 10,
  questionTimeLimit: 30
})

const isSubmitting = ref(false)

// 選項數據
const questionCounts = [5, 10, 15, 20]
const answerTimes = [
  { label: '15秒', value: 15 },
  { label: '30秒', value: 30 },
  { label: '45秒', value: 45 }
]

// 計算屬性
const canSubmit = computed(() => {
  return form.value.hostName.trim().length >= 2 && !isSubmitting.value
})

const estimatedDuration = computed(() => {
  // 估算遊戲時長（題目時間 + 結果展示時間）
  const questionTime = form.value.totalQuestions * form.value.questionTimeLimit
  const resultTime = form.value.totalQuestions * 10 // 每題結果展示10秒
  const totalSeconds = questionTime + resultTime + 60 // 額外緩衝時間
  return Math.round(totalSeconds / 60)
})

// 方法
const createRoom = async () => {
  if (!canSubmit.value) return

  isSubmitting.value = true
  uiStore.setLoading(true, '正在創建房間...')

  try {
    logInfo('VIEW_CREATE_ROOM', '開始創建房間', {
      hostName: form.value.hostName,
      totalQuestions: form.value.totalQuestions,
      questionTimeLimit: form.value.questionTimeLimit
    })

    // 詳細日誌記錄
    if (window.debugLogger) {
      window.debugLogger.info('CREATE_ROOM', '開始創建房間', {
        hostName: form.value.hostName,
        totalQuestions: form.value.totalQuestions,
        questionTimeLimit: form.value.questionTimeLimit
      })
    }

    // 1. 先通過 HTTP API 創建房間
    console.log('📡 通過 HTTP API 創建房間...')
    const roomData = await apiService.createRoom({
      hostName: form.value.hostName,
      gameMode: 'two_types',
      totalQuestions: form.value.totalQuestions,
      questionTimeLimit: form.value.questionTimeLimit
    })

    if (window.debugLogger) {
      window.debugLogger.info('CREATE_ROOM', 'HTTP API 創建房間成功', roomData)
    }

    console.log('✅ 房間創建成功:', roomData)

    logInfo('VIEW_CREATE_ROOM', 'API 創建房間成功', roomData)

    // 2. 設置房間信息到 store
    gameStore.setRoom({
      id: roomData.roomId,
      hostId: '', // WebSocket 連接後會更新
      hostName: form.value.hostName,
      status: 'waiting',
      players: {},
      currentQuestion: 0,
      totalQuestions: form.value.totalQuestions,
      questionTimeLimit: form.value.questionTimeLimit,
      currentHost: '',
      timeLeft: 0,
      questions: [],
      createdAt: new Date(),
      roomUrl: roomData.joinUrl,
      joinCode: roomData.roomId
    })

    // 3. 設置當前玩家為主持人
    gameStore.setPlayer({
      id: '', // WebSocket 連接後會分配
      name: form.value.hostName,
      roomId: roomData.roomId,
      score: 0,
      isHost: true,
      isConnected: false, // 尚未連接 WebSocket
      lastActivity: new Date()
    })

    // 4. 建立 WebSocket 連接到指定房間
    console.log('🔗 建立 WebSocket 連接...')
    
    if (window.debugLogger) {
      window.debugLogger.info('CREATE_ROOM', '檢查 WebSocket 連接狀態', {
        isConnected: socketStore.isConnected,
        willForceReconnect: true
      })
    }
    
    // 強制重新建立連接，確保是全新的連接
    if (socketStore.isConnected) {
      console.log('🔄 斷開現有連接，準備重新連接...')
      socketStore.disconnect()
      // 短暫等待確保連接完全關閉
      await new Promise(resolve => setTimeout(resolve, 500))
    }
    
    console.log('🔗 開始建立新的 WebSocket 連接...')
    socketStore.connect()
    
    // 等待 WebSocket 連接建立
    await new Promise((resolve, reject) => {
      let attempts = 0
      const maxAttempts = 50 // 5秒超時
      
      const checkConnection = () => {
        attempts++
        if (window.debugLogger && attempts % 10 === 0) {
          window.debugLogger.debug('CREATE_ROOM', `等待 WebSocket 連接... 嘗試 ${attempts}/${maxAttempts}`)
        }
        
        if (socketStore.isConnected) {
          resolve(true)
        } else if (attempts >= maxAttempts) {
          reject(new Error('WebSocket 連接超時'))
        } else {
          setTimeout(checkConnection, 100)
        }
      }
      checkConnection()
    })

    if (window.debugLogger) {
      window.debugLogger.info('CREATE_ROOM', 'WebSocket 連接成功，準備加入房間')
    }

    logInfo('VIEW_CREATE_ROOM', 'WebSocket 連線完成，即將以主持人身分加入', {
      roomId: roomData.roomId
    })

    // 5. 通過 WebSocket 加入房間作為主持人
    socketStore.sendMessage({
      type: 'JOIN_AS_HOST',
      data: {
        roomId: roomData.roomId,
        hostName: form.value.hostName
      }
    })

    // 停止 loading 狀態
    isSubmitting.value = false
    uiStore.setLoading(false)

    // 6. 跳轉到等待大廳
    console.log('🎯 跳轉到大廳:', `/lobby/${roomData.roomId}`)
    router.push(`/lobby/${roomData.roomId}`)
    
  } catch (error: unknown) {
    captureError('VIEW_CREATE_ROOM', error, {
      form: { ...form.value }
    })
    const err = error instanceof Error ? error : new Error(String(error))
    console.error('❌ 創建房間失敗:', err)

    if (window.debugLogger) {
      window.debugLogger.error('CREATE_ROOM', '創建房間失敗', {
        error: err.message,
        stack: err.stack
      })
    }

    let errorMessage = '創建房間失敗'
    if (err.message?.toLowerCase().includes('timeout')) {
      errorMessage = '創建房間超時，請重試'
    } else if (err.message?.toLowerCase().includes('network')) {
      errorMessage = '網路連線失敗，請檢查網路'
    } else if (err.message) {
      errorMessage = err.message
    }

    logError('VIEW_CREATE_ROOM', '創建房間失敗', {
      error: errorMessage
    })

    uiStore.showError(errorMessage)
    isSubmitting.value = false
    uiStore.setLoading(false)
  }
}

// 不再需要監聽房間創建事件，因為已改為直接 HTTP API 創建

// 組件卸載時清理
onMounted(() => {
  logInfo('VIEW_CREATE_ROOM', '頁面載入')
})

onUnmounted(() => {
  logInfo('VIEW_CREATE_ROOM', '離開頁面')
  // 重置狀態
  isSubmitting.value = false
  uiStore.setLoading(false)
  
  // 如果還在創建中但離開頁面，重置遊戲狀態
  if (isSubmitting.value && !gameStore.currentRoom) {
    console.log('🧹 頁面離開，重置遊戲狀態')
    gameStore.resetGame()
  }
})
</script>
