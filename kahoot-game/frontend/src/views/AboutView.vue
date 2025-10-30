<template>
  <div class="about-view min-h-screen flex items-center justify-center p-4">
    <div class="max-w-2xl w-full">
      <!-- 返回按鈕 -->
      <div class="mb-6">
        <router-link 
          to="/"
          class="inline-flex items-center text-white/70 hover:text-white transition-colors"
        >
          <svg class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"></path>
          </svg>
          返回主頁
        </router-link>
      </div>

      <!-- 關於內容 -->
      <div class="card card-body fade-in">
        <div class="text-center mb-8">
          <div class="text-4xl mb-4">🎮</div>
          <h1 class="text-3xl font-bold text-white mb-2">風格多人遊戲</h1>
          <p class="text-white/70 text-lg">即時互動 • 主角輪替 • 寓教於樂</p>
        </div>

        <!-- 遊戲介紹 -->
        <div class="space-y-6">
          <div>
            <h2 class="text-xl font-bold text-white mb-3">🎯 遊戲特色</h2>
            <div class="grid md:grid-cols-2 gap-4">
              <div class="bg-white/10 rounded-xl p-4">
                <div class="text-2xl mb-2">📱</div>
                <h3 class="text-white font-semibold mb-1">QR Code 快速加入</h3>
                <p class="text-white/70 text-sm">掃描即可快速加入遊戲，無需下載 App</p>
              </div>
              <div class="bg-white/10 rounded-xl p-4">
                <div class="text-2xl mb-2">👑</div>
                <h3 class="text-white font-semibold mb-1">主角輪替制</h3>
                <p class="text-white/70 text-sm">每題輪流當主角，朗讀題目給其他人</p>
              </div>
              <div class="bg-white/10 rounded-xl p-4">
                <div class="text-2xl mb-2">⚡</div>
                <h3 class="text-white font-semibold mb-1">即時同步</h3>
                <p class="text-white/70 text-sm">WebSocket 技術確保所有玩家同步</p>
              </div>
              <div class="bg-white/10 rounded-xl p-4">
                <div class="text-2xl mb-2">🏆</div>
                <h3 class="text-white font-semibold mb-1">智能計分</h3>
                <p class="text-white/70 text-sm">答對 + 速度加分 + 主角獎勵</p>
              </div>
            </div>
          </div>

          <div>
            <h2 class="text-xl font-bold text-white mb-3">📋 遊戲規則</h2>
            <div class="space-y-3 text-white/80">
              <div class="flex items-start space-x-3">
                <span class="text-blue-400 font-bold">1.</span>
                <p>主持人創建房間並設定題目數量和答題時間</p>
              </div>
              <div class="flex items-start space-x-3">
                <span class="text-blue-400 font-bold">2.</span>
                <p>玩家掃描 QR Code 或輸入房間代碼加入遊戲</p>
              </div>
              <div class="flex items-start space-x-3">
                <span class="text-blue-400 font-bold">3.</span>
                <p>每題會隨機指定一位「主角」負責朗讀題目</p>
              </div>
              <div class="flex items-start space-x-3">
                <span class="text-blue-400 font-bold">4.</span>
                <p>所有玩家在限定時間內選擇答案</p>
              </div>
              <div class="flex items-start space-x-3">
                <span class="text-blue-400 font-bold">5.</span>
                <p>根據正確性和速度獲得分數，主角答對有額外加分</p>
              </div>
              <div class="flex items-start space-x-3">
                <span class="text-blue-400 font-bold">6.</span>
                <p>遊戲結束後查看排行榜和個人統計</p>
              </div>
            </div>
          </div>

          <div>
            <h2 class="text-xl font-bold text-white mb-3">🎯 計分系統</h2>
            <div class="bg-white/10 rounded-xl p-4">
              <div class="space-y-2 text-sm">
                <div class="flex justify-between text-white/80">
                  <span>答對基礎分：</span>
                  <span class="font-bold text-green-400">+100 分</span>
                </div>
                <div class="flex justify-between text-white/80">
                  <span>速度加分：</span>
                  <span class="font-bold text-blue-400">剩餘時間 × 2</span>
                </div>
                <div class="flex justify-between text-white/80">
                  <span>主角獎勵：</span>
                  <span class="font-bold text-yellow-400">+50 分</span>
                </div>
                <div class="border-t border-white/20 pt-2 mt-2">
                  <div class="flex justify-between text-white font-bold">
                    <span>最高可能分數：</span>
                    <span>210 分/題</span>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div>
            <h2 class="text-xl font-bold text-white mb-3">🛠️ 技術架構</h2>
            <div class="grid md:grid-cols-2 gap-4">
              <div class="bg-white/10 rounded-xl p-4">
                <h3 class="text-white font-semibold mb-2">前端技術</h3>
                <ul class="text-white/70 text-sm space-y-1">
                  <li>• Vue 3 + TypeScript</li>
                  <li>• Tailwind CSS</li>
                  <li>• Socket.io-client</li>
                  <li>• Pinia 狀態管理</li>
                </ul>
              </div>
              <div class="bg-white/10 rounded-xl p-4">
                <h3 class="text-white font-semibold mb-2">後端技術</h3>
                <ul class="text-white/70 text-sm space-y-1">
                  <li>• Go + Gin</li>
                  <li>• WebSocket 即時通訊</li>
                  <li>• Redis 快取</li>
                  <li>• PostgreSQL 資料庫</li>
                </ul>
              </div>
            </div>
          </div>
        </div>

        <!-- 版本資訊 -->
        <div class="mt-8 pt-6 border-t border-white/20 text-center">
          <p class="text-white/60 text-sm">
            Version 1.0.0 • 開源專案 • MIT License
          </p>
          <p class="text-white/40 text-xs mt-1">
            Made with ❤️ using modern web technologies
          </p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
// 靜態頁面，無需額外邏輯
</script>

<style scoped>
@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.fade-in {
  animation: fadeIn 0.6s ease-out;
}
</style>