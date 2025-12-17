#!/bin/bash
set -e

echo "🚀 开始重载 Go 后端..."

# 以脚本所在目录为项目根目录，避免从其他路径运行导致的相对路径问题
PROJECT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$PROJECT_DIR"

echo "🧱 编译并安装到 ./bin/ ..."
make install

echo "🔁 通过 PM2 重载应用..."
pm2 startOrReload ecosystem.config.js

echo "🎉 Go 后端已重载完成！"

