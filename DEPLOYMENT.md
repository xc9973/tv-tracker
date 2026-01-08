# Docker 部署指南

## 1. 前置条件
- 已安装 Docker 与 Docker Compose
- 拥有 TMDB API Key、Telegram Bot Token 与 Chat ID
- 机器具备访问 TMDB、Telegram 的网络环境

## 2. 初始化配置
1. 复制环境变量示例：
   ```bash
   cp .env.example .env
   ```
2. 编辑 `.env`，填入 `TMDB_API_KEY`、`TELEGRAM_*`、`WEB_API_TOKEN` 等参数。前端 `VITE_API_TOKEN` 需与 `WEB_API_TOKEN` 保持一致。
3. 创建数据目录（持久化 SQLite 与备份）：
   ```bash
   mkdir -p data/backups
   ```

## 3. 构建与启动
```bash
docker compose build
docker compose up -d
```
- `tv-tracker-api`：Go 后端 + Telegram Bot + HTTP API（内部监听 18080）
- `tv-tracker-web`：Nginx 提供 React 静态资源，并代理 `/api` 到后端

访问入口：`http://<NAS_IP>/`

## 4. 目录与卷说明
```
./data/
├── tv_tracker.db      # SQLite 数据库
└── backups/           # 定期备份文件
```
Compose 中将 `./data` 映射到容器 `/app/data`，请在 NAS 上做好备份。

## 5. NAS 使用提示
- **端口冲突**：若 NAS 已占用 80 端口，可修改 `docker-compose.yml` 的 `ports` 映射，例如 `18080:80`。
- **Synology/QNAP**：在 File Station/共享文件夹中放置项目目录，再通过 Docker GUI/SSH 执行 Compose。
- **日志查看**：
  ```bash
  docker compose logs -f
  docker compose logs -f tv-tracker-api
  ```
- **升级流程**：
  ```bash
  git pull
  docker compose build
  docker compose up -d
  ```

## 6. 常用排查
- **前端 401**：确认 `.env` 中 `WEB_API_TOKEN` 与 `VITE_API_TOKEN` 一致，并刷新浏览器缓存。
- **API 不可达**：`docker compose ps` 检查 `tv-tracker-api` 是否 Healthy；使用 `docker compose logs tv-tracker-api` 查看报错。
- **数据库恢复**：从 `data/backups/` 复制文件覆盖 `tv_tracker.db`，再重启 API 容器。

## 7. 验证步骤
1. `docker compose ps`：确认两个服务均为 `Up` 状态。
2. `curl http://localhost/api/health`：无需鉴权应返回 `{ "status": "ok" }`。
3. `curl -H "Authorization: Bearer $WEB_API_TOKEN" http://localhost/api/dashboard`：返回任务 JSON。
4. 浏览器访问主页，验证 Dashboard/Library 正常加载。
5. `curl http://localhost/library`：应返回 `index.html`（SPA fallback）。
6. `docker compose restart tv-tracker-api` 后刷新页面，验证数据仍然存在（持久化生效）。
7. `curl http://localhost:18080/api/health` 应失败（API 端口未对外暴露）。
