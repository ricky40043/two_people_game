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
        <div class="text-center mb-6">
          <div class="text-4xl mb-3">{{ roomExpired ? '⚠️' : '🚪' }}</div>
          <h1 class="text-2xl font-bold text-white mb-2">{{ roomExpired ? '房間不可用' : '加入房間' }}</h1>
          <p class="text-white/70">{{ roomExpired ? roomExpiredReason : '輸入房間代碼和您的暱稱' }}</p>
        </div>

        <!-- 房間過期 / 不存在 警告卡片 -->
        <div v-if="roomExpired" class="space-y-4 text-center">
          <div class="bg-red-500/20 border border-red-400/50 rounded-2xl p-6 text-red-100 mb-6">
            <p class="font-bold text-lg mb-2">掃描連結/房間已失效</p>
            <p class="text-xs opacity-90">此房間已被主持人關閉、過期關閉或遊戲已經開始。</p>
          </div>

          <router-link to="/" class="btn btn-primary w-full py-4 text-lg font-bold block text-center shadow-lg">
            🏠 返回主頁
          </router-link>

          <button @click="roomExpired = false; form.roomId = ''" class="btn btn-outline w-full py-3 text-sm">
            ✏️ 輸入其他房間代碼
          </button>
        </div>

        <!-- 正常加入表單 -->
        <form v-else @submit.prevent="joinRoom" class="space-y-6">
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
          :disabled="isSubmitting"
          class="w-full btn btn-outline py-4"
          :class="{ 'opacity-50 cursor-not-allowed': isSubmitting }"
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
        
        <div id="qr-reader" class="bg-gray-100 rounded-xl h-64 flex items-center justify-center mb-4 overflow-hidden">
          <!-- QR Code 掃描器容器 -->
        </div>
        
        <div class="flex space-x-4">
          <button 
            @click="closeQRScanner"
            class="flex-1 btn bg-gray-500 hover:bg-gray-600"
          >
            取消
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useSocketStore } from '@/stores/socket'
import { useGameStore } from '@/stores/game'
import { useUIStore } from '@/stores/ui'
import { apiService } from '@/services/api'
import { logInfo, logError, captureError } from '@/utils/logger'
import { Html5Qrcode } from 'html5-qrcode'

const route = useRoute()
const router = useRouter()
const socketStore = useSocketStore()
const gameStore = useGameStore()
const uiStore = useUIStore()

// Props
const props = defineProps<{
  roomId?: string
}>()

// 響應式數據
const isCheckingRoom = ref(false)
const roomExpired = ref(false)
const roomExpiredReason = ref('')

// 表單數據
const form = ref({
  roomId: '',
  playerName: '玩家A'
})

const isSubmitting = ref(false)
const showQRScanner = ref(false)
let html5QrCode: Html5Qrcode | null = null

// 監聽 QR 掃描器打開/關閉
watch(showQRScanner, async (newVal) => {
  if (newVal) {
    // 延遲一下讓 DOM 渲染完成
    setTimeout(async () => {
      try {
        html5QrCode = new Html5Qrcode('qr-reader')
        await html5QrCode.start(
          { facingMode: 'environment' },
          {
            fps: 10,
            qrbox: { width: 250, height: 250 }
          },
          (decodedText) => {
            // 掃描成功
            console.log('QR Code 掃描結果:', decodedText)
            // 解析 URL 或直接提取房間 ID
            let roomId = decodedText
            // 如果是 URL，嘗試提取房間 ID
            if (decodedText.includes('/join/')) {
              const match = decodedText.match(/\/join\/([A-Z0-9]+)/i)
              if (match) {
                roomId = match[1]
              }
            }
            // 如果是完整 URL，也嘗試從路徑提取
            if (decodedText.includes('?')) {
              const urlParams = new URL(decodedText)
              // 檢查路徑是否包含 roomId
              const pathMatch = urlParams.pathname.match(/\/join\/([A-Z0-9]+)/i)
              if (pathMatch) {
                roomId = pathMatch[1]
              }
            }
            
            form.value.roomId = roomId.toUpperCase()
            closeQRScanner()
            uiStore.showSuccess('掃描成功！已自動填入房間代碼')
          },
          () => {
            // 掃描錯誤，忽略
            void 0
          }
        )
      } catch (e) {
        console.error('啟動 QR 掃描器失敗:', e)
        uiStore.showError('無法啟動相機，請檢查權限')
      }
    }, 100)
  }
})

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

    // 監聽錯誤並立即重置
    const errorHandler = (event: MessageEvent) => {
      try {
        const message = JSON.parse(event.data)
        if (message.type === 'ERROR') {
          isSubmitting.value = false
          uiStore.setLoading(false)
          socketStore.socket?.removeEventListener('message', errorHandler)
        }
        if (message.type === 'PLAYER_JOINED') {
          socketStore.socket?.removeEventListener('message', errorHandler)
        }
      } catch (e) {}
    }
    socketStore.socket?.addEventListener('message', errorHandler)
    
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
  // 檢查是否在 HTTPS 環境或本地環境
  const isHttps = window.location.protocol === 'https:'
  const isLocalhost = window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1'
  
  if (!isHttps && !isLocalhost) {
    uiStore.showError('QR 掃描需要在 HTTPS 環境或本地 localhost 下使用')
    return
  }
  
  if (!canUseCamera.value) {
    uiStore.showError('您的設備不支援相機功能，請檢查瀏覽器權限設置')
    return
  }
  showQRScanner.value = true
}

const closeQRScanner = async () => {
  if (html5QrCode) {
    try {
      await html5QrCode.stop()
      html5QrCode = null
    } catch (e) {
      console.error('關閉 QR 掃描器失敗:', e)
    }
  }
  showQRScanner.value = false
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

const checkRoomStatus = async (targetRoomId: string) => {
  if (!targetRoomId || targetRoomId.length !== 6) return
  isCheckingRoom.value = true
  roomExpired.value = false

  try {
    logInfo('VIEW_JOIN_ROOM', '開頁即時檢查房間狀態', { targetRoomId })
    const room = await apiService.getRoom(targetRoomId)
    if (!room) {
      roomExpired.value = true
      roomExpiredReason.value = '此房間不存在或已過期關閉'
    } else if (room.status !== 'waiting') {
      roomExpired.value = true
      roomExpiredReason.value = room.status === 'finished' ? '該房間遊戲已經結束' : '該房間遊戲已經開始'
    }
  } catch (error) {
    roomExpired.value = true
    roomExpiredReason.value = '此房間不存在、已關閉或無效'
  } finally {
    isCheckingRoom.value = false
  }
}

// 生命週期
onMounted(() => {
  logInfo('VIEW_JOIN_ROOM', '頁面載入', {
    prefilledRoomId: props.roomId || route.params.roomId
  })
  // 如果 URL 中有房間 ID，自動填入並開頁即時檢查
  const targetId = props.roomId || (route.params.roomId as string)
  if (targetId) {
    form.value.roomId = targetId.toUpperCase()
    checkRoomStatus(form.value.roomId)
  }
})

onUnmounted(() => {
  logInfo('VIEW_JOIN_ROOM', '離開頁面')
  unwatchRoom()
})
</script>
