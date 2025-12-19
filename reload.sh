#!/bin/bash
set -e

echo "🚀 开始重载 Go 后端..."

# 以脚本所在目录为项目根目录，避免从其他路径运行导致的相对路径问题
PROJECT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$PROJECT_DIR"

echo "🧱 编译并安装到 ./bin/ ..."
make install

echo "🔁 通过 PM2 重载应用..."
CONFIG="$PROJECT_DIR/ecosystem.config.js"
APP_NAME="GP_back_end_go"

# 检查 PM2 状态，stopped/errored 直接删再 start，online 执行 reload
STATUS="$(pm2 describe "$APP_NAME" 2>/dev/null | awk -F: '/ status/{print $2;exit}' | xargs)"
if [ -z "$STATUS" ]; then
  echo "ℹ️ 未发现 $APP_NAME，尝试 start ..."
  pm2 start "$CONFIG"
elif [ "$STATUS" = "stopped" ] || [ "$STATUS" = "errored" ]; then
  echo "ℹ️ $APP_NAME 状态为 $STATUS，先 delete 再 start config ..."
  pm2 delete "$APP_NAME" || true
  pm2 start "$CONFIG"
else
  pm2 reload "$APP_NAME" || pm2 restart "$APP_NAME" || pm2 start "$CONFIG"
fi

echo "当前 PM2 状态："
pm2 status "$APP_NAME" || true

echo "🎉 Go 后端已重载完成！"

