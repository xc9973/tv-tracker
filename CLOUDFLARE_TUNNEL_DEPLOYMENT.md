# Cloudflare Tunnel 部署指南

## 📋 当前架构分析

### 镜像架构：单体应用（单镜像）

**当前配置采用的是单镜像部署方案**，所有功能集成在一个容器中：

- **后端服务**：Go 应用（包含 API 和 Telegram Bot）
- **前端资源**：静态 HTML 文件（`web/simple/index.html`）
- **数据库**：SQLite（通过 Docker Volume 持久化）

### 优点

✅ **部署简单** - 只需管理一个容器  
✅ **资源占用低** - 无需额外的 Nginx 容器  
✅ **配置简单** - 环境变量集中管理  
✅ **性能良好** - 静态文件由 Go 直接服务，性能足够  

### 缺点

❌ **扩展性受限** - 无法独立扩展前端和后端  
❌ **灵活性较低** - 前端更新需要重建整个镜像  

---

## 🚀 通过 Cloudflare Tunnel 暴露服务

### 方案概述

使用 Cloudflare Tunnel 将本地服务安全地暴露到公网，无需开放路由器端口。

### 端口配置

当前配置：
```yaml
ports:
  - "8318:18080"  # 宿主机端口 8318 映射到容器内 18080
```

**说明**：
- 容器内监听端口：`18080`（由 `WEB_LISTEN_ADDR` 配置）
- 宿主机暴露端口：`8318`
- Cloudflare Tunnel 将连接到宿主机的 `8318` 端口

---

## 📝 Cloudflare Tunnel 配置步骤

### 方法一：使用 cloudflared（推荐）

#### 1. 安装 cloudflared

**macOS**:
```bash
brew install cloudflare/cloudflare/cloudflared
```

**Linux**:
```bash
# Ubuntu/Debian
wget -q https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64.deb
sudo dpkg -i cloudflared-linux-amd64.deb

# CentOS/RHEL
rpm -i https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-x86_64.rpm

# 验证安装
cloudflared --version
```

**Docker**（推荐用于服务器）:
```bash
docker pull cloudflare/cloudflared:latest
```

#### 2. 登录 Cloudflare 账户

```bash
cloudflared tunnel login
```

这会打开浏览器，让您选择要使用的域名和授权的 Zone。

#### 3. 创建 Tunnel

```bash
cloudflared tunnel create tv-tracker
```

**输出示例**：
```
Tunnel credentials written to /home/user/.cloudflared/[TUNNEL_ID].json
cloudflared chose a random ID for this tunnel: [TUNNEL_ID]
```

**重要**：保存返回的 Tunnel ID，后续配置会用到。

#### 4. 配置 Tunnel

创建配置文件 `~/.cloudflared/config.yml`：

```yaml
tunnel: <你的TUNNEL_ID>
credentials-file: /root/.cloudflared/<TUNNEL_ID>.json

ingress:
  # 主服务路由
  - hostname: tv-tracker.yourdomain.com
    service: http://localhost:8318
  
  # 可选：API 健康检查（无需认证）
  - hostname: tv-tracker-api.yourdomain.com
    service: http://localhost:8318
    path: /api/health
  
  # 默认规则（必须放在最后）
  - service: http_status:404
```

**配置说明**：
- `tunnel`: 您的 Tunnel ID
- `credentials-file`: 凭证文件路径
- `hostname`: 您的子域名（需要先在 Cloudflare DNS 中添加 A 记录或 CNAME）
- `service`: 本地服务地址和端口

#### 5. 启动 Tunnel

**开发环境（前台运行）**:
```bash
cloudflared tunnel run tv-tracker
```

**生产环境（后台运行）**:

**方式 A：使用 systemd**
```bash
# 安装服务
cloudflared tunnel service install

# 启动服务
sudo systemctl start cloudflared-tunnel@tv-tracker

# 开机自启
sudo systemctl enable cloudflared-tunnel@tv-tracker

# 查看状态
sudo systemctl status cloudflared-tunnel@tv-tracker
```

**方式 B：使用 Docker**
```bash
docker run -d \
  --name cloudflared \
  --restart=unless-stopped \
  -v ~/.cloudflared:/home/cloudflared/.cloudflared \
  cloudflare/cloudflared:latest \
  tunnel --config /home/cloudflared/.cloudflared/config.yml run tv-tracker
```

#### 6. 配置 DNS

在 Cloudflare DNS 控制台添加记录：

**方式 A：CNAME 记录（推荐）**
```
类型: CNAME
名称: tv-tracker
目标: <TUNNEL_ID>.cfargotunnel.com
代理状态: 已代理（橙色云朵）
TTL: 自动
```

**方式 B：A 记录**
```
类型: A
名称: tv-tracker
IPv4 地址: 192.0.2.1（任意 IP，Tunnel 不使用）
代理状态: 已代理（橙色云朵）
TTL: 自动
```

