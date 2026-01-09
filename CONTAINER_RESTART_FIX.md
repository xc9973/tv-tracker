# 容器重启问题排查与修复

## 🔍 问题现象

容器状态显示: `Restarting (1) XX seconds ago`

## 📋 排查步骤

### 1. 运行诊断脚本

```bash
cd /opt/tv-tracker
chmod +x diagnose.sh
./diagnose.sh
```

诊断脚本会检查:
- 容器状态
- 容器日志
- .env 文件配置
- 环境变量设置

### 2. 查看容器日志

```bash
docker logs tv-tracker --tail 100
```

关键错误信息:
- `TMDB_API_KEY is required but not set` - 缺少 TMDB API Key
- `WEB_API_TOKEN is required when WEB_ENABLED=true` - 缺少 Web API Token

---

## ✅ 解决方案

### 方案 1: 检查并更新 .env 文件

```bash
cd /opt/tv-tracker

# 检查 .env 文件是否存在
ls -la .env

# 如果不存在，从示例文件复制
cp .env.example .env

# 编辑 .env 文件
vim .env
```

**必须配置的环境变量**:

```bash
# 必需 - TMDB API Key
TMDB_API_KEY=your_actual_tmdb_api_key_here

# 如果启用 Web 界面（必需）
WEB_ENABLED=true
WEB_API_TOKEN=your_secret_token_here

# Telegram 配置（可选，如果禁用 bot）
DISABLE_BOT=false  # 如果不使用 Telegram bot，设为 true
TELEGRAM_BOT_TOKEN=your_bot_token
TELEGRAM_CHAT_ID=your_chat_id
```

### 方案 2: 只使用 Web 功能，禁用 Telegram

如果你只想使用 Web 界面，不需要 Telegram 功能:

```bash
# .env 文件内容
TMDB_API_KEY=your_actual_api_key
WEB_ENABLED=true
WEB_API_TOKEN=your_secret_token
DISABLE_BOT=true
```

### 方案 3: 临时禁用配置验证（不推荐）

如果你想暂时跳过验证进行调试，可以修改代码中的验证逻辑。但**不推荐**这样做，因为会导致运行时错误。

---

## 🚀 重启容器

配置完成后，重启容器:

```bash
cd /opt/tv-tracker

# 停止容器
docker compose down

# 重新构建并启动
docker compose up -d

# 检查状态
docker ps

# 查看日志
docker logs -f tv-tracker
```

### 预期结果

容器正常运行时的日志应该包含:

```json
{"level":"info","ts":...,"msg":"Database initialized","path":"/app/data/tv_tracker.db"}
{"level":"info","ts":...,"msg":"HTTP API listening","address":":18080"}
{"level":"info","ts":...,"msg":"Telegram bot disabled","disable_bot":true}
```

或如果启用了 Telegram:

```json
{"level":"info","ts":...,"msg":"Telegram bot initialized","chat_id":123456}
{"level":"info","ts":...,"msg":"Scheduler started","report_time":"09:00"}
```

---

## 🔧 常见问题

### Q1: 容器启动后立即退出

**原因**: 配置验证失败

**解决**: 
1. 运行 `docker logs tv-tracker` 查看具体错误
2. 补充缺失的环境变量
3. 重启容器

### Q2: .env 文件存在但容器仍然重启

**原因**: docker-compose.yml 没有正确读取 .env 文件

**解决**:
```bash
# 检查 docker-compose.yml 是否正确
cat docker-compose.yml | grep -A 10 environment

# 手动指定 .env 文件
docker compose --env-file .env up -d
```

### Q3: 在服务器上看到 "No services to build" 警告

**原因**: Docker compose 使用了已有的镜像

**解决**: 这是正常的，如果你之前已经构建过镜像。如果需要强制重新构建:
```bash
docker compose build --no-cache
docker compose up -d
```

### Q4: 环境变量设置了但仍然报错

**原因**: 
- 环境变量格式错误（有空格、引号等）
- .env 文件编码问题

**解决**:
```bash
# 检查 .env 文件内容
cat -A .env | head -20

# 正确格式:
TMDB_API_KEY=abc123  # ✓ 正确
TMDB_API_KEY = abc123  # ✗ 错误（有空格）
TMDB_API_KEY="abc123"  # ✗ 不推荐（有引号）
```

---

## 📞 获取帮助

如果以上方法都无法解决问题:

1. 收集诊断信息:
   ```bash
   ./diagnose.sh > diagnostic_report.txt
   ```

2. 检查 GitHub Issues 或创建新 Issue

3. 提供以下信息:
   - 容器日志 (隐藏敏感信息)
   - docker-compose.yml 配置
   - .env 文件配置 (隐藏 API Key)
   - 诊断脚本输出

---

## ✅ 验证修复

容器正常运行后:

```bash
# 1. 检查容器状态（应该是 Up）
docker ps | grep tv-tracker

# 2. 检查健康状态
docker inspect tv-tracker | grep -A 5 Health

# 3. 测试 API
curl -H "X-API-Token: your_token" http://localhost:8318/api/health

# 4. 访问 Web 界面
# 打开浏览器访问: http://your-server-ip:8318
```

成功！🎉
