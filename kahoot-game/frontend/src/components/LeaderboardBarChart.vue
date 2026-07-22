<template>
  <div class="leaderboard-bar-chart w-full max-w-4xl mx-auto py-4">
    <!-- 前三名長條圖主要展示區 -->
    <div class="space-y-4">
      <TransitionGroup name="leaderboard-flip" tag="div" class="space-y-4">
        <div
          v-for="(item, index) in displayList"
          :key="item.playerId"
          class="leaderboard-item relative overflow-hidden rounded-2xl p-4 transition-all duration-500 shadow-lg"
          :class="getRankBgClass(index)"
        >
          <!-- 條狀圖動態背景 -->
          <div
            class="absolute top-0 left-0 bottom-0 opacity-25 transition-all duration-1000 ease-out rounded-2xl"
            :class="getBarColorClass(index)"
            :style="{ width: item.barWidth + '%' }"
          ></div>

          <!-- 卡片內部內容 -->
          <div class="relative z-10 flex items-center justify-between">
            <!-- 左側：名次標章 + 頭像 + 姓名 -->
            <div class="flex items-center space-x-4">
              <!-- 名次徽章 -->
              <div 
                class="w-12 h-12 rounded-full flex items-center justify-center font-black text-xl shadow-md border-2"
                :class="getBadgeClass(index)"
              >
                <span v-if="index === 0">🥇</span>
                <span v-else-if="index === 1">🥈</span>
                <span v-else-if="index === 2">🥉</span>
                <span v-else class="text-white text-base">#{{ index + 1 }}</span>
              </div>

              <!-- 玩家頭像 -->
              <PlayerAvatar
                :name="item.playerName"
                :is-online="isPlayerOnline(item.playerId)"
                :is-host="false"
                size="md"
              />

              <!-- 玩家名稱與本題加分 -->
              <div>
                <div class="flex items-center space-x-2">
                  <span class="text-white font-bold text-xl md:text-2xl drop-shadow-sm">
                    {{ item.playerName }}
                  </span>
                  <span 
                    v-if="item.playerId === currentHostId" 
                    class="bg-yellow-400/90 text-black text-xs font-black px-2 py-0.5 rounded-full uppercase"
                  >
                    👑 主角
                  </span>
                </div>
                
                <!-- 本題加分標示 (飄字/動畫) -->
                <div 
                  v-if="animateStage >= 1 && item.scoreGained > 0" 
                  class="text-green-300 font-extrabold text-sm flex items-center animate-bounce mt-0.5"
                >
                  <span>⚡ +{{ item.scoreGained }} 分</span>
                </div>
              </div>
            </div>

            <!-- 右側：當前顯示分數 -->
            <div class="text-right">
              <div class="text-3xl md:text-4xl font-black text-white tracking-tight drop-shadow">
                {{ item.displayScore }} <span class="text-sm font-normal opacity-80">分</span>
              </div>
            </div>
          </div>
        </div>
      </TransitionGroup>

      <!-- 未進入前三名的其他玩家簡要榜 -->
      <div v-if="remainingList.length > 0" class="mt-6 pt-4 border-t border-white/20">
        <h4 class="text-white/80 font-bold text-sm mb-3">其他玩家名次</h4>
        <div class="grid grid-cols-2 md:grid-cols-3 gap-3">
          <div
            v-for="(p, i) in remainingList"
            :key="p.playerId"
            class="flex items-center justify-between bg-black/20 backdrop-blur-sm p-3 rounded-xl border border-white/10"
          >
            <div class="flex items-center space-x-2 truncate">
              <span class="text-white/60 font-mono font-bold text-xs">#{{ i + 4 }}</span>
              <span class="text-white font-semibold text-sm truncate">{{ p.playerName }}</span>
            </div>
            <span class="text-yellow-300 font-bold text-sm ml-2">{{ p.displayScore }}分</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useGameStore } from '@/stores/game'
import PlayerAvatar from '@/components/PlayerAvatar.vue'
import type { ScoreInfo } from '@/types'

const props = defineProps<{
  scores: ScoreInfo[]
  currentHostId?: string | null
}>()

const gameStore = useGameStore()

// 內部狀態項
interface DisplayScoreItem extends ScoreInfo {
  displayScore: number
  previousScore: number
  targetScore: number
  barWidth: number
}

// 0: 舊分數+舊排序, 1: 分數條延伸與數字滾動, 2: 排名位移切換
const animateStage = ref(0)
const internalItems = ref<DisplayScoreItem[]>([])

