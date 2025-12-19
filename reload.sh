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

if pm2 describe "$APP_NAME" >/dev/null 2>&1; then
  pm2 reload "$CONFIG" || {
    echo "⚠️  reload 失败，尝试 restart..."
    pm2 restart "$CONFIG"
  }
else
  pm2 start "$CONFIG"
fi

echo "当前 PM2 状态："
pm2 status "$APP_NAME" || true

echo "🎉 Go 后端已重载完成！"

