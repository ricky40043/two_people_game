<template>
  <div class="game-host-view min-h-screen bg-gradient-to-br from-purple-600 via-blue-600 to-blue-700">
    <!-- 遊戲進行中 -->
    <div v-if="gameStore.gameState === 'playing'" class="h-screen flex flex-col">
      <!-- 頂部狀態欄 -->
      <div class="bg-black/30 backdrop-blur-sm p-4">
        <div class="container mx-auto">
          <div class="flex items-center justify-between">
            <div class="flex items-center space-x-6">
              <div class="text-white">
                <span class="text-sm opacity-80">房間:</span>
                <span class="font-mono font-bold ml-1">{{ roomId }}</span>
              </div>
              <div class="text-white">
                <span class="text-sm opacity-80">題目:</span>
                <span class="font-bold ml-1">{{ currentQuestionNumber }}/{{ gameStore.totalQuestions }}</span>
              </div>
              <div class="text-white">
                <span class="text-sm opacity-80">玩家:</span>
                <span class="font-bold ml-1">{{ gameStore.playerCount }}</span>
              </div>
            </div>
            
            <div class="flex items-center space-x-4">
              <!-- 遊戲進度 -->
              <div class="w-48 bg-white/20 rounded-full h-2">
                <div 
                  class="bg-white rounded-full h-2 transition-all duration-500"
                  :style="{ width: gameStore.gameProgress + '%' }"
                ></div>
              </div>
              
              <!-- 結束遊戲按鈕 -->
              <button
                @click="showEndGameConfirm = true"
                class="btn bg-red-500 hover:bg-red-600 text-sm"
              >
                結束遊戲
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- 主要遊戲區域 -->
      <div class="flex-1 flex">
        <!-- 左側 - 題目顯示 -->
        <div class="flex-1 flex flex-col justify-center p-8">
          <div class="max-w-4xl mx-auto w-full">
            <!-- 當前主角提示 -->
            <div v-if="currentHostPlayer" class="text-center mb-8">
              <div class="inline-flex items-center bg-yellow-400 text-black px-6 py-3 rounded-full">
                <span class="text-2xl mr-2">👑</span>
                <span class="font-bold text-lg">主角: {{ currentHostPlayer.name }}</span>
              </div>
              <p class="text-white/80 mt-2">請大聲朗讀題目給其他玩家聽</p>
            </div>

            <!-- 題目卡片 -->
            <div v-if="gameStore.currentQuestion" class="question-card fade-in">
              <div class="text-center mb-8">
                <div class="text-gray-600 text-lg mb-2">
                  第 {{ currentQuestionNumber }} 題
                </div>
                <h2 class="text-3xl font-bold text-gray-800 leading-tight">
                  {{ gameStore.currentQuestion.questionText }}
                </h2>
              </div>

              <!-- 答案選項 -->
              <div class="grid grid-cols-2 gap-4">
                <div 
                  v-for="option in questionOptions" 
                  :key="option.key"
                  class="answer-option"
                  :class="option.colorClass"
                >
                  <div class="flex items-center">
                    <div class="w-12 h-12 bg-white/20 rounded-xl flex items-center justify-center mr-4">
                      <span class="text-2xl font-bold">{{ option.key }}</span>
                    </div>
                    <span class="text-xl font-semibold flex-1">{{ option.text }}</span>
                  </div>
                </div>
              </div>

              <!-- 倒數計時器 -->
              <div class="mt-8 text-center">
                <div class="inline-flex items-center bg-white rounded-full px-8 py-4">
                  <span class="text-3xl font-bold mr-4" :class="timeLeftClass">
                    {{ gameStore.timeLeft }}
                  </span>
                  <span class="text-gray-600 text-lg">秒</span>
                </div>
              </div>
            </div>

            <!-- 等待下一題 -->
            <div v-else class="text-center py-16">
              <div class="text-4xl mb-4">⏳</div>
              <h2 class="text-2xl font-bold text-white mb-2">準備下一題...</h2>
              <p class="text-white/70">正在載入題目</p>
            </div>
          </div>
        </div>

        <!-- 右側 - 玩家狀態 -->
        <div class="w-80 bg-black/20 backdrop-blur-sm p-6">
          <div class="mb-6">
            <h3 class="text-white font-bold text-lg mb-4">🏆 即時排行榜</h3>
            <div class="space-y-3 max-h-80 overflow-y-auto">
              <div
                v-for="(scoreInfo, index) in gameStore.sortedScores"
                :key="scoreInfo.playerId"
                class="flex items-center justify-between p-3 bg-white/10 rounded-xl"
                :class="{
                  'bg-yellow-400/30 border-2 border-yellow-400': index === 0,
                  'bg-gray-400/30 border border-gray-400': index === 1,
                  'bg-orange-400/30 border border-orange-400': index === 2
                }"
              >
                <div class="flex items-center space-x-3">
                  <div class="flex items-center justify-center w-8 h-8 rounded-full bg-white/20 text-white font-bold">
                    {{ index + 1 }}
                  </div>
                  <PlayerAvatar
                    :name="scoreInfo.playerName"
                    :is-online="isPlayerOnline(scoreInfo.playerId)"
                    :is-host="scoreInfo.playerId === gameStore.currentRoom?.hostId"
                    size="sm"
                  />
                  <div>
                    <div class="text-white font-medium text-sm">{{ scoreInfo.playerName }}</div>
                    <div v-if="scoreInfo.scoreGained > 0" class="text-green-400 text-xs">
                      +{{ scoreInfo.scoreGained }}
                    </div>
                  </div>
                </div>
                <div class="text-white font-bold">{{ scoreInfo.score }}</div>
              </div>
            </div>
          </div>

          <!-- 答題統計 -->
          <div class="mb-6">
            <h3 class="text-white font-bold text-lg mb-4">📊 答題統計</h3>
            <div class="space-y-2">
              <div class="flex justify-between text-white/80 text-sm">
                <span>已答題:</span>
                <span>{{ answeredCount }}/{{ gameStore.playerCount }}</span>
              </div>
              <div class="w-full bg-white/20 rounded-full h-2">
                <div 
                  class="bg-green-400 rounded-full h-2 transition-all duration-300"
                  :style="{ width: answerProgress + '%' }"
                ></div>
              </div>
            </div>
          </div>

          <!-- 控制按鈕 -->
          <div class="space-y-3">
            <button
              @click="nextQuestion"
              :disabled="!canNextQuestion"
              class="w-full btn"
              :class="canNextQuestion ? 'btn-primary' : 'bg-gray-500 cursor-not-allowed'"
            >
              <span v-if="isLastQuestion">查看結果</span>
              <span v-else>下一題</span>
            </button>
            
            <button
              @click="skipQuestion"
              class="w-full btn btn-warning text-sm"
            >
              ⏭ 跳過這題
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 題目結果顯示 -->
    <div v-else-if="gameStore.gameState === 'show_result'" class="h-screen flex flex-col justify-center p-8">
      <div class="max-w-4xl mx-auto w-full text-center">
        <div class="question-card fade-in">
          <h2 class="text-2xl font-bold text-gray-800 mb-6">
            第 {{ currentQuestionNumber }} 題結果
          </h2>
          
          <!-- 主角答案 -->
          <div v-if="hostAnswerInfo.show" class="mb-8">
            <div class="text-lg text-gray-600 mb-2">主角選擇:</div>
            <div class="text-3xl font-bold text-blue-600 mb-4">
              選項 {{ hostAnswerInfo.answer }} - {{ hostAnswerInfo.text }}
            </div>
            <div class="text-gray-600 text-lg">
              主角: {{ hostAnswerInfo.playerName }}
            </div>
          </div>

          <!-- 猜測統計 -->
          <div class="grid grid-cols-2 gap-8 mb-8">
            <div class="text-center">
              <div class="text-4xl font-bold text-green-600">{{ correctGuesses }}</div>
              <div class="text-gray-600">猜對人數</div>
            </div>
            <div class="text-center">
              <div class="text-4xl font-bold text-red-600">{{ wrongGuesses }}</div>
              <div class="text-gray-600">猜錯人數</div>
            </div>
          </div>

          <button
            @click="continueGame"
            class="btn btn-primary text-xl py-4 px-8"
          >
            <span v-if="isLastQuestion">🏆 查看最終結果</span>
            <span v-else>▶️ 繼續遊戲</span>
          </button>
        </div>
      </div>
    </div>

    <!-- 結束遊戲確認對話框 -->
    <div 
      v-if="showEndGameConfirm" 
      class="fixed inset-0 bg-black/50 backdrop-blur-sm z-50 flex items-center justify-center p-4"
      @click.self="showEndGameConfirm = false"
    >
      <div class="bg-white rounded-2xl p-6 max-w-sm w-full">
        <div class="text-center mb-4">
          <div class="text-4xl mb-2">⚠️</div>
          <h3 class="text-xl font-bold text-gray-800 mb-2">確定要結束遊戲？</h3>
          <p class="text-gray-600 text-sm">遊戲進度將不會被保存</p>
        </div>
        
        <div class="flex space-x-4">
          <button
            @click="showEndGameConfirm = false"
            class="flex-1 btn bg-gray-500 hover:bg-gray-600"
          >
            取消
          </button>
          <button
            @click="endGame"
            class="flex-1 btn bg-red-500 hover:bg-red-600"
          >
            結束
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useGameStore } from '@/stores/game'
import { useUIStore } from '@/stores/ui'
import { useGameLogic } from '@/composables/useGameLogic'
import PlayerAvatar from '@/components/PlayerAvatar.vue'