---

### 方法二：使用 Docker Compose 集成

修改 `docker-compose.yml`，添加 cloudflared 服务：

```yaml
services:
  tv-tracker:
    build:
      context: .
      dockerfile: Dockerfile.api
    container_name: tv-tracker
    restart: unless-stopped
    ports:
      - "8318:18080"
    environment:
      - TMDB_API_KEY=${TMDB_API_KEY}
      - TELEGRAM_BOT_TOKEN=${TELEGRAM_BOT_TOKEN}
      - TELEGRAM_CHAT_ID=${TELEGRAM_CHAT_ID}
      - TELEGRAM_CHANNEL_ID=${TELEGRAM_CHANNEL_ID}
      - DB_PATH=/app/data/tv_tracker.db
      - BACKUP_DIR=/app/data/backups
      - REPORT_TIME=${REPORT_TIME:-09:00}
      - DISABLE_BOT=${DISABLE_BOT:-false}
      - WEB_ENABLED=true
      - WEB_LISTEN_ADDR=:18080
      - WEB_API_TOKEN=${WEB_API_TOKEN}
    volumes:
      - ./data:/app/data
    networks:
      - tv-tracker-net

  cloudflared:
    image: cloudflare/cloudflared:latest
    container_name: cloudflared-tunnel
    restart: unless-stopped
    command: tunnel run
    environment:
      - TUNNEL_TOKEN=<你的Tunnel Token>
    networks:
      - tv-tracker-net

networks:
  tv-tracker-net:
    driver: bridge
```

**获取 Tunnel Token**：
```bash
cloudflared tunnel token tv-tracker
```

---

## 🔧 高级配置

### 1. 多域名配置

如果您想使用多个域名：

```yaml
ingress:
  - hostname: tv-tracker.yourdomain.com
    service: http://localhost:8318
  
  - hostname: tracker.example.com
    service: http://localhost:8318
  
  - service: http_status:404
```

### 2. 路径规则配置

根据 URL 路径路由到不同服务（如果有多服务）：

```yaml
ingress:
  - hostname: tv-tracker.yourdomain.com
    service: http://localhost:8318
    # 所有请求都转发到主服务
  
  # 可选：为 API 单独配置域名
  - hostname: api.tv-tracker.yourdomain.com
    service: http://localhost:8318
    path: /api/.*
  
  - service: http_status:404
```

### 3. 访问控制

限制只有特定 IP 或国家可以访问：

```yaml
ingress:
  - hostname: tv-tracker.yourdomain.com
    service: http://localhost:8318
    # 只允许特定 IP
    originRequest:
      ipRules:
        - action: allow
          expression: "ip.src_addr in {1.2.3.4/32}"
  
  - service: http_status:403
```

### 4. 添加 Basic Auth

在 Tunnel 层面添加额外认证：

```yaml
ingress:
  - hostname: tv-tracker.yourdomain.com
    service: http://localhost:8318
    originRequest:
      noTLSVerify: true
      http2Origin: false
      # 注意：Basic Auth 需要在应用层配置
```

**更推荐的方式**：保持当前的 `WEB_API_TOKEN` 机制。

---

## 🔒 安全建议

### 1. Cloudflare Access（零信任网络）

如果您需要更强的安全控制，可以使用 Cloudflare Access：

```bash
# 安装 cloudflared
cloudflared tunnel login

# 创建 Access 策略
# 在 Cloudflare Dashboard 中配置：
# Zero Trust > Networks > Tunnels > 你的Tunnel > Configure
# Public Hostname > Add a public hostname
# > Access > Policy > 添加规则
```

**策略示例**：
- 允许特定 Email 域名
- 需要 OTP 验证
- 限制地理位置

### 2. 证书配置

虽然 Cloudflare Tunnel 自动处理 TLS，但您也可以：

```yaml
ingress:
  - hostname: tv-tracker.yourdomain.com
    service: https://localhost:8318
    originRequest:
      noTLSVerify: true  # 如果使用自签名证书
      caPool: /path/to/ca.pem
```

### 3. 速率限制

在 Cloudflare Dashboard 中配置：
```
Security > WAF > Custom rules > Create rule
```

**规则示例**：
```
If: (http.request.uri.path contains "/api/")
Then: Rate limit (100 requests per minute)
```

---

## 📊 监控与日志

### 查看 Tunnel 日志

```bash
# 实时日志
cloudflared tunnel info tv-tracker

# 详细日志
cloudflared --loglevel debug tunnel run tv-tracker
```

### Cloudflare Dashboard

