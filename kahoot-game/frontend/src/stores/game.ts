import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Room, Player, Question, GameState, ScoreInfo } from '@/types'

export const useGameStore = defineStore('game', () => {
  // 狀態
  const currentRoom = ref<Room | null>(null)
  const currentPlayer = ref<Player | null>(null)
  const questions = ref<Question[]>([])
  const currentQuestionIndex = ref(0)
  const gameState = ref<GameState>('waiting')
  const scores = ref<ScoreInfo[]>([])
  const timeLeft = ref(0)
  const currentHost = ref<string | null>(null)
  const isHost = ref(false)
  const answeredCount = ref(0)   // 本題作答人數
  const correctCount = ref(0)   // 本題答對人數

  // 計算屬性
  const currentQuestion = computed(() => {
    if (questions.value.length > 0 && currentQuestionIndex.value < questions.value.length) {
      return questions.value[currentQuestionIndex.value]
    }
    return null
  })

  const totalQuestions = computed(() => questions.value.length)

  const gameProgress = computed(() => {
    if (totalQuestions.value === 0) return 0
    return Math.round((currentQuestionIndex.value / totalQuestions.value) * 100)
  })

  const playerCount = computed(() => {
    if (!currentRoom.value?.players) return 0
    return Object.values(currentRoom.value.players).filter(p => !p.isHost).length
  })

  const sortedScores = computed(() => {
    return [...scores.value]
      .filter(s => !currentRoom.value?.players[s.playerId]?.isHost)
      .sort((a, b) => b.score - a.score)
  })

  const myScore = computed(() => {
    if (!currentPlayer.value) return 0
    const myScoreInfo = scores.value.find(s => s.playerId === currentPlayer.value?.id)
    return myScoreInfo?.score || 0
  })

  const myRank = computed(() => {
    if (!currentPlayer.value) return 0
    const sorted = sortedScores.value
    const index = sorted.findIndex(s => s.playerId === currentPlayer.value?.id)
    return index >= 0 ? index + 1 : 0
  })

  // 動作
  const setRoom = (room: Room) => {
    currentRoom.value = room
  }

  const setPlayer = (player: Player) => {
    currentPlayer.value = player
    isHost.value = player.isHost
  }

  const addPlayer = (player: Player) => {
    if (currentRoom.value) {
      currentRoom.value.players[player.id] = player
    }
  }

  const removePlayer = (playerId: string) => {
    if (currentRoom.value && currentRoom.value.players[playerId]) {
      delete currentRoom.value.players[playerId]
    }
  }

  const setQuestions = (newQuestions: Question[]) => {
    questions.value = newQuestions
    currentQuestionIndex.value = 0
  }

  const setCurrentQuestion = (question: Question) => {
    // 確保題目列表有足夠的長度（保留 currentQuestionIndex，不強制重設為 0）
    while (questions.value.length <= currentQuestionIndex.value) {
      questions.value.push({} as Question)
    }
    questions.value[currentQuestionIndex.value] = question

    console.log(`✅ 設置題目 ${currentQuestionIndex.value + 1}:`, question.questionText)
  }

  const setCurrentQuestionIndex = (index: number) => {
    currentQuestionIndex.value = index
  }

  const nextQuestion = () => {
    if (currentQuestionIndex.value < questions.value.length - 1) {
      currentQuestionIndex.value++
      return true
    }
    return false
  }

  const setGameState = (state: GameState) => {
    gameState.value = state
  }

  const updateScores = (newScores: ScoreInfo[], answered?: number, correct?: number) => {
    // 確保 newScores 是一個有效的數組
    if (Array.isArray(newScores)) {
      scores.value = newScores
      console.log('✅ 更新分數數組:', newScores.length, '個玩家')
    } else {
      console.error('❌ updateScores 接收到無效的數組:', newScores)
      scores.value = [] // 設置為空數組避免錯誤
    }
    
    // 更新答題統計
    if (answered !== undefined) {
      answeredCount.value = answered
    }
    if (correct !== undefined) {
      correctCount.value = correct
    }
  }

  const updateTimeLeft = (time: number) => {
    timeLeft.value = time
  }

  const setCurrentHost = (hostId: string) => {
    currentHost.value = hostId
  }

  const resetGame = () => {
    currentRoom.value = null
    currentPlayer.value = null
    questions.value = []
    currentQuestionIndex.value = 0
    gameState.value = 'waiting'
    scores.value = []
    timeLeft.value = 0
    currentHost.value = null
    isHost.value = false
  }

  const updatePlayerScore = (playerId: string, newScore: number) => {
    const scoreIndex = scores.value.findIndex(s => s.playerId === playerId)
    if (scoreIndex >= 0) {
      scores.value[scoreIndex].score = newScore
    } else {
      // 找到玩家名稱
      const player = currentRoom.value?.players[playerId]
      if (player) {
        scores.value.push({
          playerId,
          playerName: player.name,
          score: newScore,
          rank: 0,
          scoreGained: 0
        })
      }
    }
  }

  const isMyTurn = computed(() => {
    return currentHost.value === currentPlayer.value?.id
  })

  const getPlayerById = (playerId: string): Player | null => {
    return currentRoom.value?.players[playerId] || null
  }

  const getCurrentHostPlayer = computed(() => {
    if (!currentHost.value || !currentRoom.value) return null
    return currentRoom.value.players[currentHost.value] || null
  })

  // 更新玩家答題狀態
  const updatePlayerAnswerStatus = (playerId: string, answerInfo: { hasAnswered: boolean, answer?: string, isHost?: boolean }) => {
    if (currentRoom.value && currentRoom.value.players[playerId]) {
      const player = currentRoom.value.players[playerId]
      player.hasAnswered = answerInfo.hasAnswered
      if (answerInfo.answer) {
        player.currentAnswer = answerInfo.answer
      }
      if (answerInfo.isHost !== undefined) {
        player.isCurrentHost = answerInfo.isHost
      }
      console.log(`📝 更新玩家答題狀態: ${player.name} - ${answerInfo.hasAnswered ? '已答題' : '未答題'} - 答案: ${answerInfo.answer || '未知'}`)
    }
  }

  // 重置所有玩家答題狀態
  const resetPlayerAnswerStatus = () => {
    if (currentRoom.value) {
      Object.values(currentRoom.value.players).forEach(player => {
        player.hasAnswered = false
        player.currentAnswer = ''
        player.isCurrentHost = false
      })
      console.log('🔄 重置所有玩家答題狀態')
    }
  }

  // 計算已答題玩家數量
  const answeredPlayersCount = computed(() => {
    if (!currentRoom.value) return 0
    return Object.values(currentRoom.value.players).filter(player => !player.isHost && player.hasAnswered).length
  })

  return {
    // 狀態
    currentRoom,
    currentPlayer,
    questions,
    currentQuestionIndex,
    gameState,
    scores,
    timeLeft,
    currentHost,
    isHost,

    // 計算屬性
    currentQuestion,
    totalQuestions,
    gameProgress,
    playerCount,
    sortedScores,
    myScore,
    myRank,
    isMyTurn,
    getCurrentHostPlayer,
    answeredPlayersCount,
    answeredCount,
    correctCount,

    // 動作
    setRoom,
    setPlayer,
    addPlayer,
    removePlayer,
    setQuestions,
    setCurrentQuestion,
    setCurrentQuestionIndex,
    nextQuestion,
    setGameState,
    updateScores,
    updateTimeLeft,
    setCurrentHost,
    resetGame,
    updatePlayerScore,
    getPlayerById,
    updatePlayerAnswerStatus,
    resetPlayerAnswerStatus
  }
})