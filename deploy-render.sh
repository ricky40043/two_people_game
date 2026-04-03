#!/bin/bash
# 部署到 Render 前的本地測試腳本
# 用法：
#   ./deploy-render.sh build   → 在本地 build Docker image 測試
#   ./deploy-render.sh run     → 在本地跑 container（port 8080）
#   ./deploy-render.sh push    → git push 觸發 Render 自動部署

set -e

IMAGE_NAME="kahoot-game"
CONTEXT="./kahoot-game"
DOCKERFILE="./kahoot-game/Dockerfile"

case "$1" in
  build)
    echo "🔨 Building Docker image..."
    docker build -t "$IMAGE_NAME" -f "$DOCKERFILE" "$CONTEXT"
    echo "✅ Build 成功：$IMAGE_NAME"
    ;;

  run)
    echo "🚀 Starting container on http://localhost:8080 ..."
    docker run --rm \
      -p 8080:8080 \
      -e PORT=8080 \
      -e ENV=production \
      -e GIN_MODE=release \
      -e DB_HOST=skip \
      "$IMAGE_NAME"
    ;;

  push)
    echo "📤 Pushing to git → Render auto-deploy..."
    git add -A
    git status
    read -rp "Commit message: " msg
    git commit -m "$msg"
    git push
    echo "✅ 推送完成，前往 Render Dashboard 查看部署狀態"
    ;;

  *)
    echo "用法: $0 {build|run|push}"
    echo ""
    echo "  build  → 本地 build Docker image 驗證"
    echo "  run    → 本地跑 container（port 8080）"
    echo "  push   → git push 觸發 Render 自動部署"
    exit 1
    ;;
esac
