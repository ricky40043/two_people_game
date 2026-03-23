<template>
  <div class="lobby-view min-h-screen bg-gradient-to-br from-purple-600 via-blue-600 to-blue-700">
    <!-- 主持人視角 -->
    <div v-if="gameStore.isHost" class="host-lobby">
      <!-- 頂部標題 -->
      <div class="bg-black/20 backdrop-blur-sm p-4">
        <div class="container mx-auto">
          <div class="flex items-center justify-between">
            <div class="flex items-center space-x-4">
              <h1 class="text-2xl font-bold text-white">🎮 遊戲大廳</h1>
              <div class="bg-white/20 px-3 py-1 rounded-full">
                <span class="text-white font-mono text-lg">{{ roomId }}</span>
              </div>
            </div>
            <div class="flex items-center space-x-4">
              <div class="text-white/80">
                <span class="text-sm">玩家數量:</span>
                <span class="font-bold ml-1">{{ gameStore.playerCount }}/20</span>
              </div>
              <button
                @click="leaveRoom"
                class="btn bg-red-500 hover:bg-red-600 text-sm"
              >
                離開房間
              </button>
            </div>
          </div>
        </div>
      </div>

      <div class="container mx-auto p-6">
        <div class="grid md:grid-cols-2 gap-8">
          <!-- QR Code 區域 -->
          <div class="space-y-6">
            <div class="card card-body text-center">
              <h2 class="text-xl font-bold text-white mb-4">🔗 邀請玩家加入</h2>
              
              <!-- QR Code -->
              <QRCodeDisplay
                :data="joinUrl"
                :size="192"
                title="🎮 加入遊戲"
                :description="`房間代碼: ${roomId}`"
                :show-actions="false"
                @generated="onQRGenerated"
              />
              
              <!-- 房間資訊 -->
              <div class="space-y-3">
                <div class="bg-white/10 rounded-xl p-4">
                  <div class="text-white/80 text-sm mb-1">房間代碼</div>
                  <div class="text-2xl font-mono font-bold text-white tracking-widest">
                    {{ roomId }}
                  </div>
                  <button
                    @click="copyRoomId"
                    class="mt-2 text-xs btn btn-outline py-1 px-3"
                  >
                    📋 複製代碼
                  </button>
                </div>
                
                <div class="bg-white/10 rounded-xl p-4">
                  <div class="text-white/80 text-sm mb-1">加入網址</div>
                  <div class="text-sm text-white break-all">
                    {{ joinUrl }}
                  </div>
                  <button
                    @click="copyJoinUrl"
                    class="mt-2 text-xs btn btn-outline py-1 px-3"
                  >
                    🔗 複製網址
                  </button>
                </div>
              </div>

              <!-- 分享按鈕 -->
              <div class="flex space-x-2 mt-4">
                <button
                  @click="shareRoom"
                  v-if="canShare"
                  class="flex-1 btn btn-primary text-sm"
                >
                  📱 分享
                </button>
                <button
                  @click="downloadQR"
                  class="flex-1 btn btn-outline text-sm"
                >
                  💾 下載 QR
                </button>
              </div>
            </div>

            <!-- 遊戲設定 -->
            <div class="card card-body">
              <h3 class="text-lg font-bold text-white mb-3">⚙️ 遊戲設定</h3>
              <div class="space-y-3 text-sm">
                <div class="flex justify-between text-white/80">
                  <span>主持人：</span>
                  <span class="font-medium">{{ gameStore.currentRoom?.hostName }}</span>
                </div>
                <div class="flex justify-between text-white/80">
                  <span>題目數量：</span>
                  <span class="font-medium">{{ gameStore.currentRoom?.totalQuestions }} 題</span>
                </div>
                <div class="flex justify-between text-white/80">
                  <span>答題時間：</span>
                  <span class="font-medium">{{ gameStore.currentRoom?.questionTimeLimit }} 秒</span>
                </div>
                <div class="flex justify-between text-white/80">
                  <span>預計時長：</span>
                  <span class="font-medium">{{ estimatedDuration }} 分鐘</span>
                </div>
              </div>
            </div>
          </div>

          <!-- 玩家列表 -->
          <div class="space-y-6">
            <div class="card card-body">
              <div class="flex items-center justify-between mb-4">
                <h3 class="text-lg font-bold text-white">👥 玩家列表</h3>
                <div class="text-white/80 text-sm">
                  {{ gameStore.playerCount }} 人已加入
                </div>
              </div>

              <div class="player-list max-h-80 overflow-y-auto">
                <div
                  v-for="player in playerList"
                  :key="player.id"
                  class="player-item"
                >
                  <div class="flex items-center space-x-3">
                    <PlayerAvatar
                      :name="player.name"
                      :is-online="player.isConnected"
                      :is-host="player.isHost"
                      size="md"
                    />
                    <div>
                      <div class="text-white font-medium">{{ player.name }}</div>
                      <div class="text-white/60 text-xs flex items-center">
                        <span :class="[
                          'w-2 h-2 rounded-full mr-1',
                          player.isConnected ? 'bg-green-400' : 'bg-red-400'
                        ]"></span>
                        {{ player.isConnected ? '在線' : '離線' }}
                      </div>
                    </div>
                  </div>
                  <div class="flex items-center space-x-2">
                    <span v-if="player.isHost" class="text-xs bg-yellow-500 text-black px-2 py-1 rounded-full font-bold">
                      主持人
                    </span>
                    <span class="text-white/60 text-xs">
                      {{ formatJoinTime(player.lastActivity) }}
                    </span>
                  </div>
                </div>

                <!-- 空狀態 -->
                <div v-if="gameStore.playerCount === 0" class="text-center py-8">
                  <div class="text-white/60 text-sm">還沒有玩家加入</div>
                  <div class="text-white/40 text-xs mt-1">分享房間代碼邀請朋友吧！</div>
                </div>
              </div>
            </div>

            <!-- 開始遊戲按鈕 -->
            <div class="space-y-4">
              <button
                @click="startGame"
                :disabled="!canStartGame"
                class="w-full btn text-xl py-6"
                :class="canStartGame ? 'btn-success' : 'bg-gray-500 cursor-not-allowed'"
              >
                <span v-if="canStartGame" class="flex items-center justify-center">
                  <span class="text-2xl mr-3">🚀</span>
                  開始遊戲 ({{ gameStore.playerCount }} 人)
                </span>
                <span v-else class="flex items-center justify-center">
                  <span class="text-2xl mr-3">⏳</span>
                  等待更多玩家 (至少需要 2 人)
                </span>
              </button>

              <!-- 測試按鈕 -->
              <button
                v-if="isDev"
                @click="startGameForce"
                class="w-full btn btn-warning text-sm py-2"
              >
                🧪 強制開始 (測試用)
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 玩家視角 -->
    <div v-else class="player-lobby">
      <div class="min-h-screen flex items-center justify-center p-4">
        <div class="max-w-md w-full">
          <div class="card card-body text-center fade-in">
            <div class="text-4xl mb-4">⏳</div>
            <h1 class="text-2xl font-bold text-white mb-4">等待遊戲開始</h1>
            
            <div class="space-y-4 mb-6">
              <div class="bg-white/10 rounded-xl p-4">
                <div class="text-white/80 text-sm">房間代碼</div>
                <div class="text-xl font-mono font-bold text-white">{{ roomId }}</div>
              </div>
              
              <div class="bg-white/10 rounded-xl p-4">
                <div class="text-white/80 text-sm">您的暱稱</div>
                <div class="text-lg font-medium text-white">{{ gameStore.currentPlayer?.name }}</div>
              </div>
              
              <div class="bg-white/10 rounded-xl p-4">
                <div class="text-white/80 text-sm">當前玩家</div>
                <div class="text-lg font-bold text-white">{{ gameStore.playerCount }} 人</div>
              </div>
            </div>

            <!-- 玩家列表 -->
            <div class="text-left mb-6">
              <h3 class="text-white font-medium mb-3 text-center">👥 其他玩家</h3>
              <div class="space-y-2 max-h-40 overflow-y-auto">
                <div
                  v-for="player in otherPlayers"
                  :key="player.id"
                  class="flex items-center space-x-3 p-2 bg-white/5 rounded-lg"
                >
                  <PlayerAvatar
                    :name="player.name"
                    :is-online="player.isConnected"
                    :is-host="player.isHost"
                    size="sm"
                  />
                  <span class="text-white/90 text-sm">{{ player.name }}</span>
                  <span v-if="player.isHost" class="text-xs bg-yellow-500 text-black px-1 py-0.5 rounded font-bold">
                    主持人
                  </span>
                </div>
              </div>
            </div>

            <div class="text-white/60 text-sm mb-4">
              等待主持人開始遊戲...
            </div>

            <button
              @click="leaveRoom"
              class="btn btn-outline w-full"
            >
              離開房間
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useGameStore } from '@/stores/game'
import { useSocketStore } from '@/stores/socket'
import { useUIStore } from '@/stores/ui'
import { useGameLogic } from '@/composables/useGameLogic'
import QRCodeDisplay from '@/components/QRCodeDisplay.vue'
import PlayerAvatar from '@/components/PlayerAvatar.vue'
import { logInfo, logWarn, logError, logDebug, captureError } from '@/utils/logger'

