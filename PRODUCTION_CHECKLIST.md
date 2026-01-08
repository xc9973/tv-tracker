# 生产环境部署检查清单

## 必需的环境变量

### 1. TMDB API 配置
```bash
TMDB_API_KEY=你的_TMDB_API_密钥
```
- 从 https://www.themoviedb.org/settings/api 获取
- **必须配置**,否则无法搜索和同步剧集

### 2. Telegram Bot 配置 (如果启用)
```bash
TELEGRAM_BOT_TOKEN=你的_Bot_Token
TELEGRAM_CHAT_ID=你的_Chat_ID
TELEGRAM_CHANNEL_ID=频道_ID  # 可选,用于发送日报
REPORT_TIME=09:00  # 可选,默认 08:00
```

获取方式:
- Bot Token: 通过 @BotFather 创建机器人获取
- Chat ID: 通过 @userinfobot 获取个人 ID
- Channel ID: 添加 Bot 到频道,通过 API 获取

### 3. Web API 配置 (如果启用)
```bash
WEB_ENABLED=true
WEB_LISTEN_ADDR=:18080  # 容器内端口
WEB_API_TOKEN=生成一个强密码作为API令牌
```

生成随机 Token:
```bash
openssl rand -base64 32
```

### 4. 数据库配置
```bash
DB_PATH=./data/tv_tracker.db
BACKUP_DIR=./data/backups
```

## 部署模式选择

### 模式 1: Docker Compose (推荐)
适合有 Web 界面需求的场景

```bash
# 1. 复制环境变量模板
cp .env.example .env

# 2. 编辑 .env 文件,填入必要的配置
vim .env

# 3. 启动服务
docker-compose up -d

# 4. 查看日志
docker-compose logs -f

# 5. 访问 Web 界面
# http://your-server-ip
```

环境变量示例:
```bash
TMDB_API_KEY=你的实际密钥
TELEGRAM_BOT_TOKEN=你的实际Token
TELEGRAM_CHAT_ID=你的实际ChatID
TELEGRAM_CHANNEL_ID=你的频道ID
WEB_ENABLED=true
WEB_API_TOKEN=生成的强密码
REPORT_TIME=09:00
```

### 模式 2: 纯 Bot 模式
适合只需要 Telegram 交互的场景

```bash
# 1. 构建二进制
./build.sh

# 2. 设置环境变量
export TMDB_API_KEY=你的密钥
export TELEGRAM_BOT_TOKEN=你的Token
export TELEGRAM_CHAT_ID=你的ChatID
export DISABLE_BOT=false
export WEB_ENABLED=false

# 3. 运行
./tv-tracker
```

### 模式 3: 纯 Web 模式
适合只需要 Web 界面的场景

```bash
export TMDB_API_KEY=你的密钥
export DISABLE_BOT=true
export WEB_ENABLED=true
export WEB_API_TOKEN=强密码
export WEB_LISTEN_ADDR=:18080

./tv-tracker
```

## 部署前检查

- [ ] TMDB_API_KEY 已正确配置
- [ ] 如启用 Bot: TELEGRAM_BOT_TOKEN 和 TELEGRAM_CHAT_ID 已配置
- [ ] 如启用 Web: WEB_API_TOKEN 已生成并配置
- [ ] 数据目录 `./data` 已创建并有写权限
- [ ] 端口 80 (Nginx) 或 18080 (API) 未被占用
- [ ] Docker 和 Docker Compose 已安装(Docker模式)
- [ ] 防火墙已开放必要端口

## 验证部署

### 1. 检查服务状态
```bash
# Docker 模式
docker-compose ps

# 应该看到两个服务都是 Up 状态:
# tv-tracker-api
# tv-tracker-web
```

### 2. 检查日志
```bash
# Docker 模式
docker-compose logs -f tv-tracker-api

# 应该看到:
# - "HTTP API listening on :18080" (如果启用 Web)
# - "TV Tracker bot started" (如果启用 Bot)
# - "Scheduler started" (如果启用 Bot)
```