const route = useRoute()
const router = useRouter()
const gameStore = useGameStore()
const uiStore = useUIStore()
const gameLogic = useGameLogic()

// Props
const props = defineProps<{
  roomId: string
}>()

// 響應式數據
const showEndGameConfirm = ref(false)

// 計算屬性
const roomId = computed(() => props.roomId || route.params.roomId as string)

const currentQuestionNumber = computed(() => gameStore.currentQuestionIndex + 1)

const currentHostPlayer = computed(() => gameLogic.currentHostPlayer.value)

const questionOptions = computed(() => {
  const question = gameStore.currentQuestion
  if (!question) return []
  
  return [
    { key: 'A', text: question.optionA, colorClass: 'answer-a' },
    { key: 'B', text: question.optionB, colorClass: 'answer-b' }
  ]
})

const timeLeftClass = computed(() => {
  const timeLeft = gameStore.timeLeft
  if (timeLeft <= 5) return 'text-red-600'
  if (timeLeft <= 10) return 'text-orange-600'
  return 'text-green-600'
})

const answerProgress = computed(() => gameLogic.answerProgress.value)
const canNextQuestion = computed(() => gameLogic.canNextQuestion.value)
const answeredCount = computed(() => gameLogic.answeredCount.value)

const isLastQuestion = computed(() => {
  return gameStore.currentQuestionIndex >= gameStore.totalQuestions - 1
})

