<template>
  <div class="game-player-view min-h-screen bg-gradient-to-br from-purple-600 via-blue-600 to-blue-700">
    <!-- 連接中 -->
    <div v-if="!gameStore.currentRoom || !gameStore.currentPlayer" class="h-screen flex items-center justify-center p-4">
      <div class="max-w-md w-full text-center">
        <div class="card card-body fade-in">
          <div class="text-4xl mb-4">🔗</div>
          <h1 class="text-2xl font-bold text-white mb-4">連接中...</h1>
          <p class="text-white/70 mb-6">正在加入房間，請稍候</p>
          
          <div class="bg-white/10 rounded-xl p-3">
            <div class="text-white/80 text-sm">房間代碼</div>
            <div class="text-lg font-mono font-bold text-white">{{ roomId }}</div>
          </div>
        </div>
      </div>
    </div>
    
    <!-- 等待開始 -->
    <div v-else-if="gameStore.gameState === 'waiting'" class="h-screen flex items-center justify-center p-4">
      <div class="max-w-md w-full text-center">
        <div class="card card-body fade-in">
          <div class="text-4xl mb-4">⏳</div>
          <h1 class="text-2xl font-bold text-white mb-4">等待遊戲開始</h1>
          <p class="text-white/70 mb-6">主持人正在準備題目...</p>
          
          <div class="space-y-3">
            <div class="bg-white/10 rounded-xl p-3">
              <div class="text-white/80 text-sm">您的暱稱</div>
              <div class="text-lg font-medium text-white">{{ gameStore.currentPlayer?.name }}</div>
            </div>
            
            <div class="bg-white/10 rounded-xl p-3">
              <div class="text-white/80 text-sm">房間代碼</div>
              <div class="text-lg font-mono font-bold text-white">{{ roomId }}</div>
            </div>
            
            <div class="bg-white/10 rounded-xl p-3">
              <div class="text-white/80 text-sm">當前分數</div>
              <div class="text-lg font-bold text-white">{{ gameStore.myScore }} 分</div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 答題介面 -->
    <div v-else-if="gameStore.gameState === 'playing'" class="h-screen flex flex-col">
      <!-- 頂部資訊 -->
      <div class="bg-black/30 backdrop-blur-sm p-4">
        <div class="flex items-center justify-between">
          <div class="flex items-center space-x-4">
            <div class="text-white">
              <span class="text-sm opacity-80">題目:</span>
              <span class="font-bold ml-1">{{ currentQuestionNumber }}/{{ gameStore.totalQuestions }}</span>
            </div>
            <div class="text-white">
              <span class="text-sm opacity-80">分數:</span>
              <span class="font-bold ml-1">{{ gameStore.myScore }}</span>
            </div>
            <div class="text-white">
              <span class="text-sm opacity-80">排名:</span>
              <span class="font-bold ml-1">#{{ gameStore.myRank }}</span>
            </div>
          </div>
          
          <!-- 倒數計時 -->
          <div class="flex items-center space-x-2">
            <div 
              class="w-12 h-12 rounded-full flex items-center justify-center font-bold text-lg"
              :class="timeLeftClass"
              style="background: rgba(255, 255, 255, 0.2)"
            >
              {{ gameStore.timeLeft }}
            </div>
          </div>
        </div>
        
        <!-- 進度條 -->
        <div class="mt-3 w-full bg-white/20 rounded-full h-1">
          <div 
            class="bg-white rounded-full h-1 transition-all duration-500"
            :style="{ width: gameStore.gameProgress + '%' }"
          ></div>
        </div>
      </div>

      <!-- 主角提示 -->
      <div v-if="isMyTurn" class="bg-yellow-400 text-black p-4 text-center">
        <div class="flex items-center justify-center">
          <span class="text-2xl mr-2">👑</span>
          <span class="font-bold text-lg">您是本題主角！其他人要猜您的選擇</span>
        </div>
      </div>
      
      <!-- 非主角提示 -->
      <div v-else-if="gameStore.gameState === 'playing'" class="bg-blue-500 text-white p-4 text-center">
        <div class="flex items-center justify-center">
          <span class="text-2xl mr-2">🤔</span>
          <span class="font-bold text-lg">猜猜主角會選什麼？主角是：{{ getCurrentHostName() }}</span>
        </div>
      </div>

      <!-- 題目顯示 -->
      <div class="flex-1 flex flex-col justify-center p-4">
        <div v-if="gameStore.currentQuestion" class="max-w-2xl mx-auto w-full">
          <!-- 題目文字 -->
          <div class="bg-white/95 rounded-3xl p-6 mb-6 text-center shadow-2xl">
            <div class="text-gray-600 text-sm mb-2">
              第 {{ currentQuestionNumber }} 題
            </div>
            <h2 class="text-xl md:text-2xl font-bold text-gray-800 leading-tight">
              {{ gameStore.currentQuestion.questionText }}
            </h2>
          </div>

          <!-- 答案選項 -->
          <div class="grid grid-cols-1 gap-4">
            <button
              v-for="option in questionOptions"
              :key="option.key"
              @click="selectAnswer(option.key)"
              :disabled="hasAnswered || gameStore.timeLeft <= 0"
              class="answer-button"
              :class="[
                option.colorClass,
                {
                  'selected': selectedAnswer === option.key,
                  'disabled': hasAnswered || gameStore.timeLeft <= 0,
                  'correct': false, // 「2種人」遊戲沒有標準正確答案
                  'incorrect': false
                }
              ]"
            >
              <div class="flex items-center justify-center">
                <div class="w-8 h-8 bg-white/30 rounded-lg flex items-center justify-center mr-3">
                  <span class="font-bold text-lg">{{ option.key }}</span>
                </div>
                <span class="font-semibold text-lg flex-1 text-left">{{ option.text }}</span>
                
                <!-- 答題狀態圖示 -->
                <div v-if="hasAnswered && selectedAnswer === option.key" class="ml-3">
                  <div v-if="showResult">
                    <span class="text-2xl">✓</span>
                  </div>
                  <div v-else class="text-2xl">✓</div>
                </div>
              </div>
            </button>
          </div>

          <!-- 答題狀態 -->
          <div class="mt-6 text-center">
            <div v-if="!hasAnswered && gameStore.timeLeft > 0" class="text-white/80">
              選擇您的答案
            </div>
            <div v-else-if="hasAnswered && !showResult" class="text-green-400 font-semibold">
              ✓ 已提交答案，等待其他玩家...
              <div class="text-sm text-white/60 mt-1">
                已答題: {{ answeredPlayersCount }}/{{ gameStore.playerCount }}
              </div>
            </div>
            <div v-else-if="gameStore.timeLeft <= 0" class="text-red-400 font-semibold">
              ⏰ 時間到！
            </div>
          </div>
        </div>

        <!-- 沒有題目 -->
        <div v-else class="text-center">
          <div class="text-4xl mb-4">📝</div>
          <h2 class="text-2xl font-bold text-white mb-2">準備題目中...</h2>
          <p class="text-white/70">請稍候</p>
        </div>
      </div>
    </div>

    <!-- 題目結果 -->
    <div v-else-if="gameStore.gameState === 'show_result'" class="h-screen flex flex-col justify-center p-4">
      <div class="max-w-2xl mx-auto w-full text-center">
        <div class="bg-white rounded-3xl p-8 shadow-2xl fade-in">
          <h2 class="text-2xl font-bold text-gray-800 mb-6">
            第 {{ currentQuestionNumber }} 題結果
          </h2>
          
          <!-- 答題結果 -->
          <div class="mb-6">
            <div class="text-lg text-gray-600 mb-2">主角選擇:</div>
            <div class="text-3xl font-bold text-blue-600 mb-4">
              等待結果公布...
            </div>
            
            <!-- 個人結果 -->
            <div class="bg-gray-100 rounded-xl p-4 mb-4">
              <div class="text-lg font-semibold mb-2">您的選擇</div>
              <div class="grid grid-cols-2 gap-4">
                <div>
                  <div class="text-2xl font-bold text-blue-600">
                    {{ selectedAnswer ? `選項 ${selectedAnswer}` : '未作答' }}
                  </div>
                  <div class="text-sm text-gray-600">{{ isMyTurn ? '您是主角' : '您的猜測' }}</div>
                </div>
                <div>
                  <div class="text-2xl font-bold text-blue-600">+{{ scoreGained }}</div>
                  <div class="text-sm text-gray-600">獲得分數</div>
                </div>
              </div>
            </div>
            
            <!-- 說明 -->
            <div v-if="gameStore.currentQuestion?.explanation" class="text-gray-600 text-sm">
              {{ gameStore.currentQuestion.explanation }}
            </div>
          </div>

          <!-- 當前排名 -->
          <div class="bg-gradient-to-r from-purple-100 to-blue-100 rounded-xl p-4">
            <div class="text-lg font-semibold text-gray-800 mb-2">當前排名</div>
            <div class="flex items-center justify-center space-x-6">
              <div class="text-center">
                <div class="text-3xl font-bold text-purple-600">#{{ gameStore.myRank }}</div>
                <div class="text-sm text-gray-600">排名</div>
              </div>
              <div class="text-center">
                <div class="text-3xl font-bold text-blue-600">{{ gameStore.myScore }}</div>
                <div class="text-sm text-gray-600">總分</div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 遊戲結束 -->
    <div v-else-if="gameStore.gameState === 'finished'" class="h-screen flex flex-col justify-center p-4">
      <div class="max-w-md mx-auto w-full text-center">
        <div class="card card-body fade-in">
          <div class="text-4xl mb-4">🏆</div>
          <h1 class="text-2xl font-bold text-white mb-4">遊戲結束！</h1>
          
          <!-- 個人成績 -->
          <div class="bg-white/10 rounded-xl p-4 mb-6">
            <div class="text-white font-semibold mb-3">您的最終成績</div>
            <div class="grid grid-cols-2 gap-4">
              <div class="text-center">
                <div class="text-2xl font-bold text-yellow-400">#{{ gameStore.myRank }}</div>
                <div class="text-white/80 text-sm">最終排名</div>
              </div>
              <div class="text-center">
                <div class="text-2xl font-bold text-blue-400">{{ gameStore.myScore }}</div>
                <div class="text-white/80 text-sm">總分</div>
              </div>
            </div>
          </div>

          <div class="space-y-3">
            <router-link 
              :to="`/results/${roomId}`"
              class="block btn btn-primary"
            >
              查看詳細結果
            </router-link>
            
            <router-link 
              to="/"
              class="block btn btn-outline"
            >
              返回主頁
            </router-link>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useGameStore } from '@/stores/game'