const route = useRoute()
const router = useRouter()
const gameStore = useGameStore()
const socketStore = useSocketStore()
const uiStore = useUIStore()
const gameLogic = useGameLogic()

// Props
const props = defineProps<{
  roomId: string
}>()

// 響應式數據

// 計算屬性
const roomId = computed(() => props.roomId || route.params.roomId as string)

const joinUrl = computed(() => {
  // 總是使用前端當前的 Origin，確保連結與使用者瀏覽器一致
  // 例如：如果用 localhost 開，就顯示 localhost；用 IP 開，就顯示 IP
  const baseUrl = window.location.origin
  return `${baseUrl}/join/${roomId.value}`
})

const playerList = computed(() => {
  if (!gameStore.currentRoom?.players) return []
  return Object.values(gameStore.currentRoom.players).filter(p => !p.isHost)
})

const otherPlayers = computed(() => {
  return playerList.value.filter(p => p.id !== gameStore.currentPlayer?.id)
})

const canStartGame = computed(() => {
  return gameStore.playerCount >= 2 && gameStore.isHost
})

const estimatedDuration = computed(() => {
  const room = gameStore.currentRoom
  if (!room) return 0
  const questionTime = room.totalQuestions * room.questionTimeLimit
  const resultTime = room.totalQuestions * 10
  return Math.round((questionTime + resultTime + 60) / 60)
})

