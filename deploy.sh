#!/bin/bash
# TV Tracker 部署脚本

set -e

echo "🚀 开始部署 TV Tracker..."

# 检查 .env 文件
if [ ! -f .env ]; then
    echo "❌ 错误: 请先创建 .env 文件"
    echo "   cp .env.example .env"
    echo "   然后填入你的 API Key"
    exit 1
fi

# 创建数据目录
mkdir -p data

# 构建并启动
echo "📦 构建 Docker 镜像..."
docker compose build

echo "🔄 启动服务..."
docker compose up -d

echo ""
echo "✅ 部署完成!"
echo ""
echo "📍 访问地址: http://your-server-ip:8080"
echo "📊 查看日志: docker compose logs -f"
echo "🛑 停止服务: docker compose down"
