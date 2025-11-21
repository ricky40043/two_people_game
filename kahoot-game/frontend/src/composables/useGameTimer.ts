import { ref, computed, onUnmounted } from 'vue'

export function useGameTimer() {
  const timeLeft = ref(0)
  const isRunning = ref(false)
  const isPaused = ref(false)
  
  let timerId: NodeJS.Timeout | null = null
  let startTime = 0

  // 計算屬性
  const progress = computed(() => {
    if (startTime === 0) return 0
    return Math.round(((startTime - timeLeft.value) / startTime) * 100)
  })

  const isWarning = computed(() => timeLeft.value <= 10 && timeLeft.value > 5)
  const isDanger = computed(() => timeLeft.value <= 5)
  
  const timeLeftClass = computed(() => {
    if (isDanger.value) return 'text-red-600'
    if (isWarning.value) return 'text-orange-600'
    return 'text-green-600'
  })

  const formatTime = computed(() => {
    const minutes = Math.floor(timeLeft.value / 60)
    const seconds = timeLeft.value % 60
    if (minutes > 0) {
      return `${minutes}:${seconds.toString().padStart(2, '0')}`
    }
    return seconds.toString()
  })

  // 回調函數
  const callbacks = {
    onTick: null as ((timeLeft: number) => void) | null,
    onFinish: null as (() => void) | null,
    onWarning: null as ((timeLeft: number) => void) | null,
  }

  // 方法
  const start = (duration: number) => {
    stop() // 停止之前的計時器
    
    timeLeft.value = duration
    startTime = duration
    isRunning.value = true
    isPaused.value = false

    timerId = setInterval(() => {
      if (isPaused.value) return

      timeLeft.value--
      
      // 觸發回調
      callbacks.onTick?.(timeLeft.value)
      
      // 警告回調 (剩餘 10 秒時)
      if (timeLeft.value === 10) {
        callbacks.onWarning?.(timeLeft.value)
      }
      
      // 時間到
      if (timeLeft.value <= 0) {
        stop()
        callbacks.onFinish?.()
      }
    }, 1000)
  }

  const pause = () => {
    if (!isRunning.value || isPaused.value) return
    isPaused.value = true
  }

  const resume = () => {
    if (!isRunning.value || !isPaused.value) return
    isPaused.value = false
  }

  const stop = () => {
    if (timerId) {
      clearInterval(timerId)
      timerId = null
    }
    isRunning.value = false
    isPaused.value = false
    timeLeft.value = 0
    startTime = 0
  }

  const addTime = (seconds: number) => {
    if (!isRunning.value) return
    timeLeft.value += seconds
    startTime += seconds
  }

  const setCallbacks = (newCallbacks: Partial<typeof callbacks>) => {
    Object.assign(callbacks, newCallbacks)
  }

  // 清理
  onUnmounted(() => {
    stop()
  })

  return {
    // 狀態
    timeLeft,
    isRunning,
    isPaused,
    progress,
    isWarning,
    isDanger,
    timeLeftClass,
    formatTime,

    // 方法
    start,
    pause,
    resume,
    stop,
    addTime,
    setCallbacks,
  }
}