const canShare = computed(() => {
  return 'share' in navigator
})

const isDev = computed(() => {
  return import.meta.env.DEV
})

// 方法
const onQRGenerated = (_canvas: HTMLCanvasElement) => {
  logDebug('VIEW_LOBBY', 'QR Code 生成成功')
}

const copyToClipboard = async (text: string): Promise<boolean> => {
  try {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(text)
      return true
    }
    const textarea = document.createElement('textarea')
    textarea.value = text
    textarea.style.position = 'fixed'
    textarea.style.opacity = '0'
    document.body.appendChild(textarea)
    textarea.select()
    const success = document.execCommand('copy')
    document.body.removeChild(textarea)
    return success
  } catch {
    return false
  }
}

const copyRoomId = async () => {
  const success = await copyToClipboard(roomId.value)
  if (success) {
    logInfo('VIEW_LOBBY', '複製房間代碼成功', { roomId: roomId.value })
    uiStore.showSuccess('房間代碼已複製')
  } else {
    captureError('VIEW_LOBBY', new Error('Clipboard copy failed'), { action: 'copyRoomId' })
    uiStore.showError('複製失敗，請手動複製')
  }
}

const copyJoinUrl = async () => {
  const success = await copyToClipboard(joinUrl.value)
  if (success) {
    logInfo('VIEW_LOBBY', '複製房間網址成功', { joinUrl: joinUrl.value })
    uiStore.showSuccess('加入網址已複製')
  } else {
    captureError('VIEW_LOBBY', new Error('Clipboard copy failed'), { action: 'copyJoinUrl' })
    uiStore.showError('複製失敗，請手動複製')
  }
}

const shareRoom = async () => {
  if (!canShare.value) return
  
  try {
    await navigator.share({
      title: '🎮 Ricky 遊戲小舖邀請',
      text: `加入我的遊戲房間！房間代碼: ${roomId.value}`,
      url: joinUrl.value
    })
    logInfo('VIEW_LOBBY', '分享房間成功', { roomId: roomId.value })
  } catch (error) {
    captureError('VIEW_LOBBY', error, { action: 'shareRoom' })
    logError('VIEW_LOBBY', '分享房間失敗', error)
  }
}

const downloadQR = () => {
  // TODO: 實現下載功能
  uiStore.showSuccess('下載功能開發中')
}

