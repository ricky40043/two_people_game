<template>
  <div class="join-room-view min-h-screen flex items-center justify-center p-4">
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
          <div class="text-4xl mb-4">🚪</div>
          <h1 class="text-2xl font-bold text-white mb-2">加入房間</h1>
          <p class="text-white/70">輸入房間代碼和您的暱稱</p>
        </div>

        <form @submit.prevent="joinRoom" class="space-y-6">
          <!-- 房間代碼 -->
          <div>
            <label class="block text-white/90 font-medium mb-2">
              房間代碼 <span class="text-red-400">*</span>
            </label>
            <input
              v-model="form.roomId"
              type="text"
              class="input text-center text-2xl font-mono tracking-widest"
              placeholder="ABC123"
              required
              maxlength="6"
              :disabled="isSubmitting"
              @input="formatRoomId"
            />
            <div class="text-white/60 text-sm mt-1 text-center">
              請輸入 6 位房間代碼
            </div>
          </div>

          <!-- 玩家暱稱 -->
          <div>
            <label class="block text-white/90 font-medium mb-2">
              您的暱稱 <span class="text-red-400">*</span>
            </label>
            <input
              v-model="form.playerName"
              type="text"
              class="input"
              placeholder="請輸入您的暱稱"
              required
              maxlength="20"
              :disabled="isSubmitting"
            />
            <div class="text-white/60 text-sm mt-1">
              {{ form.playerName.length }}/20 個字符
            </div>
          </div>

          <!-- 提交按鈕 -->
          <button
            type="submit"
            :disabled="!canSubmit || isSubmitting"
            class="w-full btn btn-primary text-lg py-4"
            :class="{ 'opacity-50 cursor-not-allowed': !canSubmit || isSubmitting }"
          >
            <span v-if="isSubmitting" class="flex items-center justify-center">
              <div class="loading-spinner mr-2"></div>
              加入中...
            </span>
            <span v-else class="flex items-center justify-center">
              <span class="text-xl mr-2">🎮</span>
              加入遊戲
            </span>
          </button>
        </form>

        <!-- 或者分隔線 -->
        <div class="flex items-center my-6">
          <div class="flex-1 border-t border-white/20"></div>
          <span class="px-4 text-white/60 text-sm">或者</span>
          <div class="flex-1 border-t border-white/20"></div>
        </div>

        <!-- QR Code 掃描 -->
        <button
          @click="startQRScanner"
          :disabled="!canUseCamera || isSubmitting"
          class="w-full btn btn-outline py-4"
          :class="{ 'opacity-50 cursor-not-allowed': !canUseCamera || isSubmitting }"
        >
          <span class="flex items-center justify-center">
            <span class="text-xl mr-2">📱</span>
            掃描 QR Code
          </span>
        </button>

        <!-- 提示信息 -->
        <div class="mt-6">
          <div class="bg-blue-500/20 border border-blue-500/30 rounded-lg p-3">
            <div class="flex items-start">
              <span class="text-blue-400 text-lg mr-2 mt-0.5">💡</span>
              <div class="text-blue-100 text-sm text-left">
                <p class="font-medium mb-1">如何獲得房間代碼？</p>
                <ul class="space-y-1 text-xs">
                  <li>• 請向遊戲主持人索取 6 位房間代碼</li>
                  <li>• 或掃描主持人螢幕上的 QR Code</li>
                  <li>• 房間代碼僅包含大寫字母和數字</li>
                </ul>
              </div>
            </div>
          </div>
        </div>
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
          <button 
            @click="mockQRScan"
            class="flex-1 btn btn-primary"
          >
            模擬掃描
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useSocketStore } from '@/stores/socket'
import { useGameStore } from '@/stores/game'
import { useUIStore } from '@/stores/ui'
import { logInfo, logError, captureError } from '@/utils/logger'

const route = useRoute()
const router = useRouter()
const socketStore = useSocketStore()
const gameStore = useGameStore()
const uiStore = useUIStore()

// Props
const props = defineProps<{
  roomId?: string
}>()

// 表單數據
const form = ref({
  roomId: '',
  playerName: ''
})

const isSubmitting = ref(false)
const showQRScanner = ref(false)

