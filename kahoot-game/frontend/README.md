# 🎮 Kahoot 風格多人遊戲 - 前端

Vue 3 + TypeScript + Tailwind CSS 構建的現代化遊戲前端

## ✨ 特色功能

- 🎯 **雙端設計** - 主持人大螢幕 + 玩家手機端
- 📱 **QR Code 加入** - 掃描即可快速加入遊戲
- 🔄 **主角輪替制** - 每題輪流當主角朗讀
- ⚡ **即時同步** - WebSocket 即時通訊
- 🎨 **現代化 UI** - 美觀的漸變設計
- 📊 **即時計分** - 動態排行榜和分數動畫

## 🛠️ 技術棧

- **前端框架**: Vue 3 + Composition API
- **類型支援**: TypeScript
- **狀態管理**: Pinia
- **路由管理**: Vue Router
- **樣式框架**: Tailwind CSS
- **建構工具**: Vite
- **即時通訊**: Socket.io-client

## 🚀 快速開始

### 安裝依賴
```bash
npm install
```

### 開發伺服器
```bash
npm run dev
```
訪問 `http://localhost:5173`

### 建構生產版本
```bash
npm run build
```

### 預覽生產版本
```bash
npm run preview
```

## 📁 專案結構

```
src/
├── components/          # 可復用組件
├── views/              # 頁面組件
├── stores/             # Pinia 狀態管理
├── router/             # Vue Router 配置
├── types/              # TypeScript 類型定義
├── utils/              # 工具函數
├── assets/             # 靜態資源
├── App.vue             # 根組件
├── main.ts             # 應用入口
└── style.css           # 全域樣式
```

## 🎮 頁面說明

### 主要頁面
- **HomeView** - 主頁，選擇創建或加入房間
- **CreateRoomView** - 創建房間，設置遊戲參數
- **JoinRoomView** - 加入房間，輸入房間代碼
- **LobbyView** - 等待大廳，顯示玩家列表
- **GameHostView** - 主持人視角，控制遊戲進行
- **GamePlayerView** - 玩家視角，答題介面
- **ResultsView** - 遊戲結果，顯示排行榜

### 狀態管理
- **gameStore** - 遊戲狀態（房間、玩家、題目、分數）
- **socketStore** - WebSocket 連線和訊息處理
- **uiStore** - UI 狀態（載入、錯誤提示等）

## 🔗 WebSocket 事件

### 發送事件
- `CREATE_ROOM` - 創建房間
- `JOIN_ROOM` - 加入房間
- `START_GAME` - 開始遊戲
- `SUBMIT_ANSWER` - 提交答案
- `LEAVE_ROOM` - 離開房間

### 接收事件
- `ROOM_CREATED` - 房間創建成功
- `PLAYER_JOINED` - 玩家加入
- `GAME_STARTED` - 遊戲開始
- `NEW_QUESTION` - 新題目
- `SCORES_UPDATE` - 分數更新
- `GAME_FINISHED` - 遊戲結束

## 🎨 自定義樣式

### CSS 變數
```css
:root {
  --primary-gradient: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  --success-color: #10b981;
  --error-color: #ef4444;
}
```

### 自定義類別
- `.btn` - 按鈕基礎樣式
- `.card` - 卡片容器
- `.input` - 輸入框樣式
- `.fade-in` - 淡入動畫
- `.slide-in` - 滑入動畫

## 📱 響應式設計

支援多種設備尺寸：
- 📱 手機: 375px+
- 📟 平板: 768px+
- 💻 桌面: 1024px+

## 🔧 環境配置

### 開發環境
```env
# 預設會依據瀏覽器所在的 IP 自動連線到 :8080
# 只有在自訂埠或協定時才需要設定
VITE_API_PORT=8080
VITE_API_PROTOCOL=http
VITE_WS_PORT=8080
VITE_WS_PROTOCOL=ws
```

### 生產環境
```env
# 指向正式環境的 API / WebSocket
VITE_API_URL=https://api.your-domain.com
VITE_WS_URL=wss://api.your-domain.com/ws

# 若部署在子路徑 (例如 https://your-domain.com/kahoot)
VITE_BASE_PATH=/kahoot/
```

## 🚀 部署

### Vercel 部署
```bash
npm run build
vercel --prod
```

### Netlify 部署
```bash
npm run build
# 上傳 dist 資料夾
```

## 🐛 開發注意事項

1. **WebSocket 連線** - 確保後端服務器正在運行
2. **CORS 設定** - 確保後端允許前端域名
3. **瀏覽器支援** - 需要支援 ES2020+ 的現代瀏覽器
4. **觸控優化** - 按鈕大小適合手機觸控

## 📈 性能優化

- ✅ 路由懶載入
- ✅ 組件代碼分割
- ✅ 圖片延遲載入
- ✅ WebSocket 連線復用
- ✅ 狀態管理優化

## 🤝 開發指南

### 添加新頁面
1. 在 `src/views/` 創建組件
2. 在 `src/router/index.ts` 添加路由
3. 在 `src/types/index.ts` 定義相關類型

### 添加新狀態
1. 在相應的 store 文件中添加狀態
2. 定義 actions 和 getters
3. 在組件中使用 store

### 添加 WebSocket 事件
1. 在 `socketStore` 中添加事件監聽
2. 在 `gameStore` 中更新相關狀態
3. 在組件中響應狀態變化
