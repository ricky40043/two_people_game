import { defineStore } from 'pinia'
import { ref } from 'vue'
import { useGameStore } from './game'
import { useUIStore } from './ui'
import { logInfo, logWarn, logError, logDebug, captureError } from '@/utils/logger'
import type { WebSocketMessage } from '@/types'



const resolveWebSocketURL = () => {
  // 優先使用環境變數
  const envWsUrl = import.meta.env.VITE_WS_URL?.toString().trim()
  if (envWsUrl) {
    return envWsUrl
  }

  // Fallback：同源（前後端合一部署 / Vite dev proxy 都走這條）
  const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws'
  return `${protocol}://${window.location.host}/ws`
}

export const useSocketStore = defineStore('socket', () => {
  const socket = ref<WebSocket | null>(null)
  const isConnected = ref(false)
  const reconnectAttempts = ref(0)
  const maxReconnectAttempts = 5
  const shouldReconnect = ref(true)

  // 斷線彈窗狀態（重連失敗 / 房間被關閉時顯示，使用者點確認才回首頁）
  const showDisconnectDialog = ref(false)
  const disconnectDialogTitle = ref('連線已中斷')
  const disconnectDialogMessage = ref('')

  const openDisconnectDialog = (title: string, message: string) => {
    // 已經顯示過就不重複覆蓋，避免訊息被後續事件洗掉
    if (showDisconnectDialog.value) return
    shouldReconnect.value = false
    disconnectDialogTitle.value = title
    disconnectDialogMessage.value = message
    showDisconnectDialog.value = true
    logWarn('WS', '顯示斷線彈窗', { title, message })
  }

  // 使用者按下「確認」→ 清除所有狀態並回首頁
  const acknowledgeDisconnect = () => {
    showDisconnectDialog.value = false
    localStorage.removeItem('ricky_game_session')
    gameStore.resetGame()
    if (socket.value) {
      try { socket.value.close() } catch { /* ignore */ }
      socket.value = null
    }
    isConnected.value = false
    window.location.href = '/'
  }

  // 重連彈窗狀態
  const showRejoinDialog = ref(false)
  const rejoinDialogData = ref<{
    roomId: string
    playerId: string
    playerName: string
    hostName: string
    roomStatus: string
    isHost: boolean
    hostToken?: string
  } | null>(null)

  const gameStore = useGameStore()
  const uiStore = useUIStore()

  // WebSocket 連線
  const connect = () => {
    shouldReconnect.value = true
    logInfo('WS', '嘗試建立 WebSocket 連線')

    // 如果已有連線，先關閉
    if (socket.value) {
      try {
        socket.value.close()
      } catch (e) {
        logWarn('WS', '關閉現有連線時出錯', e)
      }
      socket.value = null
    }

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

      // 檢查是否有舊 session — 發 CHECK_ROOM 查詢，不自動加入
      const savedSession = localStorage.getItem('ricky_game_session')
      if (savedSession) {
        try {
          const session = JSON.parse(savedSession)
          if (session.roomId && session.playerId) {
            logInfo('WS', '偵測到現有會話，發送 CHECK_ROOM 查詢...', { roomId: session.roomId, playerId: session.playerId })
            setTimeout(() => {
              if (isConnected.value) {
                sendMessage({
                  type: 'CHECK_ROOM',
                  data: {
                    roomId: session.roomId,
                    playerId: session.playerId
                  }
                })
              }
            }, 500)
          }
        } catch (e) {
          logError('WS', '解析快取會話失敗', e)
          localStorage.removeItem('ricky_game_session')
        }
      }
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

      uiStore.showError('連線已斷開，嘗試重新連線...')

      // 自動重連，指數退避
      if (reconnectAttempts.value < maxReconnectAttempts) {
        reconnectAttempts.value++
        const delay = Math.min(2000 * Math.pow(1.5, reconnectAttempts.value - 1), 10000)
        logInfo('WS', `將在 ${delay}ms 後進行第 ${reconnectAttempts.value} 次自動重連`, {
          attempt: reconnectAttempts.value,
          max: maxReconnectAttempts
        })

        setTimeout(() => {
          // 檢查是否已經重連成功
          if (!isConnected.value && shouldReconnect.value) {
            connect()
          }
        }, delay)
      } else {
        logError('WS', '達到最大重連次數，停止重連')
        openDisconnectDialog(
          '連線已斷開',
          '與伺服器的連線已中斷且無法自動恢復（可能是閒置太久或網路不穩）。點擊「確認」回到首頁重新開始。'
        )
      }
    }

    socket.value.onerror = (error) => {
      captureError('WS', error, { phase: 'runtime' })
      logWarn('WS', 'WebSocket 連線錯誤', error)
      // 不要在這裡設置 isConnected = false，讓 onclose 處理
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
        console.log('📨 收到 PLAYER_ANSWERED 訊息:', message.data)
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
      case 'ROOM_CLOSED':
        handleRoomClosed(message.data)
        break
      case 'ROOM_SETTINGS_UPDATED':
        handleRoomSettingsUpdated(message.data)
        break
      case 'ROOM_RESET_TO_LOBBY':
        handleRoomResetToLobby(message.data)
        break
      case 'ROOM_STATUS':
        handleRoomStatus(message.data)
        break
      case 'REJOIN_SUCCESS':
        handleRejoinSuccess(message.data)
        break
      case 'PLAYER_REJOINED':
        handlePlayerRejoined(message.data)
        break
      case 'PONG':
        logDebug('WS_RX', '收到 Pong', message.data)
        break
      default:
        logWarn('WS_RX', '收到未知訊息類型', { type: message.type, payload: message })
    }
  }

  // 處理 CHECK_ROOM 回覆
  const handleRoomStatus = (data: any) => {
    logInfo('ROOM', 'CHECK_ROOM 回覆', data)

    const savedSession = localStorage.getItem('ricky_game_session')
    if (!savedSession) return

    let session: any
    try {
      session = JSON.parse(savedSession)
    } catch {
      localStorage.removeItem('ricky_game_session')
      return
    }

    if (!data.exists) {
      // 房間不存在 → 提示後由使用者確認回首頁
      logInfo('ROOM', '房間不存在')
      openDisconnectDialog('房間不存在', '這個房間不存在或已經過期關閉，點擊「確認」回到首頁。')
      return
    }

    if (data.status === 'abandoned') {
      logInfo('ROOM', '房間已廢棄')
      openDisconnectDialog('房間已關閉', '這個房間已經結束並被關閉，點擊「確認」回到首頁。')
      return
    }

    if (!data.playerExists) {
      // 玩家不在房間裡了 → 提示後由使用者確認回首頁
      logInfo('ROOM', '玩家不在房間中')
      openDisconnectDialog('已離開房間', '您已不在這個房間中，點擊「確認」回到首頁。')
      return
    }

    // 注意：status === 'finished' 也要重連。
    // 這樣結算畫面重新整理、或按「再來一局」時才能回到原本的房間，
    // 而不是被踢回首頁被迫重開一間新房。


    // 房間存在且玩家在裡面 → 自動 REJOIN（不彈 dialog）
    logInfo('ROOM', '自動重連中...', {
      roomId: data.roomId,
      playerName: data.playerName,
      isHost: data.isHost,
      status: data.status
    })
    rejoinRoom(data.roomId, session.playerId, session.hostToken)
  }

  // 使用者選擇「返回遊戲」（保留供外部呼叫，但不再自動彈 dialog）
  const confirmRejoin = () => {
    if (!rejoinDialogData.value) return
    const { roomId, playerId, hostToken } = rejoinDialogData.value
    showRejoinDialog.value = false
    rejoinDialogData.value = null
    rejoinRoom(roomId, playerId, hostToken)
  }

  // 使用者選擇「離開，回首頁」
  const cancelRejoin = () => {
    if (!rejoinDialogData.value) return
    showRejoinDialog.value = false
    rejoinDialogData.value = null
    localStorage.removeItem('ricky_game_session')
    gameStore.resetGame()
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

    // 保存主持人會話 - 使用後端返回的 clientId 和 hostToken
    if (data.clientId) {
      localStorage.setItem('ricky_game_session', JSON.stringify({
        roomId: data.roomId,
        playerId: data.clientId,
        playerName: data.hostName,
        isHost: true,
        hostToken: data.hostToken || ''
      }))
    }

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

    // 保存主持人會話（保留 hostToken）
    const existingSession = localStorage.getItem('ricky_game_session')
    let hostToken = ''
    try {
      if (existingSession) {
        const parsed = JSON.parse(existingSession)
        if (parsed.hostToken) hostToken = parsed.hostToken
      }
    } catch { /* ignore */ }

    localStorage.setItem('ricky_game_session', JSON.stringify({
      roomId: data.roomId,
      playerId: data.clientId,
      playerName: data.hostName,
      isHost: true,
      hostToken: data.hostToken || hostToken
    }))

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

      // 保存玩家會話
      localStorage.setItem('ricky_game_session', JSON.stringify({
        roomId: data.roomId,
        playerId: data.playerId,
        playerName: data.playerName,
        isHost: false
      }))
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

    const name = data.playerName || '玩家'
    uiStore.showSuccess(`${name} 加入了遊戲`)
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
    // 檢查房間 ID 是否匹配，避免處理舊房間的訊息
    if (data.roomId && gameStore.currentRoom?.id && data.roomId !== gameStore.currentRoom.id) {
      console.warn(`⚠️ 忽略舊房間的題目: 收到 ${data.roomId}, 當前 ${gameStore.currentRoom.id}`)
      return
    }

    logInfo('QUESTION', '收到新題目', {
      questionId: data.questionId,
      currentQuestion: data.currentQuestion,
      questionIndex: data.questionIndex,
      hostPlayer: data.hostPlayer,
      timeLimit: data.timeLimit,
      roomId: data.roomId
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
    console.log('👥 當前房間玩家:', gameStore.currentRoom?.players)

    // 更新玩家的答題狀態
    const player = gameStore.getPlayerById(data.playerId)
    console.log('👥 查詢玩家 ID:', data.playerId, '結果:', player)
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
    console.log('🎯 完整數據:', JSON.stringify(data, null, 2))

    const totalPlayers = data.totalPlayers || Object.keys(gameStore.currentRoom?.players || {}).length
    const answeredCount = data.answeredCount || 0
    const correctCount = data.correctCount || 0

    console.log(`📈 答題統計: 總玩家=${totalPlayers}, 已作答=${answeredCount}, 答對=${correctCount}`)

    // 確保 scores 是數組
    if (!data.scores || !Array.isArray(data.scores)) {
      console.error('❌ 分數數據格式錯誤:', data.scores)
      logError('SCORE', '分數數據格式錯誤', { scores: data.scores })
      return
    }

    const recordedScores = data.scores.length

    data.scores.forEach((score: any, index: number) => {
      console.log(`   ${index + 1}. ${score.playerName || score.PlayerName} (ID: ${score.playerId || score.PlayerID})`)
      console.log(`      ├─ 總分: ${score.score || score.Score}`)
      console.log(`      ├─ 本題得分: ${score.scoreGained || score.scoreGained || 0}`)
      console.log(`      └─ 排名: 第${score.rank || score.Rank}名`)
    })

    logInfo('SCORE', '分數更新', {
      totalPlayers,
      recordedScores,
      answeredCount,
      correctCount,
      hostAnswer: data.hostAnswer
    })

    if (recordedScores < answeredCount) {
      logWarn('SCORE', '答案記錄不完整', {
        expectedPlayers: answeredCount,
        actualRecords: recordedScores
      })
    }

    // 轉換後端數據格式為前端格式（處理不同的字段名稱）
    const formattedScores = data.scores.map((score: any) => ({
      playerId: score.playerId || score.PlayerID,
      playerName: score.playerName || score.PlayerName,
      score: score.score !== undefined ? score.score : (score.Score || 0),
      rank: score.rank !== undefined ? score.rank : (score.Rank || 0),
      scoreGained: score.scoreGained !== undefined ? score.scoreGained : (score.ScoreGained || 0),
      correctAnswers: score.correctAnswers || score.correctGuesses || 0,
      accuracy: score.accuracy !== undefined ? score.accuracy : (score.accuracy || 0),
      timesAsHost: score.timesAsHost || score.timesAsHost || 0,
      timesAsGuesser: score.timesAsGuesser || score.timesAsGuesser || 0
    }))

    gameStore.updateScores(formattedScores, answeredCount, correctCount)

    // 切換到分數顯示狀態
    gameStore.setGameState('show_result')

    // 顯示本題結果
    const currentPlayerScore = data.scores.find((s: any) => s.playerId === gameStore.currentPlayer?.id || s.PlayerID === gameStore.currentPlayer?.id)
    if (currentPlayerScore) {
      const scoreGained = currentPlayerScore.scoreGained || currentPlayerScore.ScoreGained || 0
      console.log(`💰 當前玩家 ${gameStore.currentPlayer?.name} 本題得分: ${scoreGained}`)

      if (scoreGained > 0) {
        uiStore.showSuccess(`獲得 ${scoreGained} 分！`)
      } else {
        uiStore.showInfo('這次沒有得分，下次加油！')
      }
    } else {
      console.log('⚠️ 找不到當前玩家的分數記錄')
      console.log('當前玩家ID:', gameStore.currentPlayer?.id)
      console.log('分數列表中的玩家IDs:', data.scores?.map((s: any) => s.playerId || s.PlayerID))
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
    console.log('🎯 完整數據:', JSON.stringify(data, null, 2))
    console.log('🎯 當前玩家ID:', gameStore.currentPlayer?.id)

    // 處理不同的後端數據格式
    let finalRanking: any[] = []

    if (data.finalStats && Array.isArray(data.finalStats)) {
      // 新格式：finalStats
      finalRanking = data.finalStats.map((stats: any) => ({
        playerId: stats.playerId || stats.PlayerID,
        playerName: stats.playerName || stats.PlayerName,
        score: stats.totalScore !== undefined ? stats.totalScore : (stats.Score || stats.score || 0),
        rank: stats.rank || stats.Rank || 0,
        correctAnswers: stats.correctGuesses || stats.correctAnswers || stats.CorrectAnswers || 0,
        accuracy: stats.guessAccuracy !== undefined ? Math.round(stats.guessAccuracy) : (stats.accuracy || stats.Accuracy || 0),
        timesAsHost: stats.asHost || stats.AsHost || 0,
        timesAsGuesser: stats.asGuesser || stats.AsGuesser || 0
      }))
      console.log('📊 使用 finalStats 格式')
    } else if (data.finalRanking && Array.isArray(data.finalRanking)) {
      // 舊格式：finalRanking
      finalRanking = data.finalRanking.map((stats: any) => ({
        playerId: stats.playerId || stats.PlayerID,
        playerName: stats.playerName || stats.PlayerName,
        score: stats.totalScore !== undefined ? stats.totalScore : (stats.Score || stats.score || 0),
        rank: stats.rank || stats.Rank || 0,
        correctAnswers: stats.correctGuesses || stats.correctAnswers || stats.CorrectAnswers || 0,
        accuracy: stats.guessAccuracy !== undefined ? Math.round(stats.guessAccuracy) : (stats.accuracy || stats.Accuracy || 0),
        timesAsHost: stats.asHost || stats.AsHost || 0,
        timesAsGuesser: stats.asGuesser || stats.AsGuesser || 0
      }))
      console.log('📊 使用 finalRanking 格式')
    } else {
      console.error('❌ 遊戲結束數據格式錯誤:', data)
      logError('GAME', '遊戲結束數據格式錯誤', { data })
      return
    }

    console.log('📊 最終排名數據:')
    finalRanking.forEach((player: any, index: number) => {
      console.log(`   ${index + 1}. ${player.playerName} (ID: ${player.playerId}) - ${player.score} 分 (答對 ${player.correctAnswers} 題)`)
    })

    logInfo('GAME', '遊戲結束', {
      finalStatsCount: finalRanking.length,
      totalQuestions: data.totalQuestions
    })

    // 更新遊戲狀態和分數
    gameStore.setGameState('finished')
    gameStore.updateScores(finalRanking)

    // 顯示遊戲結束信息
    const winnerName = finalRanking[0]?.playerName || '未知'
    uiStore.showSuccess(`遊戲結束！恭喜 ${winnerName} 獲勝！`)

    console.log('🏁 === 前端遊戲結束處理完成 ===')

    // 保留 session：結算畫面重新整理、或按「再來一局」時才回得去原本的房間。
    // 只在使用者明確按「返回主頁」(cleanupAfterGame) 或房間被清掉時才清除。
    logDebug('GAME', '遊戲結束，保留 session 以便回到房間')
  }

  const handleError = (data: any) => {
    logError('WS_RX', '伺服器返回錯誤', data)
    uiStore.showError(data.message)

    // 如果是找不到房間或玩家，清除會話
    if (data.code === 'ROOM_NOT_FOUND' || data.code === 'PLAYER_NOT_FOUND') {
      localStorage.removeItem('ricky_game_session')
    }
  }

  const handleRoomClosed = (data: any) => {
    logWarn('ROOM', '房間已關閉', data)
    openDisconnectDialog(
      '房間已關閉',
      data.reason || '此房間已被關閉或過期清理，點擊「確認」回到首頁。'
    )
  }

  const handleRejoinSuccess = (data: any) => {
    logInfo('ROOM', '重連成功', data)

    // 使用伺服器回覆的 isHost（通過 hostToken 驗證後的結果）
    const isHost = data.isHost || false

    // 恢復房間狀態
    gameStore.setRoom({
      id: data.roomId,
      hostId: data.hostId || '',
      hostName: data.hostName || '',
      status: data.roomStatus || data.gameState,
      players: {},
      currentQuestion: data.currentQuestionIndex ?? data.currentQuestion ?? 0,
      totalQuestions: data.totalQuestions,
      questionTimeLimit: 30,
      currentHost: data.currentHost || '',
      timeLeft: data.timeLeft || 0,
      questions: [],
      createdAt: new Date()
    })

    // 恢復玩家狀態
    gameStore.setPlayer({
      id: data.playerId,
      name: data.playerName,
      roomId: data.roomId,
      score: data.score || 0,
      isHost: isHost,
      isConnected: true,
      lastActivity: new Date()
    })

    // 更新 localStorage 的 isHost 狀態
    const savedSession = localStorage.getItem('ricky_game_session')
    if (savedSession) {
      try {
        const session = JSON.parse(savedSession)
        session.isHost = isHost
        localStorage.setItem('ricky_game_session', JSON.stringify(session))
      } catch { /* ignore */ }
    }

    // 更新玩家列表
    if (data.players) {
      handlePlayerJoined({ players: data.players, roomId: data.roomId })
    }

    // 恢復當前題目資料（遊戲進行中）
    if (data.currentQuestionData) {
      const qIndex = data.currentQuestionIndex ?? data.currentQuestion ?? 0
      gameStore.setCurrentQuestionIndex(qIndex)
      gameStore.setCurrentQuestion({
        id: data.currentQuestionData.id,
        questionText: data.currentQuestionData.questionText,
        optionA: data.currentQuestionData.optionA,
        optionB: data.currentQuestionData.optionB,
        category: data.currentQuestionData.category || '',
        timesUsed: 0,
        isActive: true,
        createdAt: new Date()
      })
    }

    // 恢復當前主角
    if (data.currentHost) {
      gameStore.setCurrentHost(data.currentHost)
    }

    // 恢復分數
    if (data.finalStats && Array.isArray(data.finalStats)) {
      // 遊戲已結束：直接沿用結算格式，結果頁重新整理後才會完整
      handleGameFinished({ finalStats: data.finalStats, totalQuestions: data.totalQuestions })
    } else if (data.scores && Array.isArray(data.scores)) {
      gameStore.updateScores(data.scores)
    }

    // 恢復剩餘時間（避免重連後按鈕因 timeLeft=0 被 disable）
    if (data.timeLeft !== undefined && data.timeLeft > 0) {
      gameStore.updateTimeLeft(data.timeLeft)
    }

    // 恢復全場所有人本題答題狀態 (避免 1/2 答題進度卡死)
    if (data.playerAnswerStatus && typeof data.playerAnswerStatus === 'object') {
      Object.entries(data.playerAnswerStatus).forEach(([pid, statusInfo]: [string, any]) => {
        gameStore.updatePlayerAnswerStatus(pid, {
          hasAnswered: statusInfo.hasAnswered,
          answer: statusInfo.answer,
          isHost: statusInfo.isHost
        })
      })
      console.log('🔄 重連成功，已同步本題全場答題狀態:', data.playerAnswerStatus)
    }

    // 映射後端房間狀態到前端 gameState
    const roomStatus = data.roomStatus || data.gameState || 'waiting'
    let mappedState: 'waiting' | 'playing' | 'show_result' | 'finished' = 'waiting'
    if (roomStatus === 'waiting') {
      mappedState = 'waiting'
    } else if (['playing', 'question_display', 'answering', 'starting'].includes(roomStatus)) {
      mappedState = 'playing'
    } else if (roomStatus === 'show_result') {
      mappedState = 'show_result'
    } else if (roomStatus === 'finished') {
      mappedState = 'finished'
    }
    gameStore.setGameState(mappedState)

    logInfo('ROOM', '重連處理完成，連線已恢復！', {
      roomId: data.roomId,
      mappedState
    })

    uiStore.showSuccess('歡迎回來，連線已恢復！')
  }

  const handlePlayerRejoined = (data: any) => {
    logInfo('PLAYER', '其他玩家重連', data)
    if (data.players) {
      handlePlayerJoined({
        players: data.players,
        roomId: data.roomId,
        playerName: data.playerName,
        playerId: data.playerId
      })
    }
    const name = data.playerName || '玩家'
    uiStore.showInfo(`${name} 已恢復連線`)
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

  // 離開舊房間的輔助函數（建房/加入前自動呼叫）
  // 注意：不發送 LEAVE_ROOM，避免觸發 PLAYER_LEFT 廣播導致畫面閃爍
  // 後端在 AddPlayer 時會自動踢出同名舊玩家
  const leaveOldRoomIfNeeded = () => {
    const savedSession = localStorage.getItem('ricky_game_session')
    if (savedSession) {
      logInfo('WS', '清除舊 session，由後端自動踢出同名玩家')
      localStorage.removeItem('ricky_game_session')
      gameStore.resetGame()
    }
  }

  // 創建房間
  const createRoom = (hostName: string, totalQuestions: number, questionTimeLimit: number) => {
    leaveOldRoomIfNeeded()
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
    leaveOldRoomIfNeeded()
    logInfo('WS_TX', '送出 JOIN_ROOM 指令', { roomId, playerName })
    sendMessage({
      type: 'JOIN_ROOM',
      data: {
        roomId,
        playerName
      }
    })
  }

  // 重連房間 - 統一使用 REJOIN_ROOM，帶 hostToken 讓伺服器驗證身份
  const rejoinRoom = (roomId: string, playerId: string, hostToken?: string) => {
    // 如果沒傳 hostToken，從 localStorage 取
    if (!hostToken) {
      try {
        const savedSession = localStorage.getItem('ricky_game_session')
        if (savedSession) {
          const session = JSON.parse(savedSession)
          hostToken = session.hostToken || ''
        }
      } catch { /* ignore */ }
    }

    logInfo('WS_TX', '送出 REJOIN_ROOM 指令', { roomId, playerId, hasHostToken: !!hostToken })
    sendMessage({
      type: 'REJOIN_ROOM',
      data: {
        roomId,
        playerId,
        hostToken: hostToken || ''
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
    localStorage.removeItem('ricky_game_session')
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

  // 強制結束遊戲（帶 hostToken）
  const continueGame = () => {
    logInfo('WS_TX', '主持人送出 CONTINUE_GAME')
    sendMessage({ type: 'CONTINUE_GAME', data: {} })
  }

  const forceEndGame = (roomId: string) => {
    let hostToken = ''
    try {
      const savedSession = localStorage.getItem('ricky_game_session')
      if (savedSession) {
        const session = JSON.parse(savedSession)
        hostToken = session.hostToken || ''
      }
    } catch { /* ignore */ }

    logInfo('WS_TX', '送出 FORCE_END_GAME 指令', { roomId, hasHostToken: !!hostToken })
    sendMessage({
      type: 'FORCE_END_GAME',
      data: {
        roomId,
        hostToken
      }
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
      // 0. 使用者明確要離開這個房間，清掉 session 避免下次連線又自動重連回去
      localStorage.removeItem('ricky_game_session')

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

  const handleRoomSettingsUpdated = (data: any) => {
    logInfo('ROOM', '收到房間設定更新', data)
    if (data.totalQuestions) {
      gameStore.setTotalQuestions(data.totalQuestions)
    }
    if (data.questionTimeLimit) {
      gameStore.setQuestionTimeLimit(data.questionTimeLimit)
    }
    uiStore.showInfo(`房間設定已更新 (題數: ${data.totalQuestions || gameStore.currentRoom?.totalQuestions}題, 時間: ${data.questionTimeLimit || gameStore.currentRoom?.questionTimeLimit}秒)`)
  }

  const handleRoomResetToLobby = (data: any) => {
    logInfo('ROOM', '收到房間重置回到大廳事件', data)
    gameStore.setGameState('waiting')
    gameStore.updateScores(data.scores || [])
    uiStore.showSuccess('房主重新開啟了房間！準備下一局...')
  }

  const resetRoomToLobby = (roomId: string) => {
    logInfo('ROOM', '發送重置房間回到大廳請求', { roomId })
    sendMessage({
      type: 'RESET_ROOM_TO_LOBBY',
      data: { roomId }
    })
  }

  const updateRoomSettings = (roomId: string, settings: { totalQuestions?: number, questionTimeLimit?: number }) => {
    logInfo('ROOM', '發送更新房間設定', { roomId, settings })
    sendMessage({
      type: 'UPDATE_ROOM_SETTINGS',
      data: {
        roomId,
        totalQuestions: settings.totalQuestions,
        questionTimeLimit: settings.questionTimeLimit
      }
    })
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
    socket, // 導出 socket 以便外部監聽
    showRejoinDialog,
    rejoinDialogData,
    showDisconnectDialog,
    disconnectDialogTitle,
    disconnectDialogMessage,

    // 動作
    connect,
    disconnect,
    sendMessage,
    updateRoomSettings,
    resetRoomToLobby,
    createRoom,
    joinRoom,
    startGame,
    submitAnswer,
    leaveRoom,
    sendPing,
    continueGame,
    forceEndGame,
    cleanupAfterGame,
    confirmRejoin,
    cancelRejoin,
    acknowledgeDisconnect
  }
})
