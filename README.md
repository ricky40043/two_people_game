# 🎮 Kahoot 風格多人即時問答遊戲

這是一個即時互動的多人問答遊戲平台，類似 Kahoot!。玩家可以透過手機或電腦加入遊戲房間，即時回答問題並與他人競爭。

## ✨ 特色功能

- **即時互動**: 使用 WebSocket 實現毫秒級的即時通訊。
- **多人對戰**: 支援多人同時在線遊玩，即時同步分數與排名。
- **遊戲大廳**: 創建房間、加入房間、等待大廳。
- **完整遊戲流程**: 倒數計時、題目展示、即時結算、排行榜。
- **現代化介面**: 使用 Vue 3 + TailwindCSS 打造流暢的使用者體驗。

## 🛠️ 技術架構

### Backend (後端)
- **語言**: Go (Golang) 1.21+
- **框架**: Gin Web Framework
- **WebSocket**: Gorilla WebSocket
- **資料庫**: PostgreSQL (Supabase)
- **快取/即時狀態**: Redis
- **架構**: 模組化設計 (Handlers, Services, Repositories)

### Frontend (前端)
- **框架**: Vue 3 (Composition API)
- **建置工具**: Vite
- **狀態管理**: Pinia
- **樣式**: TailwindCSS
- **通訊**: Native WebSocket API

### 部署 (Infrastructure)
- **平台**: Render.com
- **配置**: Infrastructure as Code (`render.yaml`)
- **服務**:
    - **Web Service**: Go Backend
    - **Static Site**: Vue Frontend
    - **Redis**: Render Managed Redis
    - **Database**: Supabase PostgreSQL (外部連線)

## 🚀 本地開發指南

### 前置需求
- Go 1.21+
- Node.js 18+
- Redis (本地或 Docker)
- PostgreSQL (本地或 Docker)

### 1. 後端設定 (`kahoot-game/backend`)

```bash
cd kahoot-game/backend

# 複製環境變數範例
cp .env.example .env
# (請依據您的本地環境修改 .env 內的 DB 和 Redis 設定)

# 安裝依賴
go mod download

# 啟動伺服器
go run cmd/main.go
```

### 2. 前端設定 (`kahoot-game/frontend`)

```bash
cd kahoot-game/frontend

# 安裝依賴
npm install

# 啟動開發伺服器
npm run dev
```

## 📦 部署指南 (Render)

本專案已配置 `render.yaml`，支援 Render Blueprint 自動部署。

1. 將程式碼推送到 GitHub。
2. 在 [Render Dashboard](https://dashboard.render.com/) 建立新的 **Blueprint**。
3. 連結此 Repository。
4. Render 會自動偵測並建立以下服務：
    - `kahoot-game-backend`
    - `kahoot-game-frontend`
    - `kahoot-game-redis`
5. **重要**: 在 Render Dashboard 的 Backend 服務中，確認 `DATABASE_URL` 環境變數已設定為您的 Supabase 連線字串。

## 📁 專案結構

```
.
├── kahoot-game/
│   ├── backend/           # Go 後端原始碼
│   │   ├── cmd/          # 程式進入點
│   │   ├── internal/     # 核心邏輯 (Handlers, Models, Services)
│   │   └── ...
│   └── frontend/          # Vue 前端原始碼
│       ├── src/          # 頁面與元件
│       └── ...
├── render.yaml            # Render 部署配置 (Blueprint)
└── README.md              # 專案說明文件
```