### 3. 测试 API
```bash
# 健康检查(无需认证)
curl http://localhost/api/health

# 应该返回: {"status":"ok"}

# 测试认证 API
curl -H "Authorization: Bearer $WEB_API_TOKEN" \
     http://localhost/api/dashboard

# 应该返回任务数据
```

### 4. 测试 Bot
在 Telegram 中发送 `/start` 给你的 Bot,应该收到欢迎消息和菜单。

## 常见问题

### 1. "TMDB API calls will fail" 警告
- 检查 TMDB_API_KEY 是否正确设置
- 访问 https://www.themoviedb.org/settings/api 验证密钥

### 2. "Telegram bot not configured" 错误
- 确保设置了 TELEGRAM_BOT_TOKEN 和 TELEGRAM_CHAT_ID
- 或者设置 DISABLE_BOT=true 禁用 Bot

### 3. "WEB_API_TOKEN not set" 错误
- 确保在 .env 中配置了 WEB_API_TOKEN
- 或者设置 WEB_ENABLED=false 禁用 Web

### 4. 前端 401 Unauthorized
- 检查前端环境变量 VITE_API_TOKEN 与后端 WEB_API_TOKEN 是否一致
- 清除浏览器缓存或使用无痕模式

### 5. 端口冲突
```bash
# 查找占用端口的进程
lsof -i :80
lsof -i :18080

# 修改 docker-compose.yml 中的端口映射
ports:
  - "8080:80"  # 使用 8080 代替 80
```

## 备份与恢复

### 自动备份
- 每周日凌晨 3:00 自动备份数据库
- 备份保存在 `data/backups/` 目录
- 默认保留最近 7 天的备份

### 手动备份
```bash
# 通过 API
curl -X POST -H "Authorization: Bearer $WEB_API_TOKEN" \
     http://localhost/api/backup

# 通过 Bot
发送 /backup 命令给 Bot

# 直接复制数据库
cp data/tv_tracker.db data/tv_tracker_backup_$(date +%Y%m%d).db
```

### 恢复数据
```bash
# 1. 停止服务
docker-compose down

# 2. 恢复数据库
cp data/backups/tv_tracker_20260108_030000.db data/tv_tracker.db

# 3. 重启服务
docker-compose up -d
```

## 更新部署

```bash
# 1. 拉取最新代码
git pull

# 2. 重新构建镜像
docker-compose build

# 3. 重启服务
docker-compose down
docker-compose up -d

# 4. 查看日志确认正常
docker-compose logs -f
```

## 监控建议

1. **日志监控**: 定期检查 `docker-compose logs`
2. **磁盘空间**: 监控 `data/` 目录大小
3. **健康检查**: 定期访问 `/api/health` 端点
4. **Bot 响应**: 测试 Bot 命令是否正常响应
5. **备份验证**: 定期验证备份文件是否完整

## 安全建议

1. **更改默认端口**: 避免使用默认的 18080 端口
2. **使用强密码**: WEB_API_TOKEN 至少 32 字符
3. **启用 HTTPS**: 使用 Let's Encrypt 配置 SSL 证书
4. **限制访问**: 通过防火墙限制 API 访问来源
5. **定期更新**: 保持依赖包和基础镜像最新
6. **备份加密**: 对敏感数据进行加密备份

## 性能优化

1. **Nginx 缓存**: 已配置静态资源 7 天缓存
2. **数据库优化**: 定期执行 `VACUUM` 清理数据库
3. **日志轮转**: 配置 Docker 日志轮转避免占用过多空间
4. **资源限制**: 在 docker-compose.yml 中配置内存和 CPU 限制

## 故障排查

如遇到问题:
1. 查看日志: `docker-compose logs -f`
2. 检查配置: 确认环境变量正确
3. 验证网络: 确保容器间可以通信
4. 重启服务: `docker-compose restart`
5. 重建容器: `docker-compose up -d --force-recreate`

## 技术支持

- 架构文档: `ARCHITECTURE.md`
- 部署文档: `DEPLOYMENT.md`
- 使用文档: `USAGE.md`
- Issues: 项目 GitHub Issues 页面