访问 [Cloudflare Zero Trust Dashboard](https://dash.cloudflare.com/)：

- **Analytics**：流量统计
- **Logs**：请求日志
- **Health Checks**：服务健康状态

---

## 🧪 测试配置

### 1. 本地测试

```bash
# 确保服务正常运行
curl http://localhost:8318/api/health

# 应返回：{"status":"ok"}
```

### 2. Tunnel 连通性测试

```bash
# 通过 Tunnel 域名访问
curl https://tv-tracker.yourdomain.com/api/health

# 应返回：{"status":"ok"}
```

### 3. 完整功能测试

```bash
# 测试 API（需要 Token）
curl -H "Authorization: Bearer YOUR_TOKEN" \
  https://tv-tracker.yourdomain.com/api/library
```

---

## ❗ 常见问题

### 问题1：Tunnel 无法连接

**症状**：
```
Failed to fetch quick tunnel information
```

**解决方案**：
```bash
# 检查 cloudflared 版本
cloudflared --version

# 更新到最新版本
cloudflared update

# 检查网络连接
ping cloudflare.com
```

### 问题2：502 Bad Gateway

**原因**：本地服务未运行或端口错误

**解决方案**：
```bash
# 检查容器状态
docker ps | grep tv-tracker

# 检查端口映射
docker port tv-tracker

# 查看容器日志
docker logs tv-tracker
```

### 问题3：DNS 解析失败

**解决方案**：
1. 在 Cloudflare DNS 控制台确认记录已添加
2. 等待 DNS 传播（通常 1-5 分钟）
3. 使用 `dig` 验证：
```bash
dig tv-tracker.yourdomain.com
```

### 问题4：证书错误

**症状**：浏览器显示证书无效

**解决方案**：
- 确保在 Cloudflare DNS 中使用"已代理"状态（橙色云朵）
- Tunnel 会自动获取 Let's Encrypt 证书
- 清除浏览器缓存

---

## 📝 配置检查清单

部署前确认：

- [ ] Cloudflare 账户已创建并登录
- [ ] 域名已添加到 Cloudflare
- [ ] Tunnel 已创建并获取 Token
- [ ] `docker-compose.yml` 端口映射正确（`8318:18080`）
- [ ] 本地服务运行正常（`curl http://localhost:8318/api/health`）
- [ ] cloudflared 已安装并可访问
- [ ] DNS 记录已添加（CNAME 到 `*.cfargotunnel.com`）
- [ ] 配置文件路径正确
- [ ] 防火墙允许 cloudflared 出站连接

---

## 🚀 完整部署流程

### 步骤1：准备应用

```bash
# 1. 克隆仓库
git clone https://github.com/xc9973/tv-tracker.git
cd tv-tracker

# 2. 配置环境变量
cp .env.example .env
vim .env

# 3. 启动服务
mkdir -p data/backups
docker compose up -d

# 4. 验证服务
curl http://localhost:8318/api/health
```

### 步骤2：配置 Tunnel

```bash
# 1. 登录 Cloudflare
cloudflared tunnel login

# 2. 创建 Tunnel
cloudflared tunnel create tv-tracker

# 3. 获取 Token
cloudflared tunnel token tv-tracker

# 4. 更新 docker-compose.yml（添加 cloudflared 服务）
# 或创建独立的配置文件

# 5. 启动 Tunnel
docker compose up -d cloudflared

# 6. 验证 Tunnel
curl https://tv-tracker.yourdomain.com/api/health
```

### 步骤3：配置 DNS

在 Cloudflare Dashboard 中：

```
DNS > Add record
- Type: CNAME
- Name: tv-tracker
- Target: <TUNNEL_ID>.cfargotunnel.com
- Proxy status: Proxied (橙色云朵)
```

### 步骤4：测试访问

```bash
# 测试主页
curl https://tv-tracker.yourdomain.com/

# 测试 API
curl -H "Authorization: Bearer YOUR_TOKEN" \
  https://tv-tracker.yourdomain.com/api/dashboard
```

---

## 📚 参考资源

- [Cloudflare Tunnel 官方文档](https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/)
- [cloudflared GitHub](https://github.com/cloudflare/cloudflared)
- [Quick Tunnels 文档](https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/run-tunnel/trycloudflared/)

---

## 🎯 总结

**当前架构**：
- ✅ 单镜像部署，简单高效
- ✅ 端口 `8318` 对外暴露
- ✅ 内部端口 `18080` 提供服务

**Cloudflare Tunnel 配置**：
- ✅ 无需开放路由器端口
- ✅ 自动 HTTPS
- ✅ DDoS 防护
- ✅ 全球 CDN 加速

**推荐配置**：
```
本地服务: localhost:8318
    ↓
Cloudflare Tunnel
    ↓
公网域名: tv-tracker.yourdomain.com (HTTPS)
```

**访问地址**：
- 主页：`https://tv-tracker.yourdomain.com/`
- API：`https://tv-tracker.yourdomain.com/api/*`
- 健康检查：`https://tv-tracker.yourdomain.com/api/health`