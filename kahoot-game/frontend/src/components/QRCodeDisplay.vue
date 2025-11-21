<template>
  <div class="qr-code-display">
    <div class="qr-code-container bg-white p-4 rounded-2xl shadow-lg mx-auto">
      <div 
        ref="qrCodeRef" 
        :style="{ width: size + 'px', height: size + 'px' }"
        class="flex items-center justify-center"
      >
        <div v-if="!generated" class="text-gray-400 text-center">
          <div class="loading-spinner mx-auto mb-2"></div>
          <div class="text-sm">生成中...</div>
        </div>
      </div>
    </div>
    
    <div v-if="showActions" class="mt-4 flex space-x-2">
      <button
        @click="copyToClipboard"
        class="flex-1 btn btn-outline text-sm py-2"
        :disabled="!generated"
      >
        📋 複製內容
      </button>
      <button
        @click="downloadQR"
        class="flex-1 btn btn-outline text-sm py-2"
        :disabled="!generated"
      >
        💾 下載圖片
      </button>
      <button
        v-if="canShare"
        @click="shareQR"
        class="flex-1 btn btn-outline text-sm py-2"
        :disabled="!generated"
      >
        📱 分享
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch, computed, nextTick } from 'vue'
import QRCode from 'qrcode'
import { useUIStore } from '@/stores/ui'

interface Props {
  data: string
  size?: number
  title?: string
  description?: string
  showActions?: boolean
  errorCorrectionLevel?: 'L' | 'M' | 'Q' | 'H'
  margin?: number
  color?: {
    dark: string
    light: string
  }
}

const props = withDefaults(defineProps<Props>(), {
  size: 192,
  title: '掃描 QR Code',
  description: '',
  showActions: true,
  errorCorrectionLevel: 'M',
  margin: 1,
  color: () => ({
    dark: '#1f2937',
    light: '#ffffff'
  })
})

const emit = defineEmits<{
  generated: [canvas: HTMLCanvasElement]
  error: [error: Error]
}>()

const uiStore = useUIStore()

const qrCodeRef = ref<HTMLDivElement>()
const generated = ref(false)

const canShare = computed(() => {
  return 'share' in navigator
})

const generateQRCode = async () => {
  // 詳細日誌記錄 QR Code 生成過程
  if (window.debugLogger) {
    window.debugLogger.info('QR_CODE', '開始生成 QR Code', {
      data: props.data,
      size: props.size,
      hasContainer: !!qrCodeRef.value,
      errorCorrectionLevel: props.errorCorrectionLevel,
      margin: props.margin,
      color: props.color
    })
  }
  
  if (!qrCodeRef.value) {
    const error = 'QR Code 容器不存在'
    console.error(error)
    if (window.debugLogger) {
      window.debugLogger.error('QR_CODE', error)
    }
    return
  }
  
  if (!props.data) {
    const error = 'QR Code 數據為空'
    console.error(error)
    if (window.debugLogger) {
      window.debugLogger.error('QR_CODE', error, { data: props.data })
    }
    return
  }
  
  try {
    generated.value = false
    
    // 清除之前的內容
    qrCodeRef.value.innerHTML = ''
    
    if (window.debugLogger) {
      window.debugLogger.debug('QR_CODE', '開始調用 QRCode.toCanvas')
    }
    
    // 創建 canvas 元素
    const canvas = document.createElement('canvas')
    
    // 生成 QR Code 到 canvas
    await QRCode.toCanvas(canvas, props.data, {
      width: props.size,
      margin: props.margin,
      errorCorrectionLevel: props.errorCorrectionLevel,
      color: props.color
    })
    
    // 將 canvas 添加到容器中
    qrCodeRef.value.appendChild(canvas)
    
    generated.value = true
    emit('generated', canvas)
    
    if (window.debugLogger) {
      window.debugLogger.info('QR_CODE', 'QR Code 生成成功', {
        canvasWidth: canvas.width,
        canvasHeight: canvas.height,
        dataLength: props.data.length
      })
    }
    
    console.log('✅ QR Code 生成成功')
    
  } catch (error: unknown) {
    const err = error instanceof Error ? error : new Error(String(error))
    console.error('QR Code 生成失敗:', err)
    
    if (window.debugLogger) {
      window.debugLogger.error('QR_CODE', 'QR Code 生成失敗', {
        error: err.message,
        stack: err.stack,
        data: props.data,
        size: props.size,
        containerExists: !!qrCodeRef.value
      })
    }
    
    emit('error', err)
    uiStore.showError(`QR Code 生成失敗: ${err.message}`)
  }
}

const copyToClipboard = async () => {
  try {
    await navigator.clipboard.writeText(props.data)
    uiStore.showSuccess('內容已複製到剪貼板')
  } catch (error: unknown) {
    const err = error instanceof Error ? error : new Error(String(error))
    console.error('複製失敗:', err)
    uiStore.showError('複製失敗')
  }
}

const downloadQR = () => {
  if (!qrCodeRef.value || !generated.value) return
  
  const canvas = qrCodeRef.value.querySelector('canvas')
  if (!canvas) return
  
  const link = document.createElement('a')
  link.download = `qrcode-${Date.now()}.png`
  link.href = canvas.toDataURL('image/png')
  link.click()
  
  uiStore.showSuccess('QR Code 已下載')
}

const shareQR = async () => {
  if (!canShare.value || !generated.value) return
  
  try {
    const canvas = qrCodeRef.value?.querySelector('canvas')
    if (!canvas) return
    
    // 將 canvas 轉換為 blob
    canvas.toBlob(async (blob) => {
      if (!blob) return
      
      const file = new File([blob], 'qrcode.png', { type: 'image/png' })
      
      await navigator.share({
        title: props.title,
        text: props.description || props.data,
        files: [file]
      })
    }, 'image/png')
    
  } catch (error) {
    console.error('分享失敗:', error)
    // 降級到分享文字
    try {
      await navigator.share({
        title: props.title,
        text: props.data
      })
    } catch (fallbackError) {
      uiStore.showError('分享失敗')
    }
  }
}

// 監聽數據變化
watch(() => props.data, () => {
  nextTick(() => {
    generateQRCode()
  })
}, { immediate: false })

onMounted(() => {
  nextTick(() => {
    generateQRCode()
  })
})
</script>

<style scoped>
.qr-code-display {
  @apply flex flex-col items-center;
}

.qr-code-container {
  @apply inline-block;
}
</style>
