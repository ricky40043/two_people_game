import { ref, computed } from 'vue'
import { useGameStore } from '@/stores/game'
import { useSocketStore } from '@/stores/socket'
import { useUIStore } from '@/stores/ui'
import { apiService } from '@/services/api'
import { useGameTimer } from './useGameTimer'
import type { Question } from '@/types'

export function useGameLogic() {
  const gameStore = useGameStore()
  const socketStore = useSocketStore()
  const uiStore = useUIStore()
  const timer = useGameTimer()

  // 遊戲狀態
  const isLoading = ref(false)
  const currentHostIndex = ref(0)
  const playerAnswers = ref<Record<string, string>>({}) // playerId -> answer

  // 計算屬性
  const currentQuestion = computed(() => gameStore.currentQuestion)
  const totalQuestions = computed(() => gameStore.totalQuestions)
  const currentQuestionNumber = computed(() => gameStore.currentQuestionIndex + 1)
  const playerList = computed(() => Object.values(gameStore.currentRoom?.players || {}))
  
  const currentHostPlayer = computed(() => {
    const players = playerList.value.filter(p => !p.isHost) // 排除房間主持人
    if (players.length === 0) return null
    return players[currentHostIndex.value % players.length]
  })

  const answeredCount = computed(() => Object.keys(playerAnswers.value).length)
  const answerProgress = computed(() => {
    const totalPlayers = playerList.value.length
    if (totalPlayers === 0) return 0
    return Math.round((answeredCount.value / totalPlayers) * 100)
  })

  const canNextQuestion = computed(() => {
    // 所有人都答題了，或者時間到了
    return answeredCount.value >= playerList.value.length || timer.timeLeft.value <= 0
  })

  const isLastQuestion = computed(() => {
    return gameStore.currentQuestionIndex >= totalQuestions.value - 1
  })

  // 載入題目
  const loadQuestions = async (count: number) => {
    isLoading.value = true
    uiStore.setLoading(true, '正在載入題目...')

    try {
      const questions = await apiService.getRandomQuestions(count)
      
      if (questions.length === 0) {
        throw new Error('沒有可用的題目')
      }

      if (questions.length < count) {
        uiStore.showWarning(`只載入了 ${questions.length} 題，少於預期的 ${count} 題`)
      }

      gameStore.setQuestions(questions)
      uiStore.showSuccess(`成功載入 ${questions.length} 題`)
      
      return questions
    } catch (error) {
      console.error('載入題目失敗:', error)
      uiStore.showError('載入題目失敗，請稍後重試')
      
      // 使用備用題目
      const fallbackQuestions = getFallbackQuestions(count)
      gameStore.setQuestions(fallbackQuestions)
      uiStore.showWarning('使用了備用題目')
      
      return fallbackQuestions
    } finally {
      isLoading.value = false
      uiStore.setLoading(false)
    }
  }

  // 開始新題目
  const startQuestion = (questionIndex?: number) => {
    if (questionIndex !== undefined) {
      gameStore.currentQuestionIndex = questionIndex
    }

    const question = currentQuestion.value
    if (!question) {
      uiStore.showError('沒有題目可以顯示')
      return
    }

    // 重置狀態
    playerAnswers.value = {}

    // 設定當前主角
    const hostPlayer = currentHostPlayer.value
    if (hostPlayer) {
      gameStore.setCurrentHost(hostPlayer.id)
      currentHostIndex.value = (currentHostIndex.value + 1) % playerList.value.filter(p => !p.isHost).length
    }

    // 設定遊戲狀態
    gameStore.setGameState('playing')

    // 開始計時
    const timeLimit = gameStore.currentRoom?.questionTimeLimit || 30
    timer.start(timeLimit)

    // 發送 WebSocket 訊息
    socketStore.sendMessage({
      type: 'NEW_QUESTION',
      data: {
        questionId: question.id,
        questionIndex: gameStore.currentQuestionIndex,
        questionText: question.questionText,
        options: [
          question.optionA,
          question.optionB,
          question.optionC,
          question.optionD
        ].filter((option): option is string => Boolean(option)),
        hostPlayer: hostPlayer?.id || '',
        hostPlayerName: hostPlayer?.name || '',
        timeLimit,
        currentQuestion: currentQuestionNumber.value,
        totalQuestions: totalQuestions.value
      }
    })

    console.log(`📝 開始第 ${currentQuestionNumber.value} 題，主角: ${hostPlayer?.name}`)
  }

  // 提交答案
  const submitAnswer = (playerId: string, answer: string) => {
    if (playerAnswers.value[playerId]) {
      console.log('玩家已經答過題了')
      return
    }

    playerAnswers.value[playerId] = answer

    // 計算分數
    const isCorrect = answer === currentQuestion.value?.correctAnswer
    const baseScore = isCorrect ? 100 : 0
    const speedBonus = isCorrect ? Math.round(timer.timeLeft.value * 2) : 0
    const hostBonus = (playerId === gameStore.currentHost && isCorrect) ? 50 : 0
    const totalScore = baseScore + speedBonus + hostBonus

    // 更新玩家分數
    gameStore.updatePlayerScore(playerId, 
      (gameStore.getPlayerById(playerId)?.score || 0) + totalScore
    )

    console.log(`✏️ 玩家 ${playerId} 答題: ${answer}, 分數: ${totalScore}`)

    // 如果所有人都答完了，自動結束題目
    if (answeredCount.value >= playerList.value.length) {
      endQuestion()
    }
  }

  // 結束當前題目
  const endQuestion = () => {
    timer.stop()
    gameStore.setGameState('show_result')

    // 計算答題統計
    const correctAnswers = Object.entries(playerAnswers.value)
      .filter(([_, answer]) => answer === currentQuestion.value?.correctAnswer)
      .length

    const wrongAnswers = answeredCount.value - correctAnswers

    // 發送結果
    socketStore.sendMessage({
      type: 'QUESTION_FINISHED',
      data: {
        questionId: currentQuestion.value?.id,
        correctAnswer: currentQuestion.value?.correctAnswer,
        explanation: currentQuestion.value?.explanation,
        correctAnswers,
        wrongAnswers,
        totalAnswers: answeredCount.value,
        playerAnswers: playerAnswers.value
      }
    })

    console.log(`✅ 題目結束，答對: ${correctAnswers}, 答錯: ${wrongAnswers}`)
  }

  // 下一題
  const nextQuestion = () => {
    if (isLastQuestion.value) {
      endGame()
    } else {
      gameStore.nextQuestion()
      startQuestion()
    }
  }

  // 結束遊戲
  const endGame = () => {
    timer.stop()
    gameStore.setGameState('finished')

    // 計算最終排名
    const finalScores = gameStore.sortedScores

    socketStore.sendMessage({
      type: 'GAME_FINISHED',
      data: {
        winner: finalScores[0] || null,
        finalRanking: finalScores,
        gameStats: {
          duration: '5:30', // TODO: 計算實際遊戲時長
          totalQuestions: totalQuestions.value,
          totalPlayers: playerList.value.length
        }
      }
    })

    console.log('🏆 遊戲結束')
  }

  // 備用題目
  const getFallbackQuestions = (count: number): Question[] => {
    const fallbackQuestions: Question[] = [
      {
        id: 1,
        questionText: '台灣最高的山是？',
        optionA: '玉山',
        optionB: '雪山', 
        optionC: '大霸尖山',
        optionD: '合歡山',
        correctAnswer: 'A',
        explanation: '玉山海拔3952公尺，是台灣最高峰',
        category: '地理',
        difficulty: 1
      },
      {
        id: 2,
        questionText: '一年有幾個季節？',
        optionA: '2個',
        optionB: '3個',
        optionC: '4個',
        optionD: '5個',
        correctAnswer: 'C',
        explanation: '春夏秋冬四個季節',
        category: '常識',
        difficulty: 1
      },
      {
        id: 3,
        questionText: '以下哪個不是程式語言？',
        optionA: 'Python',
        optionB: 'Java',
        optionC: 'HTML',
        optionD: 'JavaScript',
        correctAnswer: 'C',
        explanation: 'HTML是標記語言，不是程式語言',
        category: '資訊',
        difficulty: 2
      }
    ]

    return fallbackQuestions.slice(0, count)
  }

  // 設置計時器回調
  timer.setCallbacks({
    onTick: (timeLeft) => {
      gameStore.updateTimeLeft(timeLeft)
    },
    onFinish: () => {
      endQuestion()
    },
    onWarning: (timeLeft) => {
      console.log(`⚠️ 剩餘 ${timeLeft} 秒`)
    }
  })

  return {
    // 狀態
    isLoading,
    playerAnswers,
    timer,

    // 計算屬性
    currentQuestion,
    currentQuestionNumber,
    currentHostPlayer,
    answeredCount,
    answerProgress,
    canNextQuestion,
    isLastQuestion,

    // 方法
    loadQuestions,
    startQuestion,
    submitAnswer,
    endQuestion,
    nextQuestion,
    endGame,
  }
}
