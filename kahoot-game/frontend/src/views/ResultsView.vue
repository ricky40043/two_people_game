<template>
  <div class="results-view min-h-screen bg-gradient-to-br from-purple-600 via-blue-600 to-blue-700 p-4">
    <div class="max-w-4xl mx-auto">
      <!-- 標題區域 -->
      <div class="text-center mb-8 fade-in">
        <div class="text-6xl mb-4">🏆</div>
        <h1 class="text-4xl font-bold text-white mb-2">遊戲結束！</h1>
        <p class="text-white/80 text-lg">
          房間 {{ roomId }} • {{ gameStore.totalQuestions }} 題
        </p>
      </div>

      <!-- 冠軍慶祝 -->
      <div v-if="winner" class="text-center mb-8 fade-in" style="animation-delay: 0.3s">
        <div class="card card-body bg-gradient-to-r from-yellow-400 to-yellow-600 text-black">
          <div class="text-4xl mb-2">👑</div>
          <h2 class="text-2xl font-bold mb-2">恭喜冠軍！</h2>
          <div class="text-3xl font-bold">{{ winner.playerName }}</div>
          <div class="text-xl mt-2">{{ winner.score }} 分</div>
        </div>
      </div>

      <div class="grid lg:grid-cols-2 gap-8">
        <!-- 排行榜 -->
        <div class="fade-in" style="animation-delay: 0.5s">
          <div class="card card-body">
            <h3 class="text-xl font-bold text-white mb-6 flex items-center">
              <span class="text-2xl mr-2">🏆</span>
              最終排行榜
            </h3>
            
            <div class="space-y-3">
              <div
                v-for="(player, index) in finalRanking"
                :key="player.playerId"
                class="flex items-center justify-between p-4 rounded-xl transition-all duration-300"
                :class="[
                  'bg-white/10',
                  {
                    'bg-gradient-to-r from-yellow-400/30 to-yellow-600/30 border-2 border-yellow-400': index === 0,
                    'bg-gradient-to-r from-gray-300/30 to-gray-500/30 border border-gray-400': index === 1,
                    'bg-gradient-to-r from-orange-400/30 to-orange-600/30 border border-orange-400': index === 2,
                  }
                ]"
              >
                <div class="flex items-center space-x-4">
                  <!-- 排名 -->
                  <div 
                    class="w-10 h-10 rounded-full flex items-center justify-center font-bold text-lg"
                    :class="{
                      'bg-yellow-400 text-black': index === 0,
                      'bg-gray-400 text-white': index === 1,
                      'bg-orange-400 text-white': index === 2,
                      'bg-white/20 text-white': index > 2
                    }"
                  >
                    {{ index + 1 }}
                  </div>
                  
                  <!-- 玩家頭像 -->
                  <PlayerAvatar
                    :name="player.playerName"
                    :is-online="true"
                    :is-host="player.playerId === gameStore.currentRoom?.hostId"
                    size="md"
                  />
                  
                  <!-- 玩家資訊 -->
                  <div>
                    <div class="text-white font-semibold">{{ player.playerName }}</div>
                    <div class="text-white/60 text-sm">
                      {{ player.correctAnswers || 0 }}/{{ gameStore.totalQuestions }} 答對
                    </div>
                  </div>
                </div>
                
                <!-- 分數 -->
                <div class="text-right">
                  <div class="text-xl font-bold text-white">{{ player.score }}</div>
                  <div class="text-white/60 text-sm">分</div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- 遊戲統計 -->
        <div class="space-y-6">
          <!-- 個人成績 -->
          <div v-if="myStats" class="card card-body fade-in" style="animation-delay: 0.7s">
            <h3 class="text-xl font-bold text-white mb-4 flex items-center">
              <span class="text-2xl mr-2">📊</span>
              您的成績
            </h3>
            
            <div class="grid grid-cols-2 gap-4">
              <div class="bg-white/10 rounded-xl p-3 text-center">
                <div class="text-2xl font-bold text-blue-400">#{{ myStats.rank }}</div>
                <div class="text-white/80 text-sm">最終排名</div>
              </div>
              <div class="bg-white/10 rounded-xl p-3 text-center">
                <div class="text-2xl font-bold text-green-400">{{ myStats.score }}</div>
                <div class="text-white/80 text-sm">總分</div>
              </div>
              <div class="bg-white/10 rounded-xl p-3 text-center">
                <div class="text-2xl font-bold text-yellow-400">{{ myStats.correctAnswers }}</div>
                <div class="text-white/80 text-sm">答對題數</div>
              </div>
              <div class="bg-white/10 rounded-xl p-3 text-center">
                <div class="text-2xl font-bold text-purple-400">{{ myStats.accuracy }}%</div>
                <div class="text-white/80 text-sm">正確率</div>
              </div>
            </div>
            
            <!-- 主角統計 -->
            <div v-if="myStats.timesAsHost > 0" class="mt-4 bg-yellow-400/20 rounded-xl p-3">
              <div class="flex items-center justify-center">
                <span class="text-xl mr-2">👑</span>
                <span class="text-white font-medium">擔任主角 {{ myStats.timesAsHost }} 次</span>
              </div>
            </div>
          </div>

          <!-- 遊戲統計 -->
          <div class="card card-body fade-in" style="animation-delay: 0.9s">
            <h3 class="text-xl font-bold text-white mb-4 flex items-center">
              <span class="text-2xl mr-2">📈</span>
              遊戲統計
            </h3>
            
            <div class="space-y-3 text-sm">
              <div class="flex justify-between text-white/80">
                <span>遊戲時長：</span>
                <span class="font-medium">{{ gameDuration }}</span>
              </div>
              <div class="flex justify-between text-white/80">
                <span>參與玩家：</span>
                <span class="font-medium">{{ finalRanking.length }} 人</span>
              </div>
              <div class="flex justify-between text-white/80">
                <span>題目總數：</span>
                <span class="font-medium">{{ gameStore.totalQuestions }} 題</span>
              </div>
              <div class="flex justify-between text-white/80">
                <span>平均正確率：</span>
                <span class="font-medium">{{ averageAccuracy }}%</span>
              </div>
              <div class="flex justify-between text-white/80">
                <span>最高分：</span>
                <span class="font-medium">{{ winner?.score || 0 }} 分</span>
              </div>
            </div>
          </div>

          <!-- 操作按鈕 -->
          <div class="space-y-3 fade-in" style="animation-delay: 1.1s">
            <button
              @click="shareResults"
              v-if="canShare"
              class="w-full btn btn-primary"
            >
              📱 分享結果
            </button>
            
            <button
              @click="playAgain"
              class="w-full btn btn-success"
            >
              🎮 再來一局
            </button>
            
            <button
              @click="returnHome"
              class="w-full btn btn-outline"
            >
              🏠 返回主頁
            </button>
          </div>
        </div>
      </div>

      <!-- 感謝訊息 -->
      <div class="text-center mt-12 fade-in" style="animation-delay: 1.3s">
        <div class="card card-body">
          <h3 class="text-lg font-bold text-white mb-2">🎉 感謝遊玩！</h3>
          <p class="text-white/70 text-sm">
            希望您玩得開心！歡迎再次體驗我們的遊戲。
          </p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useGameStore } from '@/stores/game'
