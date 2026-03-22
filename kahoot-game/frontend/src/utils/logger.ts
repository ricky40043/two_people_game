export type LogLevel = 'INFO' | 'WARN' | 'ERROR' | 'DEBUG'

interface DebugLogger {
  info: (category: string, message: string, data?: unknown) => void
  warn: (category: string, message: string, data?: unknown) => void
  error: (category: string, message: string, data?: unknown) => void
  debug: (category: string, message: string, data?: unknown) => void
}

declare global {
  interface Window {
    debugLogger?: DebugLogger
    timerEvents?: Array<{
      timestamp: number
      timeLeft: number
      questionIndex: number
    }>
  }
}

const FALLBACK_LOGGER: DebugLogger = {
  info: (category, message, data) => {
    console.info(`%c[${category}] ${message}`, 'color: #3b82f6; font-weight: bold;', data ?? '');
  },
  warn: (category, message, data) => {
    console.warn(`%c[${category}] ${message}`, 'color: #f59e0b; font-weight: bold;', data ?? '');
  },
  error: (category, message, data) => {
    console.error(`%c[${category}] ${message}`, 'color: #ef4444; font-weight: bold;', data ?? '');
  },
  debug: (category, message, data) => {
    // 允許 DEBUG 級別日誌，使用灰色顯示
    console.debug(`%c[${category}] ${message}`, 'color: #9ca3af;', data ?? '');
  }
}

function getLogger(): DebugLogger {
  return window?.debugLogger ?? FALLBACK_LOGGER
}

export function logEvent(level: LogLevel, category: string, message: string, data?: unknown) {
  const logger = getLogger()
  switch (level) {
    case 'INFO':
      logger.info(category, message, data)
      break
    case 'WARN':
      logger.warn(category, message, data)
      break
    case 'ERROR':
      logger.error(category, message, data)
      break
    case 'DEBUG':
    default:
      logger.debug(category, message, data)
      break
  }
}

export const logInfo = (category: string, message: string, data?: unknown) =>
  logEvent('INFO', category, message, data)

export const logWarn = (category: string, message: string, data?: unknown) =>
  logEvent('WARN', category, message, data)

export const logError = (category: string, message: string, data?: unknown) =>
  logEvent('ERROR', category, message, data)

export const logDebug = (category: string, message: string, data?: unknown) =>
  logEvent('DEBUG', category, message, data)

export function captureError(category: string, error: unknown, context?: Record<string, unknown>) {
  const payload = {
    ...(context || {}),
    error: error instanceof Error ? {
      name: error.name,
      message: error.message,
      stack: error.stack
    } : error
  }
  logError(category, '捕捉到錯誤', payload)
}