const startGame = async () => {
  if (!canStartGame.value) return
  
  uiStore.setLoading(true, '正在載入題目...')
  
  try {
    logInfo('VIEW_LOBBY', '主持人觸發開始遊戲', { roomId: roomId.value })
    // 載入題目
    const room = gameStore.currentRoom
    if (!room) throw new Error('房間資訊不存在')
    
    await gameLogic.loadQuestions(room.totalQuestions)
    
    // 發送開始遊戲訊息
    socketStore.startGame(roomId.value)
    
    // 如果是主持人，開始第一題
    if (gameStore.isHost) {
      setTimeout(() => {
        gameLogic.startQuestion(0)
      }, 2000) // 給其他玩家一點準備時間
    }
    
  } catch (error) {
    captureError('VIEW_LOBBY', error, { action: 'startGame' })
    logError('VIEW_LOBBY', '開始遊戲失敗', error)
    uiStore.showError('載入題目失敗，請稍後重試')
  } finally {
    uiStore.setLoading(false)
  }
}

const startGameForce = async () => {
  uiStore.setLoading(true, '強制開始遊戲...')
  
  try {
    logWarn('VIEW_LOBBY', '觸發強制開始遊戲', { roomId: roomId.value })
    // 載入備用題目
    await gameLogic.loadQuestions(5) // 強制模式只用5題
    socketStore.startGame(roomId.value)
    
    if (gameStore.isHost) {
      setTimeout(() => {
        gameLogic.startQuestion(0)
      }, 1000)
    }
    
  } catch (error) {
    captureError('VIEW_LOBBY', error, { action: 'startGameForce' })
    logError('VIEW_LOBBY', '強制開始失敗', error)
    uiStore.showError('強制開始失敗')
  } finally {
    uiStore.setLoading(false)
  }
}

const leaveRoom = () => {
  logInfo('VIEW_LOBBY', '玩家離開房間', { roomId: roomId.value })
  socketStore.leaveRoom()
  gameStore.resetGame()
  router.push('/')
}

const formatJoinTime = (date: Date) => {
  const now = new Date()
  const diff = Math.floor((now.getTime() - date.getTime()) / 1000)
  
  if (diff < 60) return '剛剛加入'
  if (diff < 3600) return `${Math.floor(diff / 60)} 分鐘前`
  return `${Math.floor(diff / 3600)} 小時前`
}

// 監聽遊戲開始事件
const unwatchGameState = gameStore.$subscribe((_mutation, state) => {
  if (state.gameState === 'playing') {
    // 遊戲開始，跳轉到對應的遊戲頁面
    if (gameStore.isHost) {
      router.push(`/game/host/${roomId.value}`)
    } else {
      router.push(`/game/player/${roomId.value}`)
    }
    unwatchGameState()
  }
})

// 生命週期
onMounted(() => {
  logInfo('VIEW_LOBBY', '頁面載入', { roomId: roomId.value, isHost: gameStore.isHost })

  // 已有房間資訊 → 正常顯示
  if (gameStore.currentRoom) return

  // 沒有房間信息 → 檢查 session，等待 App.vue 自動重連
  const savedSession = localStorage.getItem('ricky_game_session')
  if (savedSession) {
    try {
      const session = JSON.parse(savedSession)
      if (session.roomId === roomId.value) {
        logInfo('VIEW_LOBBY', '等待 App.vue 自動重連...', session)
        // App.vue 已呼叫 connect() + CHECK_ROOM → 自動 REJOIN
        setTimeout(() => {
          if (!gameStore.currentRoom) {
            uiStore.showError('重連失敗，請重新加入')
            router.push('/')
          }
        }, 5000)
        return
      }
    } catch (e) {
      console.error('解析 session 失敗', e)
    }
  }
  uiStore.showError('房間資訊不存在，請重新加入')
  router.push('/')
})

onUnmounted(() => {
  logInfo('VIEW_LOBBY', '離開頁面', { roomId: roomId.value })
  unwatchGameState()
})
</script>

<style scoped>
.player-list {
  scrollbar-width: thin;
  scrollbar-color: rgba(255, 255, 255, 0.3) transparent;
}

.player-list::-webkit-scrollbar {
  width: 4px;
}

.player-list::-webkit-scrollbar-track {
  background: transparent;
}

.player-list::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.3);
  border-radius: 2px;
}
</style>
