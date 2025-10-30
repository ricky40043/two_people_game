#!/bin/bash

echo "🧪 Ricky 遊戲小舖後端測試設置腳本"
echo "================================"

# 檢查 Go 是否安裝
if ! command -v go &> /dev/null; then
    echo "❌ Go 未安裝，請先安裝 Go 1.21+"
    exit 1
fi

echo "✅ Go 版本: $(go version)"

# 檢查是否在正確目錄
if [ ! -f "go.mod" ]; then
    echo "❌ 請在 kahoot-game/backend 目錄下執行此腳本"
    exit 1
fi

# 下載依賴
echo "📦 下載 Go 依賴..."
go mod tidy

# 檢查 Docker 是否安裝（用於啟動資料庫）
if command -v docker &> /dev/null; then
    echo "✅ Docker 已安裝，可以啟動本地資料庫"
    
    echo "🚀 啟動 Redis..."
    docker run -d --name kahoot-redis -p 6379:6379 redis:alpine 2>/dev/null || echo "Redis 容器已存在"
    
    echo "🚀 啟動 PostgreSQL..."
    docker run -d --name kahoot-postgres -p 5432:5432 -e POSTGRES_PASSWORD=password -e POSTGRES_DB=kahoot_game postgres:15 2>/dev/null || echo "PostgreSQL 容器已存在"
    
    echo "⏳ 等待資料庫啟動..."
    sleep 5
    
else
    echo "⚠️ Docker 未安裝，請手動設置 Redis 和 PostgreSQL"
    echo "Redis: localhost:6379"
    echo "PostgreSQL: localhost:5432, DB: kahoot_game, Password: password"
fi

# 創建 .env 文件（如果不存在）
if [ ! -f ".env" ]; then
    echo "📝 創建 .env 配置文件..."
    cp .env .env.backup 2>/dev/null || true
    cat > .env << EOF
# 🔧 Ricky 遊戲小舖後端環境配置

# 服務器設定
PORT=8080
HOST=localhost
ENV=development

# Redis 設定 (快取層)
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# PostgreSQL 設定 (持久化)
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=kahoot_game
DB_SSLMODE=disable

# JWT 設定 (未來用於認證)
JWT_SECRET=your-super-secret-jwt-key-change-in-production

# CORS 設定
CORS_ORIGINS=http://localhost:5173,http://localhost:3000,https://your-frontend-domain.vercel.app

# WebSocket 設定
WS_READ_BUFFER_SIZE=1024
WS_WRITE_BUFFER_SIZE=1024
WS_MAX_MESSAGE_SIZE=512

# 遊戲設定
MAX_PLAYERS_PER_ROOM=20
ROOM_ID_LENGTH=6
QUESTION_TIME_LIMIT=30
DEFAULT_TOTAL_QUESTIONS=10

# 日誌設定
LOG_LEVEL=debug
LOG_FORMAT=json
EOF
fi

echo ""
echo "✅ 設置完成！"
echo ""
echo "🚀 啟動服務器："
echo "   go run cmd/main.go"
echo ""
echo "🌐 API 端點："
echo "   健康檢查: http://localhost:8080/api/health"
echo "   活躍遊戲: http://localhost:8080/api/games"
echo "   隨機題目: http://localhost:8080/api/questions/random/5"
echo ""
echo "📡 WebSocket："
echo "   連線端點: ws://localhost:8080/ws"
echo ""
echo "🗄️ 資料庫："
echo "   Redis: localhost:6379"
echo "   PostgreSQL: localhost:5432"