const isPlayerOnline = (playerId: string) => {
  return gameStore.getPlayerById(playerId)?.isConnected ?? true
}

// 計算與準備渲染資料
const prepareItems = () => {
  if (!props.scores || props.scores.length === 0) {
    internalItems.value = []
    return
  }

  // 1. 舊狀態：預設舊分數 = score - scoreGained
  const items: DisplayScoreItem[] = props.scores.map(s => {
    const prev = Math.max(0, s.score - s.scoreGained)
    return {
      ...s,
      previousScore: prev,
      displayScore: prev,
      targetScore: s.score,
      barWidth: 0
    }
  })

  // 按舊分數排序 (舊名次)
  items.sort((a, b) => b.previousScore - a.previousScore)

  // 計算舊分數的最大值
  const maxPrevScore = Math.max(...items.map(i => i.previousScore), 1)
  items.forEach(i => {
    i.barWidth = Math.min(100, Math.max(12, (i.previousScore / maxPrevScore) * 100))
  })

  internalItems.value = items
  animateStage.value = 0

  // 開啟動畫流程
  startAnimationSequence()
}

const startAnimationSequence = () => {
  // 階段 1: 0.6s 後開始滾動分數與拉伸長條
  setTimeout(() => {
    animateStage.value = 1
    const maxTargetScore = Math.max(...internalItems.value.map(i => i.targetScore), 1)
    
    internalItems.value.forEach(item => {
      // 動態滾動數字
      animateNumber(item)
      // 更新長條圖寬度
      item.barWidth = Math.min(100, Math.max(12, (item.targetScore / maxTargetScore) * 100))
    })
  }, 600)

  // 階段 2: 1.8s 後，按最新總分進行重新排序（觸發 FLIP 排名滑動對調動畫）
  setTimeout(() => {
    animateStage.value = 2
    internalItems.value.sort((a, b) => b.targetScore - a.targetScore)
  }, 1800)
}

// 數字滾動漸變動畫
const animateNumber = (item: DisplayScoreItem) => {
  const start = item.previousScore
  const end = item.targetScore
  if (start === end) {
    item.displayScore = end
    return
  }

  const duration = 1000 // 1秒數字滾動
  const startTime = performance.now()

  const step = (now: number) => {
    const elapsed = now - startTime
    const progress = Math.min(elapsed / duration, 1)
    item.displayScore = Math.floor(start + (end - start) * progress)

    if (progress < 1) {
      requestAnimationFrame(step)
    } else {
      item.displayScore = end
    }
  }

  requestAnimationFrame(step)
}

// 前三名展示
const displayList = computed(() => internalItems.value.slice(0, 3))
// 前三名之後的剩餘列表
const remainingList = computed(() => internalItems.value.slice(3))

// 樣式輔助函式
const getRankBgClass = (index: number) => {
  if (index === 0) return 'bg-gradient-to-r from-amber-500/90 via-yellow-500/90 to-amber-600/90 border-2 border-yellow-300 shadow-yellow-500/30'
  if (index === 1) return 'bg-gradient-to-r from-slate-400/90 via-gray-400/90 to-slate-500/90 border border-slate-200 shadow-slate-400/20'
  if (index === 2) return 'bg-gradient-to-r from-amber-700/90 via-orange-600/90 to-amber-800/90 border border-orange-300 shadow-orange-500/20'
  return 'bg-white/10 backdrop-blur-md border border-white/20'
}

const getBarColorClass = (index: number) => {
  if (index === 0) return 'bg-yellow-200'
  if (index === 1) return 'bg-white'
  if (index === 2) return 'bg-orange-200'
  return 'bg-blue-400'
}

const getBadgeClass = (index: number) => {
  if (index === 0) return 'bg-yellow-400 text-yellow-950 border-yellow-200'
  if (index === 1) return 'bg-slate-200 text-slate-900 border-slate-100'
  if (index === 2) return 'bg-orange-400 text-orange-950 border-orange-200'
  return 'bg-white/20 text-white border-white/30'
}

watch(() => props.scores, () => {
  prepareItems()
}, { deep: true })

onMounted(() => {
  prepareItems()
})
</script>

<style scoped>
.leaderboard-flip-move {
  transition: transform 0.8s cubic-bezier(0.34, 1.56, 0.64, 1);
}

.leaderboard-flip-enter-active,
.leaderboard-flip-leave-active {
  transition: all 0.5s ease;
}

.leaderboard-flip-enter-from,
.leaderboard-flip-leave-to {
  opacity: 0;
  transform: translateY(30px);
}
</style>