import { useSocketStore } from '@/stores/socket'
import { useUIStore } from '@/stores/ui'

const route = useRoute()
const router = useRouter()
const gameStore = useGameStore()
const socketStore = useSocketStore()
const uiStore = useUIStore()

// Props
const props = defineProps<{
  roomId: string
}>()

// 響應式數據
const selectedAnswer = ref<string>('')
const hasAnswered = ref(false)
const showResult = ref(false)
const scoreGained = ref(0)

// 計算屬性
const roomId = computed(() => props.roomId || route.params.roomId as string)
const currentQuestionNumber = computed(() => gameStore.currentQuestionIndex + 1)

const questionOptions = computed(() => {
  const question = gameStore.currentQuestion
  if (!question) return []
  
  // 「2種人」遊戲只有兩個選項
  return [
    { key: 'A', text: question.optionA, colorClass: 'answer-a' },
    { key: 'B', text: question.optionB, colorClass: 'answer-b' }
  ]
})

const timeLeftClass = computed(() => {
  const timeLeft = gameStore.timeLeft
  if (timeLeft <= 5) return 'text-red-600 bg-red-100'
  if (timeLeft <= 10) return 'text-orange-600 bg-orange-100'
  return 'text-green-600 bg-green-100'
})

const isMyTurn = computed(() => {
  return gameStore.currentHost === gameStore.currentPlayer?.id
})

