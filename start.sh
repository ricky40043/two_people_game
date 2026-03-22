#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "🧹 清理舊進程..."
lsof -ti :3333 | xargs kill -9 2>/dev/null || true
lsof -ti :8080 | xargs kill -9 2>/dev/null || true
sleep 1

# 啟動後端
echo "🚀 啟動後端 :8080..."
cd "$SCRIPT_DIR/kahoot-game/backend"
go run cmd/main.go &
BACKEND_PID=$!

# 等後端起來（最多等 10 秒）
echo "⏳ 等待後端啟動..."
for i in $(seq 1 10); do
  if curl -s http://localhost:8080/api/health > /dev/null 2>&1; then
    echo "✅ 後端已就緒"
    break
  fi
  sleep 1
done

# 啟動前端
echo "🚀 啟動前端 :3333..."
cd "$SCRIPT_DIR/kahoot-game/frontend"
npm run dev &
FRONTEND_PID=$!

echo ""
echo "======================================"
echo "  ✅ 本地開發環境啟動完成"
echo "  前端: http://localhost:3333"
echo "  後端: http://localhost:8080"
echo "  按 Ctrl+C 停止所有服務"
echo "======================================"

trap "echo ''; echo '🛑 停止服務...'; kill $BACKEND_PID $FRONTEND_PID 2>/dev/null; exit 0" INT TERM
wait
