import confetti from 'canvas-confetti'

/**
 * 房主/大螢幕視角的全屏禮炮慶祝彩花
 */
export function fireHostConfetti() {
  const duration = 3.5 * 1000
  const end = Date.now() + duration

  // 中心第一波大爆發
  confetti({
    particleCount: 100,
    spread: 100,
    origin: { y: 0.6 }
  })

  // 兩側連續交替砲彈
  const interval: any = setInterval(() => {
    if (Date.now() > end) {
      return clearInterval(interval)
    }

    // 左側發射
    confetti({
      startVelocity: 30,
      spread: 360,
      ticks: 60,
      origin: { x: Math.random() * 0.2 + 0.1, y: Math.random() - 0.2 },
      colors: ['#FFD700', '#FF4500', '#00E5FF', '#7C4DFF', '#39FF14']
    })

    // 右側發射
    confetti({
      startVelocity: 30,
      spread: 360,
      ticks: 60,
      origin: { x: Math.random() * 0.2 + 0.7, y: Math.random() - 0.2 },
      colors: ['#FFD700', '#FF4500', '#00E5FF', '#7C4DFF', '#39FF14']
    })
  }, 250)
}

/**
 * 玩家個人視角彩花（僅前三名噴發）
 * @param rank 最終名次 (1, 2, 3)
 */
export function firePlayerConfetti(rank: number) {
  if (rank < 1 || rank > 3) return

  let colors: string[] = []
  let particleCount = 80
  let spread = 80

  if (rank === 1) {
    // 第一名：金色 + 繽紛色彩
    colors = ['#FFD700', '#FFA500', '#FF4500', '#00E5FF', '#7C4DFF', '#39FF14']
    particleCount = 120
    spread = 100
  } else if (rank === 2) {
    // 第二名：銀亮質感
    colors = ['#C0C0C0', '#E6E6FA', '#FFFFFF', '#A9A9A9', '#DCDCDC', '#B0C4DE']
    particleCount = 90
    spread = 80
  } else if (rank === 3) {
    // 第三名：古銅質感
    colors = ['#CD7F32', '#B87333', '#D2691E', '#8B4513', '#E9967A', '#F4A460']
    particleCount = 70
    spread = 70
  }

  // 發射中央慶祝彩花
  confetti({
    particleCount,
    spread,
    origin: { y: 0.6 },
    colors
  })

  // 第一名額外加碼第二次兩側爆發
  if (rank === 1) {
    setTimeout(() => {
      confetti({
        particleCount: 60,
        angle: 60,
        spread: 55,
        origin: { x: 0 },
        colors
      })
      confetti({
        particleCount: 60,
        angle: 120,
        spread: 55,
        origin: { x: 1 },
        colors
      })
    }, 400)
  }
}