const answeredPlayersCount = computed(() => {
  // 使用 GameStore 中的計算屬性
  return gameStore.answeredPlayersCount
})

// 方法
const getCurrentHostName = () => {
  if (!gameStore.currentHost || !gameStore.currentRoom?.players) return '未知'
  const hostPlayer = Object.values(gameStore.currentRoom.players).find(p => p.id === gameStore.currentHost)
  return hostPlayer?.name || '未知'
}
const selectAnswer = (answer: string) => {
  if (hasAnswered.value || gameStore.timeLeft <= 0) return
  
  selectedAnswer.value = answer
  hasAnswered.value = true
  
  // 計算答題時間
  const questionTimeLimit = gameStore.currentRoom?.questionTimeLimit || 30
  const timeUsed = questionTimeLimit - gameStore.timeLeft
  
  // 發送答案到後端（分數由後端計算）
  socketStore.submitAnswer(
    roomId.value,
    gameStore.currentQuestion?.id || 0,
    answer,
    timeUsed
  )
  
  // 給用戶回饋
  if (isMyTurn.value) {
    uiStore.showSuccess('已選擇！其他人正在猜測您的選擇...')
  } else {
    uiStore.showSuccess('已提交猜測！等待結果...')
  }
  
  console.log(`✏️ 提交答案: ${answer}, 耗時: ${timeUsed}秒, 是否主角: ${isMyTurn.value}`)
}

