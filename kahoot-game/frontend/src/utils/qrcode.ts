import QRCode from 'qrcode'

export interface QRCodeOptions {
  width?: number
  margin?: number
  errorCorrectionLevel?: 'L' | 'M' | 'Q' | 'H'
  color?: {
    dark: string
    light: string
  }
}

export const generateQRCode = async (
  text: string, 
  options: QRCodeOptions = {}
): Promise<string> => {
  const defaultOptions = {
    width: 256,
    margin: 1,
    errorCorrectionLevel: 'M' as const,
    color: {
      dark: '#000000',
      light: '#FFFFFF'
    }
  }
  
  const finalOptions = { ...defaultOptions, ...options }
  
  try {
    const dataURL = await QRCode.toDataURL(text, finalOptions)
    return dataURL
  } catch (error) {
    console.error('QR Code 生成失敗:', error)
    throw error
  }
}

export const generateQRCodeToCanvas = async (
  container: HTMLElement,
  text: string,
  options: QRCodeOptions = {}
): Promise<HTMLCanvasElement> => {
  const defaultOptions = {
    width: 256,
    margin: 1,
    errorCorrectionLevel: 'M' as const,
    color: {
      dark: '#000000',
      light: '#FFFFFF'
    }
  }
  
  const finalOptions = { ...defaultOptions, ...options }
  
  try {
    const canvas = document.createElement('canvas')
    await QRCode.toCanvas(canvas, text, finalOptions)

    // 清空容器並附加新的 canvas
    container.innerHTML = ''
    container.appendChild(canvas)

    return canvas
  } catch (error) {
    console.error('QR Code 生成失敗:', error)
    throw error
  }
}

export const downloadQRCode = (canvas: HTMLCanvasElement, filename: string = 'qrcode.png') => {
  const link = document.createElement('a')
  link.download = filename
  link.href = canvas.toDataURL('image/png')
  link.click()
}

export const shareQRCode = async (
  canvas: HTMLCanvasElement, 
  title: string = 'QR Code',
  text: string = ''
) => {
  if (!('share' in navigator)) {
    throw new Error('Web Share API 不被支援')
  }
  
  return new Promise<void>((resolve, reject) => {
    canvas.toBlob(async (blob) => {
      if (!blob) {
        reject(new Error('無法生成圖片'))
        return
      }
      
      try {
        const file = new File([blob], 'qrcode.png', { type: 'image/png' })
        
        await navigator.share({
          title,
          text,
          files: [file]
        })
        
        resolve()
      } catch (error) {
        reject(error)
      }
    }, 'image/png')
  })
}
