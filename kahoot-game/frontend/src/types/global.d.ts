// 全局類型聲明

interface DebugLogger {
  log(level: string, category: string, message: string, data?: any): void
  info(category: string, message: string, data?: any): void
  warn(category: string, message: string, data?: any): void
  error(category: string, message: string, data?: any): void
  debug(category: string, message: string, data?: any): void
  exportLogs(): void
  clear(): void
}

interface TimerEvent {
  timestamp: number
  timeLeft: number
  questionIndex: number
}

declare global {
  interface Window {
    debugLogger?: DebugLogger
    timerEvents?: TimerEvent[]
  }
}

export {}
