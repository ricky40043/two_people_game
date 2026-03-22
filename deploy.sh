#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# ============================================================
# 設定區（加到 ~/.zshrc 就不用每次打）
#   export GCP_PROJECT_ID="your-project-id"
# ============================================================
GCP_PROJECT_ID="${GCP_PROJECT_ID:-}"
GCP_REGION="${GCP_REGION:-asia-east1}"
SERVICE_NAME="kahoot-game"
# ============================================================

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; NC='\033[0m'
log()  { echo -e "${GREEN}[deploy]${NC} $1"; }
warn() { echo -e "${YELLOW}[warn]${NC} $1"; }
die()  { echo -e "${RED}[error]${NC} $1"; exit 1; }
step() { echo -e "\n${BLUE}▶ $1${NC}"; }

check_tools() {
  command -v git    > /dev/null || die "需要 git"
  command -v gcloud > /dev/null || die "需要 gcloud CLI：https://cloud.google.com/sdk/docs/install"
}

check_config() {
  [ -z "$GCP_PROJECT_ID" ] && die "請設定 GCP_PROJECT_ID：export GCP_PROJECT_ID=你的專案ID"
}

# ── Step 1：git push
push_to_git() {
  step "Step 1/2：Git Push"
  cd "$SCRIPT_DIR"

  if [ -z "$(git status --porcelain)" ]; then
    log "沒有變更，跳過 git commit"
  else
    DEFAULT_MSG="deploy: $(date '+%Y-%m-%d %H:%M')"
    printf "Commit message（Enter 用預設：$DEFAULT_MSG）：\n> "
    read -r COMMIT_MSG
    [ -z "$COMMIT_MSG" ] && COMMIT_MSG="$DEFAULT_MSG"
    git add -A
    git commit -m "$COMMIT_MSG"
  fi

  git push
  log "✅ Git push 完成"
}

# ── Step 2：建置並部署到 Cloud Run（前後端合一）
deploy() {
  step "Step 2/2：建置並部署（Cloud Run）"

  # build context = kahoot-game/（同時含 frontend/ 和 backend/）
  cd "$SCRIPT_DIR/kahoot-game"

  IMAGE="gcr.io/$GCP_PROJECT_ID/$SERVICE_NAME"

  log "📦 提交 Cloud Build（前後端合一）..."
  gcloud builds submit \
    --tag "$IMAGE" \
    --project "$GCP_PROJECT_ID" \
    --quiet

  log "🚀 部署到 Cloud Run..."
  gcloud run deploy "$SERVICE_NAME" \
    --image "$IMAGE" \
    --platform managed \
    --region "$GCP_REGION" \
    --allow-unauthenticated \
    --session-affinity \
    --min-instances 0 \
    --max-instances 3 \
    --memory 512Mi \
    --cpu 1 \
    --timeout 3600 \
    --set-env-vars "ENV=production,PORT=8080" \
    --project "$GCP_PROJECT_ID" \
    --quiet

  SERVICE_URL=$(gcloud run services describe "$SERVICE_NAME" \
    --region "$GCP_REGION" \
    --project "$GCP_PROJECT_ID" \
    --format "value(status.url)")

  log "✅ 部署完成：$SERVICE_URL"
}

# ============================================================
echo ""
echo "============================================"
echo "  🚀 上版腳本（前後端合一 Cloud Run）"
echo "============================================"

check_tools
check_config

gcloud auth print-access-token > /dev/null 2>&1 || gcloud auth login
gcloud config set project "$GCP_PROJECT_ID" --quiet

push_to_git
deploy

echo ""
echo "============================================"
echo "  ✅ 上版完成！"
echo ""
echo "  網站：$SERVICE_URL"
echo "  （前端 + 後端 + WebSocket 全在同一個 URL）"
echo "============================================"
