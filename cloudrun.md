# Cloud Run 部署指南

## 為什麼選 Cloud Run

| 項目 | Cloud Run | Render Free |
|------|-----------|-------------|
| WebSocket | ✅ 完整支援 | ❌ 30秒強制斷線 |
| 費用 | 沒人用 = $0 | 沒人用還是 $0，但有限制 |
| 免費額度 | 每月 180,000 vCPU-s（很夠） | 750 小時/月（夠，但會 sleep） |
| Cold Start | 首次連線慢 1-3 秒 | 有 Sleep，15分鐘後冷啟動更慢 |
| 設定難度 | 中（需要 gcloud CLI） | 簡單 |

## 費用試算（Side Project 情境）

假設每週玩 3 次，每次 2 小時，5 個房間 × 15 人：
- 每月 6 小時 × 0.5 vCPU = 10,800 vCPU-s
- 免費額度 180,000 vCPU-s/月
- **= 完全免費** ✅

## 部署架構

```
前端 (Vercel 免費) ──────────────────────────────┐
                                                  │
後端 (Cloud Run 免費) ◄── WebSocket / REST API ───┘
  │
  └── 記憶體模式 (不需要 Redis/PostgreSQL)
```

前端推薦用 **Vercel**（免費，靜態網站無限制），後端用 **Cloud Run**。

## 前置條件

1. 安裝 [Google Cloud CLI](https://cloud.google.com/sdk/docs/install)
2. 安裝 Docker Desktop
3. 建立 Google Cloud 專案

```bash
# 登入
gcloud auth login

# 設定專案（換成你的專案 ID）
gcloud config set project YOUR_PROJECT_ID

# 啟用必要 API
gcloud services enable run.googleapis.com
gcloud services enable cloudbuild.googleapis.com
```

## 部署後端

```bash
cd kahoot-game/backend

# 設定變數
PROJECT_ID="your-gcp-project-id"
REGION="asia-east1"          # 台灣最近的區域
SERVICE_NAME="kahoot-game-backend"
FRONTEND_URL="https://your-vercel-app.vercel.app"

# Build & Push Docker image
gcloud builds submit \
  --tag gcr.io/$PROJECT_ID/$SERVICE_NAME \
  --project $PROJECT_ID

# 部署到 Cloud Run
gcloud run deploy $SERVICE_NAME \
  --image gcr.io/$PROJECT_ID/$SERVICE_NAME \
  --platform managed \
  --region $REGION \
  --allow-unauthenticated \
  --session-affinity \
  --min-instances 0 \
  --max-instances 3 \
  --memory 256Mi \
  --cpu 1 \
  --timeout 3600 \
  --set-env-vars "ENV=production,PORT=8080,FRONTEND_URL=$FRONTEND_URL,CORS_ORIGINS=$FRONTEND_URL"
```

> `--timeout 3600` = 允許連線保持 1 小時（WebSocket 需要）
> `--session-affinity` = 同一個 client 的請求路由到同一個 instance（WebSocket 需要）
> `--min-instances 0` = 沒人用就縮到 0，不收費

## 部署前端（Vercel）

1. 把 `kahoot-game/frontend` push 到 GitHub
2. 在 [Vercel](https://vercel.com) 連結 GitHub repo
3. 設定 Build 設定：
   - **Root Directory**: `kahoot-game/frontend`
   - **Build Command**: `npm run build`
   - **Output Directory**: `dist`
4. 設定環境變數：
   - `VITE_API_URL` = Cloud Run 後端 URL（部署後取得）
   - `VITE_WS_URL` = `wss://your-cloudrun-url.run.app/ws`

## 部署流程（第一次）

```
1. 部署後端到 Cloud Run → 取得後端 URL
2. 把後端 URL 填到 Vercel 的環境變數
3. 部署前端到 Vercel → 取得前端 URL
4. 把前端 URL 更新到 Cloud Run 環境變數（FRONTEND_URL, CORS_ORIGINS）
5. 重新部署後端
```

## 更新部署（之後每次）

```bash
# 只需要重跑 build & deploy
cd kahoot-game/backend
gcloud builds submit --tag gcr.io/$PROJECT_ID/$SERVICE_NAME
gcloud run deploy $SERVICE_NAME --image gcr.io/$PROJECT_ID/$SERVICE_NAME --region $REGION
```

## Cold Start 說明

第一個人連線時，如果 instance 是 0，會等 1-3 秒啟動。
這對遊戲影響不大：主持人建房間時可能多等 1-3 秒，之後玩家加入就正常了。

如果介意，可以設 `--min-instances 1`，但這樣每月約 $10。