import { useSocketStore } from '@/stores/socket'
import { useUIStore } from '@/stores/ui'
import PlayerAvatar from '@/components/PlayerAvatar.vue'
import { logInfo, logWarn, logError } from '@/utils/logger'

const route = useRoute()
const router = useRouter()
const gameStore = useGameStore()
const socketStore = useSocketStore()
const uiStore = useUIStore()

// Props
const props = defineProps<{
  roomId: string
}>()

// 計算屬性
const roomId = computed(() => props.roomId || route.params.roomId as string)

const finalRanking = computed(() => gameStore.sortedScores)

const winner = computed(() => finalRanking.value[0] || null)

  const myStats = computed(() => {
  if (!gameStore.currentPlayer) return null

  const myRanking = finalRanking.value.find(
    p => p.playerId === gameStore.currentPlayer?.id
  )

  if (!myRanking) {
    console.log('⚠️ 找不到我的排名數據，玩家ID:', gameStore.currentPlayer?.id)
    return null
  }

  console.log('📊 我的統計數據:', myRanking)

  return {
    rank: myRanking.rank || 0,
    score: myRanking.score || 0,
    correctAnswers: myRanking.correctAnswers || 0,
    accuracy: myRanking.accuracy || 0,
    timesAsHost: myRanking.timesAsHost || 0,
  }
})

