#!/bin/bash

SUDO=""
if command -v sudo >/dev/null 2>&1; then
	SUDO="sudo"
fi

echo "====== 修复 TV Tracker 容器权限问题 ======"
echo ""

# 停止容器
echo "1. 停止容器..."
docker compose down

# 检查 data 目录
echo ""
echo "2. 检查 data 目录..."
ls -ld ./data

# 修复权限 - 容器内使用 uid 1000
echo ""
echo "3. 修复权限（设置为 uid 1000）..."
$SUDO chown -R 1000:1000 ./data
$SUDO chmod -R 755 ./data

echo ""
echo "权限修复后:"
ls -ld ./data
ls -l ./data/ 2>/dev/null || echo "目录为空"

# 重新启动
echo ""
echo "4. 重新启动容器..."
docker compose up -d

# 等待启动
echo ""
echo "等待容器启动..."
sleep 3

# 检查状态
echo ""
echo "5. 检查容器状态..."
docker ps | grep tv-tracker

# 查看日志
echo ""
echo "6. 查看日志 (前20行)..."
docker logs tv-tracker --tail 20

echo ""
echo "====== 完成 ======"
echo ""
echo "如果看到 'Database initialized' 和 'HTTP API listening'，则修复成功！"
echo "如果仍有错误，请运行: docker logs -f tv-tracker"
echo ""
