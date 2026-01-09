#!/bin/bash
# TV Tracker 回滚脚本

BACKUP_DIR="backups/images"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "🔙 TV Tracker 回滚工具"
echo ""

# 检查备份目录
if [ ! -d "$BACKUP_DIR" ]; then
    echo -e "${RED}❌ 错误: 备份目录不存在${NC}"
    exit 1
fi

# 列出可用备份
echo "📦 可用的备份版本:"
echo ""
backups=($(ls -t "$BACKUP_DIR"/tv-tracker_*.tar.gz 2>/dev/null))

if [ ${#backups[@]} -eq 0 ]; then
    echo -e "${RED}❌ 没有找到任何备份${NC}"
    exit 1
fi

# 显示备份列表
for i in "${!backups[@]}"; do
    backup_file="${backups[$i]}"
    backup_name=$(basename "$backup_file")
    backup_date=$(echo "$backup_name" | grep -oP '\d{8}_\d{6}')
    formatted_date=$(echo "$backup_date" | sed 's/\([0-9]\{4\}\)\([0-9]\{2\}\)\([0-9]\{2\}\)_\([0-9]\{2\}\)\([0-9]\{2\}\)\([0-9]\{2\}\)/\1-\2-\3 \4:\5:\6/')
    echo "  [$i] $formatted_date"
done

echo ""
echo -n "请选择要回滚的版本 [0-$((${#backups[@]}-1))]: "
read -r selection

# 验证输入
if ! [[ "$selection" =~ ^[0-9]+$ ]] || [ "$selection" -ge ${#backups[@]} ]; then
    echo -e "${RED}❌ 无效的选择${NC}"
    exit 1
fi

selected_backup="${backups[$selection]}"

echo ""
echo -e "${YELLOW}⚠️  即将回滚到: $(basename "$selected_backup")${NC}"
echo -n "确认回滚? [y/N]: "
read -r confirm

if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
    echo "已取消"
    exit 0
fi

# 执行回滚
echo ""
echo "🔄 开始回滚..."

# 1. 停止当前服务
echo "1️⃣  停止当前服务..."
docker compose down

# 2. 加载备份镜像
echo "2️⃣  加载备份镜像..."
if ! docker load < "$selected_backup"; then
    echo -e "${RED}❌ 加载备份失败${NC}"
    exit 1
fi

# 3. 重新标记镜像
echo "3️⃣  重新标记镜像..."
docker tag tv-tracker:current tmdbdingyue-tv-tracker:latest

# 4. 启动服务
echo "4️⃣  启动服务..."
if ! docker compose up -d; then
    echo -e "${RED}❌ 启动服务失败${NC}"
    exit 1
fi

# 5. 等待服务启动
echo "⏳ 等待服务启动..."
sleep 5

# 6. 健康检查
echo "🏥 执行健康检查..."
if docker compose ps | grep -q "healthy\|running"; then
    echo -e "${GREEN}✓${NC} 服务运行正常"
else
    echo -e "${YELLOW}⚠️  服务可能未完全启动，请检查日志${NC}"
fi

echo ""
echo -e "${GREEN}✅ 回滚完成!${NC}"
echo ""
echo "📊 查看日志: docker compose logs -f"