const averageAccuracy = computed(() => {
  if (finalRanking.value.length === 0) return 0
  
  const totalAccuracy = finalRanking.value.reduce((sum, player) => {
    return sum + (player.accuracy || 0)
  }, 0)
  
  const avgAccuracy = Math.round(totalAccuracy / finalRanking.value.length)
  console.log('📊 平均正確率計算:', totalAccuracy, '/', finalRanking.value.length, '=', avgAccuracy)
  
  return avgAccuracy
})

const gameDuration = computed(() => {
  // TODO: 計算實際遊戲時長
  const minutes = Math.floor(Math.random() * 10) + 3
  const seconds = Math.floor(Math.random() * 60)
  return `${minutes}:${seconds.toString().padStart(2, '0')}`
})

const canShare = computed(() => {
  return 'share' in navigator
})

// 方法
const shareResults = async () => {
  if (!canShare.value) return
  
  try {
    const myRank = myStats.value?.rank || '未知'
    const myScore = myStats.value?.score || 0
    
    await navigator.share({
      title: '🎮 Ricky 遊戲小舖結果',
      text: `我在 Ricky 遊戲小舖中獲得第 ${myRank} 名，得分 ${myScore} 分！一起來挑戰吧！`,
      url: window.location.origin
    })
    logInfo('VIEW_RESULTS', '分享遊戲結果成功', {
      roomId: roomId.value,
      myRank,
      myScore
    })
  } catch (error) {
    logError('VIEW_RESULTS', '分享遊戲結果失敗', error)
    uiStore.showError('分享失敗')
  }
}

const playAgain = () => {
  if (window.debugLogger) {
    window.debugLogger.info('CLEANUP', '用戶點擊"再來一局"，開始清理遊戲資源')
  }
  
  logInfo('VIEW_RESULTS', '用戶選擇再來一局', { roomId: roomId.value })
  uiStore.showInfo('正在清理遊戲資源...')
  
  // 清理遊戲資源
  socketStore.cleanupAfterGame()
  
  // 短暫延遲後跳轉，確保清理完成
  setTimeout(() => {
    router.push('/create')
  }, 500)
}

const returnHome = () => {
  if (window.debugLogger) {
    window.debugLogger.info('CLEANUP', '用戶點擊"返回主頁"，開始清理遊戲資源')
  }
  
  logInfo('VIEW_RESULTS', '用戶選擇返回主頁', { roomId: roomId.value })
  uiStore.showInfo('正在清理遊戲資源...')
  
  // 清理遊戲資源
  socketStore.cleanupAfterGame()
  
  // 短暫延遲後跳轉，確保清理完成
  setTimeout(() => {
    router.push('/')
  }, 500)
}

// 生命週期
onMounted(() => {
  logInfo('VIEW_RESULTS', '頁面載入', {
    roomId: roomId.value,
    finalRankingCount: finalRanking.value.length
  })
  // 確保有遊戲數據
  if (!gameStore.currentRoom || finalRanking.value.length === 0) {
    uiStore.showWarning('沒有遊戲結果數據')
    logWarn('VIEW_RESULTS', '沒有遊戲結果數據', {
      hasRoom: !!gameStore.currentRoom,
      rankingCount: finalRanking.value.length
    })
  }
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

.fade-in {
  animation: fadeIn 0.6s ease-out both;
}
</style>
