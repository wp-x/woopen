#!/usr/bin/env bash
# WoOpen 一键更新脚本：拉代码 -> 备份库 -> 重建镜像 -> 重启 -> 看日志
# 在项目根目录执行： bash deploy/update.sh
set -euo pipefail

cd "$(dirname "$0")/.."   # 切到项目根目录

# 兼容新旧 compose 命令
if docker compose version >/dev/null 2>&1; then
  DC="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
  DC="docker-compose"
else
  echo "未找到 docker compose，请先安装 Docker" >&2
  exit 1
fi

echo "[1/5] 拉取最新代码"
git pull --ff-only

echo "[2/5] 备份数据库（数据挂载在 ./data，重建不丢）"
if [ -f data/woopen.db ]; then
  cp -f data/woopen.db "data/woopen.db.bak.$(date +%Y%m%d%H%M%S)"
fi

echo "[3/5] 重建并重启容器"
$DC up -d --build

echo "[4/5] 清理悬空镜像"
docker image prune -f >/dev/null 2>&1 || true

echo "[5/5] 最近日志"
$DC logs --tail=30 woopen
echo "更新完成。迁移会在启动时自动执行（幂等），直连相关列已就绪。"
