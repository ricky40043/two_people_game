#!/bin/bash

# 確保腳本在錯誤時停止
set -e

echo "🚀 準備上傳程式碼到 GitHub..."

# 顯示當前狀態
echo "📊 當前檔案狀態："
git status -s

# 加入所有變更
echo "➕ 加入所有變更..."
git add .

# 詢問 Commit 訊息
echo ""
read -p "📝 請輸入 Commit 訊息 (直接按 Enter 使用預設訊息 'Update'): " commit_msg

# 如果使用者沒輸入，使用預設訊息
if [ -z "$commit_msg" ]; then
    commit_msg="Update"
fi

# 提交
echo "💾 正在提交..."
git commit -m "$commit_msg"

# 推送
echo "⬆️ 正在推送到 GitHub..."
git push

echo ""
echo "✅ 上傳完成！Render 將會自動開始部署。"
