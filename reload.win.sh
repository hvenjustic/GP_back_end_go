#!/usr/bin/env bash
set -e

echo "🚀 开始重载 Go 后端 (Windows)..."

PROJECT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$PROJECT_DIR"

echo "🧱 编译并安装到 ./bin/ ..."
mkdir -p bin
go build -ldflags="-w -s" -o bin/back_end_go.exe .

echo "🔁 通过 PM2 重载应用..."
CONFIG="$PROJECT_DIR/ecosystem.config.win.js"
APP_NAME="GP_back_end_go"

STATUS="$(pm2 describe "$APP_NAME" 2>/dev/null | awk -F: '/ status/{print $2;exit}' | xargs)"
case "$STATUS" in
  online)
    pm2 reload "$APP_NAME" || pm2 restart "$APP_NAME" || { pm2 delete "$APP_NAME" || true; pm2 start "$CONFIG"; }
    ;;
  stopped|errored)
    echo "ℹ️ $APP_NAME 状态为 $STATUS，先 delete 再 start config ..."
    pm2 delete "$APP_NAME" || true
    pm2 start "$CONFIG"
    ;;
  *)
    echo "ℹ️ 未发现 $APP_NAME，先清理后 start ..."
    pm2 delete "$APP_NAME" || true
    pm2 start "$CONFIG"
    ;;
esac

echo "当前 PM2 状态："
pm2 status "$APP_NAME" || true

echo "🎉 Go 后端已重载完成！"
