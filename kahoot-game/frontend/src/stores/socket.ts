import { defineStore } from 'pinia'
import { ref } from 'vue'
import { useGameStore } from './game'
import { useUIStore } from './ui'
import { logInfo, logWarn, logError, logDebug, captureError } from '@/utils/logger'
import type { WebSocketMessage } from '@/types'

const normalizePath = (path: string) => (path.startsWith('/') ? path : `/${path}`)

const resolveWebSocketURL = () => {
  const envUrl = import.meta.env.VITE_WS_URL?.toString().trim()
  if (envUrl) {
    return envUrl
  }

  if (typeof window !== 'undefined') {
    const path = normalizePath(import.meta.env.VITE_WS_PATH?.toString().trim() || '/ws')

    if (import.meta.env.DEV) {
      const protocol = (import.meta.env.VITE_WS_PROTOCOL || (window.location.protocol === 'https:' ? 'wss' : 'ws'))
        .toString()
        .replace(/:$/, '')
      const port = import.meta.env.VITE_WS_PORT?.toString().trim() || '8080'
      return `${protocol}://${window.location.hostname}:${port}${path}`
    }

    const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws'
    return `${protocol}//${window.location.host}${path}`
  }

  return 'ws://127.0.0.1:8080/ws'
}

export const useSocketStore = defineStore('socket', () => {
  const socket = ref<WebSocket | null>(null)
  const isConnected = ref(false)
  const reconnectAttempts = ref(0)
  const maxReconnectAttempts = 5
  const shouldReconnect = ref(true)

  const gameStore = useGameStore()
  const uiStore = useUIStore()

  // WebSocket 連線
  const connect = () => {
    shouldReconnect.value = true
    logInfo('WS', '嘗試建立 WebSocket 連線')

    const wsUrl = resolveWebSocketURL()

    logDebug('WS', '連線目標 URL', { wsUrl })

    try {
      socket.value = new WebSocket(wsUrl)
    } catch (error) {
      captureError('WS', error, { phase: 'create' })
      uiStore.showError('無法創建 WebSocket 連線')
      return
    }

    // WebSocket 事件處理
    socket.value.onopen = () => {
      logInfo('WS', 'WebSocket 連線成功')
      isConnected.value = true
      reconnectAttempts.value = 0
      uiStore.showSuccess('連線成功')
    }

    socket.value.onclose = (event) => {
      logWarn('WS', 'WebSocket 連線斷開', {
        code: event.code,
        reason: event.reason,
        shouldReconnect: shouldReconnect.value
      })
      isConnected.value = false

      if (!shouldReconnect.value) {
        logInfo('WS', '偵測到手動中斷，跳過自動重連')
        uiStore.showInfo('連線已關閉')
        return
      }

      uiStore.showError('連線已斷開')
      
      // 自動重連
      if (reconnectAttempts.value < maxReconnectAttempts) {
        setTimeout(() => {
          reconnectAttempts.value++
          logInfo('WS', '自動重連', {
            attempt: reconnectAttempts.value,
            max: maxReconnectAttempts
          })
          connect()
        }, 2000 * reconnectAttempts.value)
      }
    }

    socket.value.onerror = (error) => {
      captureError('WS', error, { phase: 'runtime' })
      isConnected.value = false
      uiStore.showError('WebSocket 連線錯誤')
    }

    socket.value.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data)
        logDebug('WS_RX', '收到 WebSocket 訊息', message)
        handleMessage(message)
      } catch (error) {
        captureError('WS_RX', error, { raw: event.data })
      }
    }
  }

  // 處理收到的訊息
  const handleMessage = (message: any) => {
    logInfo('WS_RX', `處理訊息 ${message.type}`)
    
    switch (message.type) {
      case 'ROOM_CREATED':
        handleRoomCreated(message.data)
        break
      case 'PLAYER_JOINED':
        handlePlayerJoined(message.data)
        break
      case 'PLAYER_LEFT':
        handlePlayerLeft(message.data)
        break
      case 'GAME_STARTED':
        handleGameStarted(message.data)
        break
      case 'NEW_QUESTION':
        handleNewQuestion(message.data)
        break
      case 'TIMER_UPDATE':
        handleTimerUpdate(message.data)
        break
      case 'QUESTION_TIMEOUT':
        handleQuestionTimeout(message.data)
        break
      case 'QUESTION_INVALID':
        handleQuestionInvalid(message.data)
        break
      case 'QUESTION_SKIPPED':
        handleQuestionSkipped(message.data)
        break
      case 'ANSWER_SUBMITTED':
        handleAnswerSubmitted(message.data)
        break
      case 'PLAYER_ANSWERED':
        handlePlayerAnswered(message.data)
        break
      case 'QUESTION_FINISHED':
        handleQuestionFinished(message.data)
        break
      case 'SCORES_UPDATE':
        handleScoresUpdate(message.data)
        break
      case 'GAME_FINISHED':
        handleGameFinished(message.data)
        break
      case 'HOST_JOINED':
        handleHostJoined(message.data)
        break
      case 'ERROR':
        handleError(message.data)
        break
      case 'PONG':
        logDebug('WS_RX', '收到 Pong', message.data)
        break
      default:
        logWarn('WS_RX', '收到未知訊息類型', { type: message.type, payload: message })
    }
  }

  // 事件處理函數
  const handleRoomCreated = (data: any) => {
      logInfo('ROOM', '房間創建成功', data)
      gameStore.setRoom({
        id: data.roomId,
        hostId: gameStore.currentPlayer?.id || '',
        hostName: data.hostName,
        status: 'waiting',
        players: {},
        currentQuestion: 0,
        totalQuestions: data.totalQuestions || 10,
        questionTimeLimit: data.questionTimeLimit || 30,
        currentHost: '',
        timeLeft: 0,
        questions: [],
        createdAt: new Date(),
        roomUrl: data.roomUrl || `${window.location.origin}/join/${data.roomId}`,
        joinCode: data.joinCode || data.roomId
      })
      uiStore.showSuccess('房間創建成功！')
  }

  const handleHostJoined = (data: any) => {
      logInfo('HOST', '主持人加入房間', data)

      // 若目前尚未設置房間，初始化一份
      if (!gameStore.currentRoom) {
        gameStore.setRoom({
          id: data.roomId,
          hostId: data.clientId,
          hostName: data.hostName,
          status: 'waiting',
          players: {},
          currentQuestion: 0,
          totalQuestions: data.totalQuestions || 10,
          questionTimeLimit: data.questionTimeLimit || 30,
          currentHost: '',
          timeLeft: 0,
          questions: [],
          createdAt: new Date(),
          roomUrl: data.roomUrl || `${window.location.origin}/join/${data.roomId}`,
          joinCode: data.roomId
        })
      }

      // 確保房間與玩家資料存在
      if (!gameStore.currentRoom) {
        logWarn('HOST', '房間尚未初始化，無法設置主持人', data)
        return
      }

      // 更新主持人的玩家資料
      const hostPlayer = {
        id: data.clientId,
        name: data.hostName,
        roomId: data.roomId,
        score: 0,
        isHost: true,
        isConnected: true,
        lastActivity: new Date()
      }

      gameStore.setPlayer(hostPlayer)
      gameStore.addPlayer(hostPlayer)

      // 更新房間資訊
      gameStore.currentRoom.hostId = data.clientId
      gameStore.currentRoom.hostName = data.hostName
      gameStore.currentRoom.roomUrl = data.roomUrl || gameStore.currentRoom.roomUrl

      if (data.players && Array.isArray(data.players)) {
        gameStore.currentRoom.players = {}
        data.players.forEach((player: any) => {
          gameStore.addPlayer({
            id: player.id,
            name: player.name,
            roomId: data.roomId,
            score: player.score || 0,
            isHost: player.isHost || player.id === data.clientId,
            isConnected: true,
            lastActivity: new Date()
          })
        })
      }

      uiStore.showSuccess('主持人已加入房間')
  }

  const handlePlayerJoined = (data: any) => {
      logInfo('PLAYER', '收到玩家加入事件', {
        playerId: data.playerId,
        playerName: data.playerName,
        roomId: data.roomId,
        totalPlayers: data.totalPlayers,
        playersCount: data.players ? data.players.length : 0
      })
      
      // 如果是當前玩家自己加入房間且尚未設置房間
      if (data.playerId && !gameStore.currentRoom) {
        // 設置房間信息
        gameStore.setRoom({
          id: data.roomId,
          hostId: '', // 將在獲取完整房間信息時更新
          hostName: '',
          status: 'waiting',
          players: {},
          currentQuestion: 0,
          totalQuestions: 10, // 預設值，將在獲取完整信息時更新
          questionTimeLimit: 30,
          currentHost: '',
          timeLeft: 0,
          questions: [],
          createdAt: new Date()
        })
        
        // 設置當前玩家信息
        gameStore.setPlayer({
          id: data.playerId,
          name: data.playerName,
          roomId: data.roomId,
          score: 0,
          isHost: false,
          isConnected: true,
          lastActivity: new Date()
        })
        
        logDebug('PLAYER', '設置當前玩家與房間完成', {
          currentPlayerId: data.playerId,
          roomId: data.roomId
        })
      }
      
      // 處理完整玩家列表更新
      if (data.players && Array.isArray(data.players)) {
        logDebug('PLAYER', '更新玩家列表', {
          receivedPlayersCount: data.players.length,
          currentPlayersCount: Object.keys(gameStore.currentRoom?.players || {}).length
        })
        
        // 清空現有玩家列表（避免重複）
        if (gameStore.currentRoom) {
          gameStore.currentRoom.players = {}
        }
        
        // 添加所有玩家到房間
        data.players.forEach((player: any) => {
          gameStore.addPlayer({
            id: player.id,
            name: player.name,
            roomId: data.roomId,
            score: player.score || 0,
            isHost: player.isHost || false,
            isConnected: true,
            lastActivity: new Date()
          })
          
          logDebug('PLAYER', '添加玩家至列表', {
            playerId: player.id,
            playerName: player.name,
            isHost: player.isHost
          })
        })
        logDebug('PLAYER', '玩家列表更新完成', {
          finalPlayersCount: Object.keys(gameStore.currentRoom?.players || {}).length
        })
      } else {
        // 如果沒有完整列表，只添加單個玩家
        if (data.playerId && data.playerName) {
          gameStore.addPlayer({
            id: data.playerId,
            name: data.playerName,
            roomId: data.roomId,
            score: 0,
            isHost: false,
            isConnected: true,
            lastActivity: new Date()
          })
          
          logDebug('PLAYER', '添加單個玩家', {
            playerId: data.playerId,
            playerName: data.playerName
          })
        }
      }
      
      uiStore.showSuccess(`${data.playerName} 加入了遊戲`)
  }

  const handlePlayerLeft = (data: any) => {
      logInfo('PLAYER', '玩家離開房間', data)

      if (Array.isArray(data.players)) {
        if (gameStore.currentRoom) {
          gameStore.currentRoom.players = {}
        }

        data.players.forEach((player: any) => {
          gameStore.addPlayer({
            id: player.id,
            name: player.name,
            roomId: player.roomId,
            score: player.score || 0,
            isHost: player.isHost || false,
            isConnected: player.isConnected ?? true,
            lastActivity: player.lastActivity ? new Date(player.lastActivity) : new Date()
          })
        })
      } else {
        gameStore.removePlayer(data.playerId)
      }

      if (typeof data.currentHost === 'string') {
        gameStore.setCurrentHost(data.currentHost)
      }

      if (data.resetAnswers) {
        gameStore.resetPlayerAnswerStatus()
      }

      uiStore.showInfo(`${data.playerName} 離開了遊戲`)

      if (data.hostChanged) {
        const newHost = gameStore.getPlayerById(data.currentHost)?.name || '未知'
        uiStore.showWarning(`主角已更新：${newHost}`)
      }
  }

  const handleGameStarted = (data: any) => {
      logInfo('GAME', '遊戲開始', data)
      gameStore.setGameState('playing')
      gameStore.setCurrentHost(data.firstHost)
      uiStore.showSuccess('遊戲開始！')
  }

  const handleNewQuestion = (data: any) => {
      logInfo('QUESTION', '收到新題目', {
        questionId: data.questionId,
        currentQuestion: data.currentQuestion,
        questionIndex: data.questionIndex,
        hostPlayer: data.hostPlayer,
        timeLimit: data.timeLimit
      })

      const questionText = data.questionText || data.question
      if (!questionText) {
        logError('QUESTION', '題目文字為空', data)
        uiStore.showError('收到的題目內容為空')
        return
      }

      const questionIndex = data.questionIndex !== undefined ? data.questionIndex : (data.currentQuestion - 1)

      // 重置上一題的答案狀態，避免統計沿用舊資料
      gameStore.resetPlayerAnswerStatus()

      logDebug('QUESTION', '更新題目索引', {
        beforeIndex: gameStore.currentQuestionIndex,
        afterIndex: questionIndex,
        questionsLength: gameStore.questions.length
      })
      
      // 先設置索引，再設置題目內容
      gameStore.setCurrentQuestionIndex(questionIndex)
      
      // 設置當前題目
      gameStore.setCurrentQuestion({
        id: data.questionId,
        questionText: questionText,
        optionA: data.optionA,
        optionB: data.optionB,
        category: data.category || '',
        timesUsed: 0,
        isActive: true,
        createdAt: new Date()
      })
      
      // 設置其他遊戲狀態
      gameStore.setCurrentHost(data.hostPlayer)
      gameStore.updateTimeLeft(data.timeLimit)
      gameStore.setGameState('playing')
      
      const currentQuestion = gameStore.currentQuestion
      logDebug('QUESTION', '驗證題目設置', {
        hasCurrentQuestion: !!currentQuestion,
        currentQuestionText: currentQuestion?.questionText,
        currentQuestionIndex: gameStore.currentQuestionIndex,
        questionsArrayLength: gameStore.questions.length
      })
      
      // 顯示題目和主角信息
      const hostPlayerName = gameStore.getPlayerById(data.hostPlayer)?.name || '未知'
      const questionNumber = data.currentQuestion || (questionIndex + 1)
      
      logDebug('QUESTION', '題目設置完成', {
        questionIndex,
        questionNumber,
        hostPlayerName,
        hasQuestion: !!gameStore.currentQuestion
      })
      
      uiStore.showInfo(`第 ${questionNumber} 題 - 主角：${hostPlayerName}`)
  }

  const handleTimerUpdate = (data: any) => {
    const timestamp = Date.now()

    if (!window.timerEvents) {
      window.timerEvents = []
    }

    window.timerEvents.push({
      timestamp,
      timeLeft: data.timeLeft,
      questionIndex: data.questionIndex || gameStore.currentQuestionIndex
    })

    window.timerEvents = window.timerEvents.filter(event => timestamp - event.timestamp < 5000)

    const recentEvents = window.timerEvents.filter(event => 
      timestamp - event.timestamp < 2000 && 
      event.questionIndex === (data.questionIndex || gameStore.currentQuestionIndex)
    )

    const sameTimes = recentEvents.filter(event => event.timeLeft === data.timeLeft)

    if (sameTimes.length > 1) {
      logWarn('TIMER', '檢測到重複計時器事件', {
        duplicateCount: sameTimes.length,
        questionIndex: data.questionIndex || gameStore.currentQuestionIndex
      })
    }

    logDebug('TIMER', '計時器更新', {
      timeLeft: data.timeLeft,
      questionIndex: data.questionIndex,
      timestamp,
      recentEventCount: recentEvents.length
    })

    gameStore.updateTimeLeft(data.timeLeft)
  }

  const handleQuestionTimeout = (data: any) => {
    logInfo('QUESTION', '答題時間結束', data)
    gameStore.updateTimeLeft(0)
    uiStore.showWarning('時間到！')
  }

  const handleQuestionInvalid = (data: any) => {
    console.log('❌ 題目無效:', data)
    uiStore.showError(data.message || '主角未答題，本題無效')
    
    // 設置遊戲狀態為顯示結果（但不計分）
    gameStore.setGameState('show_result')
  }

  const handleQuestionSkipped = (data: any) => {
    console.log('⏭️ 題目跳過:', data)
    uiStore.showInfo(data.message || '沒有玩家答題，跳過本題')
    
    // 直接進入下一題準備狀態
    gameStore.setGameState('playing')
  }

  const handleAnswerSubmitted = (data: any) => {
    console.log('📝 收到答案提交確認:', data)
    
    // 更新玩家的答案狀態
    const player = gameStore.getPlayerById(data.playerId)
    if (player) {
      gameStore.updatePlayerAnswerStatus(data.playerId, {
        hasAnswered: true,
        answer: data.answer,
        isHost: data.isHost
      })
      
      console.log(`✅ 已更新玩家 ${data.playerName} 的答案: ${data.answer}`)
    } else {
      console.error('❌ 找不到玩家:', data.playerId)
    }
    
    logDebug('QUESTION', '答案已提交', data)
    uiStore.showSuccess('答案已提交！')
  }

  const handlePlayerAnswered = (data: any) => {
    console.log('👥 收到玩家答題通知:', data)
    
    // 更新玩家的答題狀態
    const player = gameStore.getPlayerById(data.playerId)
    if (player) {
      const updateData: any = {
        hasAnswered: true,
        isHost: data.isHost
      }
      
      // 如果訊息包含答案（主持人能收到），則更新答案
      if (data.answer) {
        updateData.answer = data.answer
        console.log(`👤 玩家 ${data.playerName} 已答題: ${data.answer} (主持人可見)`)
      } else {
        console.log(`👤 玩家 ${data.playerName} 已答題 (不可見答案)`)
      }
      
      gameStore.updatePlayerAnswerStatus(data.playerId, updateData)
    } else {
      console.error('❌ 找不到玩家:', data.playerId)
    }
  }

  const handleQuestionFinished = (data: any) => {
    logInfo('QUESTION', '題目結束', data)
    gameStore.setGameState('show_result')
  }

  const handleScoresUpdate = (data: any) => {
    console.log('📊 === 前端收到分數更新事件 ===')
    console.log('🎯 主角答案:', data.hostAnswer)
    console.log('📈 所有玩家分數詳情:')
    
    const totalPlayers = Object.keys(gameStore.currentRoom?.players || {}).length
    const recordedScores = data.scores ? data.scores.length : 0

    data.scores?.forEach((score: any, index: number) => {
      console.log(`   ${index + 1}. ${score.playerName} (ID: ${score.playerId})`)
      console.log(`      ├─ 總分: ${score.score}`)
      console.log(`      ├─ 本題得分: ${score.scoreGained}`)
      console.log(`      └─ 排名: 第${score.rank}名`)
    })

    logInfo('SCORE', '分數更新', {
      totalPlayers,
      recordedScores,
      hostAnswer: data.hostAnswer
    })

    if (recordedScores < totalPlayers) {
      logWarn('SCORE', '答案記錄不完整', {
        expectedPlayers: totalPlayers,
        actualRecords: recordedScores
      })
    }

    gameStore.updateScores(data.scores)
    
    // 切換到分數顯示狀態
    gameStore.setGameState('show_result')
    
    // 顯示本題結果
    const currentPlayerScore = data.scores.find((s: any) => s.playerId === gameStore.currentPlayer?.id)
    if (currentPlayerScore) {
      const scoreGained = currentPlayerScore.scoreGained || 0
      console.log(`💰 當前玩家 ${gameStore.currentPlayer?.name} 本題得分: ${scoreGained}`)
      
      if (scoreGained > 0) {
        uiStore.showSuccess(`獲得 ${scoreGained} 分！`)
      } else {
        uiStore.showInfo('這次沒有得分，下次加油！')
      }
    } else {
      console.log('⚠️ 找不到當前玩家的分數記錄')
      console.log('當前玩家ID:', gameStore.currentPlayer?.id)
      console.log('分數列表中的玩家IDs:', data.scores?.map((s: any) => s.playerId))
    }
    
    // 顯示主角答案
    if (data.hostAnswer) {
      setTimeout(() => {
        uiStore.showInfo(`主角選擇了：${data.hostAnswer}`)
      }, 1000)
    }
    
    console.log('📊 === 前端分數更新處理完成 ===')
  }

  const handleGameFinished = (data: any) => {
    console.log('🏁 === 前端收到遊戲結束事件 ===')
    console.log('📊 最終統計數據:', data.finalStats)
    console.log('🎮 總題數:', data.totalQuestions)
    
    // 轉換新的統計格式為舊的分數格式 (為了兼容現有前端)
    const finalRanking = data.finalStats?.map((stats: any) => ({
      playerId: stats.playerId,
      playerName: stats.playerName,
      score: stats.totalScore,
      rank: stats.rank,
      correctAnswers: stats.correctGuesses,  // 只計算猜測正確次數
      accuracy: Math.round(stats.guessAccuracy), // 猜測正確率
      timesAsHost: stats.asHost,
      timesAsGuesser: stats.asGuesser
    })) || []

    logInfo('GAME', '遊戲結束', {
      finalStatsCount: data.finalStats?.length ?? 0,
      totalQuestions: data.totalQuestions
    })

    gameStore.setGameState('finished')
    gameStore.updateScores(finalRanking)
    
    // 顯示遊戲結束信息
    const winnerName = finalRanking[0]?.playerName || '未知'
    uiStore.showSuccess(`遊戲結束！恭喜 ${winnerName} 獲勝！`)
    
    console.log('🏁 === 前端遊戲結束處理完成 ===')
    
    // 不自動清理，等待用戶操作
    logDebug('GAME', '等待玩家操作清理')
  }

  const handleError = (data: any) => {
    logError('WS_RX', '伺服器返回錯誤', data)
    uiStore.showError(data.message)
  }

  // 發送消息
  const sendMessage = (message: WebSocketMessage) => {
    if (socket.value && isConnected.value) {
      logDebug('WS_TX', '發送 WebSocket 訊息', message)
      socket.value.send(JSON.stringify(message))
    } else {
      logError('WS_TX', 'WebSocket 未連線，無法發送消息', { message })
      uiStore.showError('網路連線已斷開，請重新連線')
    }
  }

  // 創建房間
  const createRoom = (hostName: string, totalQuestions: number, questionTimeLimit: number) => {
    logInfo('WS_TX', '送出 CREATE_ROOM 指令', {
      hostName,
      totalQuestions,
      questionTimeLimit
    })
    sendMessage({
      type: 'CREATE_ROOM',
      data: {
        hostName,
        totalQuestions,
        questionTimeLimit
      }
    })
  }

  // 加入房間
  const joinRoom = (roomId: string, playerName: string) => {
    logInfo('WS_TX', '送出 JOIN_ROOM 指令', { roomId, playerName })
    sendMessage({
      type: 'JOIN_ROOM',
      data: {
        roomId,
        playerName
      }
    })
  }

  // 開始遊戲
  const startGame = (roomId: string) => {
    logInfo('WS_TX', '送出 START_GAME 指令', { roomId })
    sendMessage({
      type: 'START_GAME',
      data: {
        roomId
      }
    })
  }

  // 提交答案
  const submitAnswer = (roomId: string, questionId: number, answer: string, timeUsed: number) => {
    logDebug('WS_TX', '送出 SUBMIT_ANSWER', { roomId, questionId, answer, timeUsed })
    sendMessage({
      type: 'SUBMIT_ANSWER',
      data: {
        roomId,
        questionId,
        answer,
        timeUsed
      }
    })
  }

  // 離開房間
  const leaveRoom = () => {
    logInfo('WS_TX', '送出 LEAVE_ROOM 指令')
    sendMessage({
      type: 'LEAVE_ROOM',
      data: {}
    })
  }

  // 發送 Ping
  const sendPing = () => {
    sendMessage({
      type: 'PING',
      data: {}
    })
  }

  // 遊戲結束後清理資源
  const cleanupAfterGame = () => {
      logInfo('CLEANUP', '開始清理遊戲資源', {
        hasSocket: !!socket.value,
        isConnected: isConnected.value,
        currentRoom: gameStore.currentRoom?.id,
        currentPlayer: gameStore.currentPlayer?.name
      })

    try {
      // 1. 斷開 WebSocket 連接（但保持重連能力）
      if (socket.value) {
        // 暫時禁用重連，避免清理過程中的自動重連
        shouldReconnect.value = false
        socket.value.close()
        socket.value = null
        isConnected.value = false
        logDebug('CLEANUP', 'WebSocket 連接已斷開')
      }
      
      // 2. 清理遊戲狀態
      gameStore.resetGame()
      
      // 3. 清理 UI 狀態
      uiStore.clearAllMessages()
      
      // 4. 清理計時器事件記錄
      if (window.timerEvents) {
        window.timerEvents = []
      }
      
      // 5. 重置重連計數和恢復重連能力
      reconnectAttempts.value = 0
      shouldReconnect.value = true  // 重要：恢復重連能力，允許下次連接
      
      logInfo('CLEANUP', '遊戲資源清理完成', {
        socketClosed: !socket.value,
        gameReset: !gameStore.currentRoom,
        reconnectAttemptsReset: reconnectAttempts.value === 0,
        shouldReconnectEnabled: shouldReconnect.value
      })

      uiStore.showInfo('遊戲已重置，可以創建新房間')

    } catch (error) {
      captureError('CLEANUP', error)
    }
  }

  // 斷開連線
  const disconnect = () => {
    shouldReconnect.value = false
    if (socket.value) {
      socket.value.close()
      socket.value = null
      isConnected.value = false
    }
  }

  return {
    // 狀態
    isConnected,
    reconnectAttempts,

    // 動作
    connect,
    disconnect,
    sendMessage,
    createRoom,
    joinRoom,
    startGame,
    submitAnswer,
    leaveRoom,
    sendPing,
    cleanupAfterGame
  }
})