// 計算屬性
const canSubmit = computed(() => {
  return (
    form.value.roomId.trim().length === 6 &&
    form.value.playerName.trim().length >= 2 &&
    !isSubmitting.value
  )
})

const canUseCamera = computed(() => {
  return 'mediaDevices' in navigator && 'getUserMedia' in navigator.mediaDevices
})

// 方法
const formatRoomId = () => {
  // 只允許大寫字母和數字
  let value = form.value.roomId.toUpperCase().replace(/[^A-Z0-9]/g, '')
  if (value.length > 6) {
    value = value.substring(0, 6)
  }
  form.value.roomId = value
}

const joinRoom = async () => {
  if (!canSubmit.value) return

  isSubmitting.value = true
  uiStore.setLoading(true, '正在加入房間...')

  try {
    logInfo('VIEW_JOIN_ROOM', '開始加入房間', {
      roomId: form.value.roomId,
      playerName: form.value.playerName
    })

    // 設置當前玩家
    gameStore.setPlayer({
      id: '', // 將由服務器分配
      name: form.value.playerName,
      roomId: form.value.roomId,
      score: 0,
      isHost: false,
      isConnected: true,
      lastActivity: new Date()
    })

    // 確保 WebSocket 連接
    if (!socketStore.isConnected) {
      logInfo('VIEW_JOIN_ROOM', '建立 WebSocket 連線中')
      socketStore.connect()
      
      // 等待連接建立
      await new Promise((resolve, reject) => {
        const checkConnection = () => {
          if (socketStore.isConnected) {
            resolve(true)
          } else {
            setTimeout(checkConnection, 100)
          }
        }
        checkConnection()
        
        // 5秒超時
        setTimeout(() => reject(new Error('連接超時')), 5000)
      })

      logInfo('VIEW_JOIN_ROOM', 'WebSocket 連線成功，準備送出 JOIN_ROOM')
    }

    // 發送加入房間請求
    socketStore.joinRoom(form.value.roomId, form.value.playerName)

    // 設置等待跳轉的超時
    setTimeout(() => {
      if (isSubmitting.value) {
        uiStore.showError('加入房間超時，請檢查房間代碼')
        isSubmitting.value = false
        uiStore.setLoading(false)
      }
    }, 10000) // 10秒超時
    
  } catch (error) {
    captureError('VIEW_JOIN_ROOM', error, {
      roomId: form.value.roomId,
      playerName: form.value.playerName
    })
    logError('VIEW_JOIN_ROOM', '加入房間失敗', error)
    uiStore.showError('加入房間失敗，請檢查房間代碼或網路連線')
    isSubmitting.value = false
    uiStore.setLoading(false)
  }
}

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

const mockQRScan = () => {
  // 模擬掃描到房間代碼
  const mockRoomId = 'ABC123'
  form.value.roomId = mockRoomId
  closeQRScanner()
  uiStore.showSuccess('掃描成功！已自動填入房間代碼')
}

// 監聽加入房間成功事件
const unwatchRoom = gameStore.$subscribe((_mutation, state) => {
  if (state.currentRoom && state.currentPlayer && !state.currentPlayer.isHost) {
    logInfo('VIEW_JOIN_ROOM', '玩家加入成功，即將跳轉遊戲畫面', {
      roomId: state.currentRoom.id,
      playerId: state.currentPlayer.id
    })
    
    // 停止 loading 狀態
    isSubmitting.value = false
    uiStore.setLoading(false)
    
    // 加入房間成功，直接跳轉到玩家遊戲畫面
    router.push(`/game/player/${state.currentRoom.id}`)
    unwatchRoom() // 取消監聽
  }
})

// 生命週期
onMounted(() => {
  logInfo('VIEW_JOIN_ROOM', '頁面載入', {
    prefilledRoomId: props.roomId || route.params.roomId
  })
  // 如果 URL 中有房間 ID，自動填入
  if (props.roomId) {
    form.value.roomId = props.roomId.toUpperCase()
  } else if (route.params.roomId) {
    form.value.roomId = (route.params.roomId as string).toUpperCase()
  }
})

onUnmounted(() => {
  logInfo('VIEW_JOIN_ROOM', '離開頁面')
  unwatchRoom()
})
</script>
