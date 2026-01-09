#!/bin/bash

echo "====== TV Tracker 容器诊断工具 ======"
echo ""

# 检查容器状态
echo "1. 容器状态:"
docker ps -a | grep tv-tracker
echo ""

# 查看最近的容器日志
echo "2. 最近的容器日志 (最后50行):"
docker logs tv-tracker --tail 50
echo ""

# 检查环境变量文件
echo "3. 检查 .env 文件是否存在:"
if [ -f .env ]; then
    echo "✓ .env 文件存在"
    echo ""
    echo "环境变量配置 (隐藏敏感信息):"
    cat .env | grep -v "^#" | grep -v "^$" | sed 's/=.*/=***/'
else
    echo "✗ .env 文件不存在！"
    echo ""
    echo "请创建 .env 文件，参考 .env.example:"
    cat .env.example
fi
echo ""

# 检查关键环境变量
echo "4. 检查容器内环境变量:"
docker exec tv-tracker env 2>/dev/null | grep -E "TMDB_API_KEY|WEB_API_TOKEN|WEB_ENABLED" || echo "容器未运行或无法访问"
echo ""

# 提供修复建议
echo "====== 修复建议 ======"
echo ""
echo "如果容器不断重启，请检查:"
echo "1. 确保 .env 文件存在并包含 TMDB_API_KEY"
echo "2. 确保 WEB_ENABLED=true 时 WEB_API_TOKEN 已设置"
echo "3. 查看上面的容器日志了解具体错误"
echo ""
echo "快速修复步骤:"
echo "  cp .env.example .env"
echo "  vim .env  # 编辑并填写必需的配置"
echo "  docker compose down"
echo "  docker compose up -d"
echo ""