// getCorrectAnswerText 方法已移除，「2種人」遊戲不需要標準正確答案

const resetQuestionState = () => {
  selectedAnswer.value = ''
  hasAnswered.value = false
  showResult.value = false
  scoreGained.value = 0
}

// 追蹤當前題目編號，用於判斷是否為新題目
const currentQuestionId = ref<number>(0)

// 監聽遊戲狀態變化
const unwatchGameState = gameStore.$subscribe((_mutation, state) => {
  if (state.gameState === 'playing') {
    // 檢查是否是新題目（題目ID變化）
    const newQuestionId = gameStore.currentQuestion?.id || 0
    if (newQuestionId !== currentQuestionId.value) {
      console.log(`🔄 新題目開始: ${currentQuestionId.value} → ${newQuestionId}`)
      currentQuestionId.value = newQuestionId
      resetQuestionState()
    }
  } else if (state.gameState === 'show_result') {
    showResult.value = true
    updateScoreGained()
  } else if (state.gameState === 'finished') {
    router.push(`/results/${roomId.value}`)
  }
})

// 更新分數顯示
const updateScoreGained = () => {
  // 從最新的分數陣列中找到當前玩家的得分
  const currentPlayerId = gameStore.currentPlayer?.id
  if (currentPlayerId && gameStore.scores.length > 0) {
    const playerScore = gameStore.scores.find(score => score.playerId === currentPlayerId)
    if (playerScore && playerScore.scoreGained !== undefined) {
      scoreGained.value = playerScore.scoreGained
      console.log(`💰 更新得分顯示: ${scoreGained.value} 分`)
    }
  }
}

// 生命週期
onMounted(() => {
  // 確保不是主持人
  if (gameStore.isHost) {
    uiStore.showError('主持人請使用主持人視角')
    router.push(`/game/host/${roomId.value}`)
    return
  }
  
  // 如果沒有房間或玩家信息，但有 roomId，嘗試連接 WebSocket
  if (!gameStore.currentRoom || !gameStore.currentPlayer) {
    if (roomId.value) {
      console.log('🔗 嘗試連接 WebSocket 來獲取房間信息...')
      if (!socketStore.isConnected) {
        socketStore.connect()
      }
      
      // 給一些時間讓 WebSocket 連接和房間信息同步
      setTimeout(() => {
        if (!gameStore.currentRoom || !gameStore.currentPlayer) {
          console.warn('⚠️ 無法獲取房間信息，可能需要重新加入')
          // 不強制跳轉，讓用戶看到連接中的畫面
        }
      }, 3000)
    } else {
      uiStore.showError('遊戲資訊不存在，請重新加入')
      router.push('/')
      return
    }
  }
})

onUnmounted(() => {
  unwatchGameState()
  
  // 頁面離開時只記錄日誌，不自動清理
  if (window.debugLogger) {
    window.debugLogger.info('CLEANUP', 'GamePlayerView 組件卸載', {
      gameState: gameStore.gameState,
      note: '不自動清理，等待用戶操作'
    })
  }
})
</script>

<style scoped>
.answer-button {
  @apply p-4 rounded-2xl text-white font-bold transition-all duration-300 transform;
}

.answer-button:not(.disabled):hover {
  @apply scale-105;
}

.answer-button.selected {
  @apply ring-4 ring-white/50 scale-105;
}

.answer-button.disabled {
  @apply opacity-60 cursor-not-allowed;
}

.answer-button.correct {
  @apply ring-4 ring-green-400;
}

.answer-button.incorrect {
  @apply ring-4 ring-red-400;
}

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

.fade-in {
  animation: fadeIn 0.6s ease-out;
}
</style>