// 從 GameStore 獲取玩家答案（而不是 gameLogic）
const playerAnswersFromStore = computed(() => {
  const players = gameStore.currentRoom?.players || {}
  const answers: Record<string, string> = {}
  
  Object.values(players).forEach(player => {
    if (player.currentAnswer) {
      answers[player.id] = player.currentAnswer
    }
  })
  
  console.log('📊 主持人界面玩家答案:', answers)
  return answers
})

// 主角答案信息
const hostAnswerInfo = computed(() => {
  const currentHost = gameStore.currentHost
  if (!currentHost) {
    return { show: false, answer: '', text: '', playerName: '' }
  }

  const hostPlayer = gameStore.getPlayerById(currentHost)
  const hostAnswer = playerAnswersFromStore.value[currentHost]
  
  if (!hostAnswer || !hostPlayer) {
    return { show: false, answer: '', text: '', playerName: '' }
  }
  
  const question = gameStore.currentQuestion
  const answerText = hostAnswer === 'A' ? question?.optionA : question?.optionB
  
  return {
    show: true,
    answer: hostAnswer,
    text: answerText || '',
    playerName: hostPlayer.name
  }
})

// 猜對人數 (排除主角，只計算猜測者)
const correctGuesses = computed(() => {
  const hostId = gameStore.currentHost
  if (!hostId) return 0

  const hostAnswer = playerAnswersFromStore.value[hostId]
  if (!hostAnswer) return 0
  
  const correct = Object.entries(playerAnswersFromStore.value)
    .filter(([playerId, answer]) => 
      playerId !== hostId && // 排除主角
      answer === hostAnswer // 猜對主角答案
    )
    .length
  
  console.log('📊 主持人界面統計: 主角答案=', hostAnswer, '猜對人數=', correct, '所有答案=', playerAnswersFromStore.value)
  return correct
})

