#!/bin/bash
# TV Tracker 部署脚本 (增强版)

set -e

BACKUP_DIR="backups/images"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
CURRENT_IMAGE="tv-tracker:current"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "🚀 开始部署 TV Tracker..."

# 1. 环境检查
echo "🔍 检查部署环境..."

# 检查 Docker
if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ 错误: Docker 未安装${NC}"
    echo "   请先安装 Docker: https://docs.docker.com/get-docker/"
    exit 1
fi

# 检查 Docker Compose
if ! command -v docker &> /dev/null || ! docker compose version &> /dev/null; then
    echo -e "${RED}❌ 错误: Docker Compose 未安装或版本过旧${NC}"
    echo "   请安装 Docker Compose V2+"
    exit 1
fi

# 检查 Docker 服务
if ! docker info &> /dev/null; then
    echo -e "${RED}❌ 错误: Docker 服务未运行${NC}"
    echo "   请启动 Docker 服务"
    exit 1
fi

echo -e "${GREEN}✓${NC} Docker 环境检查通过"

# 2. 检查 .env 文件
if [ ! -f .env ]; then
    echo -e "${RED}❌ 错误: 请先创建 .env 文件${NC}"
    echo "   cp .env.example .env"
    echo "   然后填入你的配置"
    exit 1
fi

echo -e "${GREEN}✓${NC} .env 文件检查通过"

# 3. 创建必要目录
echo "📁 创建目录..."
mkdir -p data/backups
mkdir -p $BACKUP_DIR

# 4. 备份当前镜像（如果存在）
if docker images | grep -q "tmdbdingyue-tv-tracker"; then
    echo "💾 备份当前镜像..."
    docker tag tmdbdingyue-tv-tracker:latest $CURRENT_IMAGE || true
    docker save $CURRENT_IMAGE | gzip > "$BACKUP_DIR/tv-tracker_$TIMESTAMP.tar.gz"
    echo -e "${GREEN}✓${NC} 镜像已备份到: $BACKUP_DIR/tv-tracker_$TIMESTAMP.tar.gz"
fi

# 5. 构建新镜像
echo "📦 构建 Docker 镜像..."
if ! docker compose build; then
    echo -e "${RED}❌ 构建失败！${NC}"
    exit 1
fi

# 6. 启动服务
echo "🔄 启动服务..."
if ! docker compose up -d; then
    echo -e "${RED}❌ 启动失败！${NC}"
    echo "尝试回滚到之前的版本..."
    if [ -f "$BACKUP_DIR/tv-tracker_$TIMESTAMP.tar.gz" ]; then
        docker load < "$BACKUP_DIR/tv-tracker_$TIMESTAMP.tar.gz"
        docker tag $CURRENT_IMAGE tmdbdingyue-tv-tracker:latest
        docker compose up -d
        echo -e "${YELLOW}⚠️  已回滚到之前的版本${NC}"
    fi
    exit 1
fi

# 7. 等待服务启动
echo "⏳ 等待服务启动..."
sleep 5

# 8. 健康检查
echo "🏥 执行健康检查..."
if docker compose ps | grep -q "healthy\|running"; then
    echo -e "${GREEN}✓${NC} 服务运行正常"
else
    echo -e "${YELLOW}⚠️  服务可能未完全启动，请检查日志${NC}"
fi

# 9. 清理旧备份（保留最近5个）
echo "🧹 清理旧备份..."
ls -t "$BACKUP_DIR"/tv-tracker_*.tar.gz 2>/dev/null | tail -n +6 | xargs -r rm -f

echo ""
echo -e "${GREEN}✅ 部署完成!${NC}"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📍 访问地址: http://your-server-ip:8318"
echo "📊 查看日志: docker compose logs -f"
echo "🛑 停止服务: docker compose down"
echo "🔙 回滚版本: ./rollback.sh"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "💡 提示: 如需回滚，可使用最近的备份:"
ls -t "$BACKUP_DIR"/tv-tracker_*.tar.gz 2>/dev/null | head -n 3 || true
