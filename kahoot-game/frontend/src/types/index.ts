// 玩家類型
export interface Player {
  id: string
  name: string
  roomId: string
  score: number
  isHost: boolean
  isConnected: boolean
  lastActivity: Date
  hasAnswered?: boolean      // 是否已答題
  currentAnswer?: string     // 當前答案 (A 或 B)
  isCurrentHost?: boolean    // 是否為當前題目的主角
}

// 房間類型
export interface Room {
  id: string
  hostId: string
  hostName: string
  status: RoomStatus
  players: Record<string, Player>
  currentQuestion: number
  totalQuestions: number
  questionTimeLimit: number
  currentHost: string
  nextHostOverride?: string
  timeLeft: number
  questions: Question[]
  createdAt: Date
  startedAt?: Date
  finishedAt?: Date
  roomUrl?: string    // QR Code 用的房間 URL
  joinCode?: string   // 房間加入代碼
}

// 房間狀態
export type RoomStatus = 'waiting' | 'starting' | 'question_display' | 'answering' | 'show_result' | 'finished'

// 遊戲狀態
export type GameState = 'waiting' | 'playing' | 'show_result' | 'finished'

// 題目類型 - 適用於「2種人」遊戲
export interface Question {
  id: number
  questionText: string
  optionA: string
  optionB: string
  category?: string
  timesUsed?: number
  isActive?: boolean
  createdAt?: Date
}

// 答案類型
export interface Answer {
  playerId: string
  questionId: number
  answer: string
  isCorrect: boolean
  responseTime: number
  scoreGained: number
  wasHost: boolean
  submittedAt: Date
}

// 分數資訊
export interface ScoreInfo {
  playerId: string
  playerName: string
  score: number
  rank: number
  scoreGained: number
}

// 遊戲統計
export interface GameStatistics {
  roomId: string
  hostName: string
  totalPlayers: number
  totalQuestions: number
  durationSeconds: number
  winnerName: string
  winnerScore: number
  avgAccuracy: number
  avgResponseTime: number
  createdAt: Date
}

// WebSocket 消息類型
export interface WebSocketMessage {
  type: string
  data: any
}

// API 請求類型
export interface CreateRoomRequest {
  hostName: string
  totalQuestions: number
  questionTimeLimit: number
}

export interface JoinRoomRequest {
  roomId: string
  playerName: string
}

export interface SubmitAnswerRequest {
  roomId: string
  questionId: number
  answer: string
  timeUsed: number
}

// API 響應類型
export interface APIResponse<T = any> {
  success: boolean
  data?: T
  error?: string
  message?: string
}

export interface RoomCreatedResponse {
  roomId: string
  hostName: string
  totalQuestions: number
  questionTimeLimit: number
  qrCode: string
  joinUrl: string
  createdAt: Date
}

// 遊戲事件類型
export interface GameEvent {
  type: 'ROOM_CREATED' | 'PLAYER_JOINED' | 'PLAYER_LEFT' | 'GAME_STARTED' | 'NEW_QUESTION' | 'QUESTION_FINISHED' | 'SCORES_UPDATE' | 'GAME_FINISHED' | 'ERROR'
  data: any
}

// 題目顯示選項
export interface QuestionOption {
  key: 'A' | 'B' | 'C' | 'D'
  text: string
  color: string
}

// 排行榜項目
export interface LeaderboardItem {
  rank: number
  playerId: string
  playerName: string
  score: number
  correctAnswers: number
  averageTime: number
}

// 遊戲配置
export interface GameConfig {
  maxPlayersPerRoom: number
  roomIdLength: number
  questionTimeLimit: number
  defaultTotalQuestions: number
}

// QR Code 配置
export interface QRCodeConfig {
  size: number
  margin: number
  color: {
    dark: string
    light: string
  }
}

// 音效類型
export type SoundEffect = 'correct' | 'incorrect' | 'tick' | 'countdown' | 'gameStart' | 'gameEnd'

// 主題顏色
export interface ThemeColors {
  primary: string
  secondary: string
  success: string
  error: string
  warning: string
  info: string
}

// 動畫類型
export type AnimationType = 'fadeIn' | 'slideIn' | 'bounce' | 'pulse' | 'scale'

// 設備類型
export type DeviceType = 'mobile' | 'tablet' | 'desktop'

// 網路狀態
export type NetworkStatus = 'online' | 'offline' | 'slow'

// 用戶偏好設置
export interface UserPreferences {
  soundEnabled: boolean
  animationsEnabled: boolean
  theme: 'light' | 'dark' | 'auto'
  language: string
}

// 錯誤類型
export interface GameError {
  code: string
  message: string
  details?: string
}

// 常見錯誤代碼
export type ErrorCode = 
  | 'ROOM_NOT_FOUND'
  | 'ROOM_FULL'
  | 'GAME_ALREADY_STARTED'
  | 'INVALID_ANSWER'
  | 'PLAYER_NOT_FOUND'
  | 'PERMISSION_DENIED'
  | 'NETWORK_ERROR'
  | 'SERVER_ERROR'

// 工具函數類型
export type Validator<T> = (value: T) => boolean | string

// 響應式狀態
export interface ResponsiveState {
  isMobile: boolean
  isTablet: boolean
  isDesktop: boolean
  screenWidth: number
  screenHeight: number
}

// 計時器類型
export interface Timer {
  duration: number
  remaining: number
  isRunning: boolean
  isPaused: boolean
}

// 遊戲階段
export type GamePhase = 
  | 'setup'        // 設置階段
  | 'lobby'        // 等待大廳
  | 'question'     // 題目顯示
  | 'answering'    // 答題時間
  | 'results'      // 結果顯示
  | 'leaderboard'  // 排行榜
  | 'final'        // 最終結果

// 玩家狀態
export type PlayerStatus = 'waiting' | 'answering' | 'answered' | 'disconnected'

// 組件 Props 基礎類型
export interface BaseComponentProps {
  class?: string
  style?: string
}

// 按鈕變體
export type ButtonVariant = 'primary' | 'secondary' | 'success' | 'danger' | 'warning' | 'info' | 'outline'

// 按鈕大小
export type ButtonSize = 'sm' | 'md' | 'lg' | 'xl'

// 輸入框類型
export type InputType = 'text' | 'number' | 'email' | 'password' | 'tel'

// 模態框類型
export interface ModalOptions {
  title: string
  content: string
  confirmText?: string
  cancelText?: string
  type?: 'info' | 'warning' | 'error' | 'success'
}