// 猜錯人數 (排除主角，只計算猜測者)
const wrongGuesses = computed(() => {
  const hostId = gameStore.currentHost
  if (!hostId) return 0

  const hostAnswer = playerAnswersFromStore.value[hostId]
  if (!hostAnswer) return 0

  const total = Object.keys(playerAnswersFromStore.value).length
  const hostCount = hostId ? (hostId in playerAnswersFromStore.value ? 1 : 0) : 0
  const guessersCount = total - hostCount // 總答題人數 - 主角
  
  return guessersCount - correctGuesses.value
})

// 方法
const isPlayerOnline = (playerId: string) => {
  const player = gameStore.getPlayerById(playerId)
  return player?.isConnected || false
}

// getCorrectAnswerText 方法已移除，「2種人」遊戲不需要正確答案概念

const nextQuestion = () => {
  gameLogic.nextQuestion()
}

const skipQuestion = () => {
  gameLogic.endQuestion()
  setTimeout(() => {
    gameLogic.nextQuestion()
  }, 2000)
}

const continueGame = () => {
  gameLogic.nextQuestion()
}

const endGame = () => {
  showEndGameConfirm.value = false
  
  if (window.debugLogger) {
    window.debugLogger.info('CLEANUP', '主持人結束遊戲，準備跳轉到結果頁面')
  }
  
  gameStore.setGameState('finished')
  router.push(`/results/${roomId.value}`)
}

// 監聽遊戲狀態變化
const unwatchGameState = gameStore.$subscribe((_mutation, state) => {
  if (state.gameState === 'finished') {
    router.push(`/results/${roomId.value}`)
    unwatchGameState()
  }
})

// 生命週期
onMounted(() => {
  // 確保是主持人且在遊戲中
  if (!gameStore.isHost) {
    uiStore.showError('只有主持人可以查看此頁面')
    router.push(`/game/player/${roomId.value}`)
    return
  }
  
  if (gameStore.gameState !== 'playing' && gameStore.gameState !== 'show_result') {
    uiStore.showError('遊戲尚未開始')
    router.push(`/lobby/${roomId.value}`)
    return
  }
})

onUnmounted(() => {
  unwatchGameState()
  
  // 頁面離開時只記錄日誌，不自動清理
  if (window.debugLogger) {
    window.debugLogger.info('CLEANUP', 'GameHostView 組件卸載', {
      gameState: gameStore.gameState,
      note: '不自動清理，等待用戶操作'
    })
  }
})
</script>

<style scoped>
.answer-option {
  @apply p-6 rounded-2xl text-white font-bold text-xl transition-all duration-300 transform hover:scale-105;
}

.question-card {
  @apply bg-white rounded-3xl shadow-2xl p-8 mx-4;
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
