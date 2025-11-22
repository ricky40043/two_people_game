import axios from 'axios'
import type { Question, APIResponse, CreateRoomRequest, RoomCreatedResponse } from '@/types'

const normalizeUrl = (url: string) => url.replace(/\/+$/, '')

const resolveApiBaseURL = () => {
  const envUrl = import.meta.env.VITE_API_URL?.toString().trim()
  if (envUrl) {
    const url = normalizeUrl(envUrl)
    return url.endsWith('/api') ? url : `${url}/api`
  }

  if (typeof window !== 'undefined') {
    if (import.meta.env.DEV) {
      const protocol = (import.meta.env.VITE_API_PROTOCOL || (window.location.protocol === 'https:' ? 'https' : 'http'))
        .toString()
        .replace(/:$/, '')
      const port = import.meta.env.VITE_API_PORT?.toString().trim() || '8080'
      return `${protocol}://${window.location.hostname}:${port}/api`
    }

    return `${normalizeUrl(window.location.origin)}/api`
  }

  return '/api'
}

// 創建 axios 實例
const api = axios.create({
  baseURL: resolveApiBaseURL(),
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// 請求攔截器
api.interceptors.request.use(
  (config) => {
    console.log('📤 API 請求:', config.method?.toUpperCase(), config.url)
    return config
  },
  (error) => {
    console.error('❌ API 請求錯誤:', error)
    return Promise.reject(error)
  }
)

// 響應攔截器
api.interceptors.response.use(
  (response) => {
    console.log('📨 API 響應:', response.status, response.config.url)
    return response
  },
  (error) => {
    console.error('❌ API 響應錯誤:', error.response?.status, error.response?.data)
    return Promise.reject(error)
  }
)

// API 方法
export const apiService = {
  // 健康檢查
  async health(): Promise<any> {
    const response = await api.get('/health')
    return response.data
  },

  // 題目相關
  async getRandomQuestions(count: number): Promise<Question[]> {
    const response = await api.get<APIResponse<Question[]>>(`/questions/random/${count}`)
    if (response.data.success) {
      return response.data.data || []
    }
    throw new Error(response.data.error || '獲取題目失敗')
  },

  async getQuestions(params?: {
    category?: string
    difficulty?: number
    limit?: number
  }): Promise<Question[]> {
    const response = await api.get<APIResponse<Question[]>>('/questions', { params })
    if (response.data.success) {
      return response.data.data || []
    }
    throw new Error(response.data.error || '獲取題目失敗')
  },

  async createQuestion(question: {
    questionText: string
    optionA: string
    optionB: string
    optionC: string
    optionD: string
    correctAnswer: string
    explanation?: string
    category?: string
    difficulty?: number
  }): Promise<Question> {
    const response = await api.post<APIResponse<Question>>('/questions', question)
    if (response.data.success) {
      return response.data.data!
    }
    throw new Error(response.data.error || '創建題目失敗')
  },

  // 房間相關
  async createRoom(data: CreateRoomRequest): Promise<RoomCreatedResponse> {
    const response = await api.post<APIResponse<RoomCreatedResponse>>('/rooms', data)
    if (response.data.success) {
      return response.data.data!
    }
    throw new Error(response.data.error || '創建房間失敗')
  },

  async getRoom(roomId: string): Promise<any> {
    const response = await api.get<APIResponse<any>>(`/rooms/${roomId}`)
    if (response.data.success) {
      return response.data.data
    }
    throw new Error(response.data.error || '獲取房間失敗')
  },

  async deleteRoom(roomId: string): Promise<void> {
    const response = await api.delete<APIResponse<void>>(`/rooms/${roomId}`)
    if (!response.data.success) {
      throw new Error(response.data.error || '刪除房間失敗')
    }
  },

  // 遊戲相關
  async getActiveGames(): Promise<any[]> {
    const response = await api.get<APIResponse<any[]>>('/games')
    if (response.data.success) {
      return response.data.data || []
    }
    throw new Error(response.data.error || '獲取活躍遊戲失敗')
  },

  async getGameStats(gameId: number): Promise<any> {
    const response = await api.get<APIResponse<any>>(`/games/${gameId}/stats`)
    if (response.data.success) {
      return response.data.data
    }
    throw new Error(response.data.error || '獲取遊戲統計失敗')
  },
}

export default api
