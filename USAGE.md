# 使用文档

本项目是一个 TV Tracker，包含后端（Go + SQLite + TMDB API + Telegram Bot）与可选的前端 Web 控制台（Vite + React）。

## 功能概览

- 订阅剧集（基于 TMDB ID）
- 同步剧集更新并生成任务
- 今日更新与待整理任务展示
- Telegram Bot 交互（订阅、同步、备份、日报）
- 可选 Web API + 前端页面

## 快速开始

### 1. 准备环境

- Go 1.23+
- Node.js 18+（若使用 Web 前端）
- SQLite（内置驱动，无需单独安装）

### 2. 配置环境变量

后端使用环境变量配置：

- `TMDB_API_KEY`：TMDB API Key（必填，否则 TMDB 请求失败）
- `TELEGRAM_BOT_TOKEN`：Telegram Bot Token（必填）
- `TELEGRAM_CHAT_ID`：管理员 Chat ID（必填）
- `TELEGRAM_CHANNEL_ID`：日报发送频道 ID（可选，未配置则发给管理员）
- `DB_PATH`：SQLite 数据库路径（默认 `tv_tracker.db`）
- `BACKUP_DIR`：备份目录（默认 `backups`）
- `REPORT_TIME`：日报时间，格式 `HH:MM`（默认 `08:00`）
- `WEB_ENABLED`：是否启用 HTTP API（`true/false`，默认 `false`）
- `WEB_LISTEN_ADDR`：HTTP 监听地址（默认 `:18080`）
- `WEB_API_TOKEN`：HTTP API Bearer Token（启用 Web 时必填）

建议在本地创建 `.env`（自行加载）或在运行命令前导出环境变量。

### 3. 启动后端服务

```bash
go run ./cmd/server
```

#### 发送一次日报并退出

```bash
go run ./cmd/server -report
```

### 4. （可选）启动前端 Web

前端需要配置环境变量：

- `VITE_API_BASE`：API 基础路径（默认 `/api`）
- `VITE_API_TOKEN`：与 `WEB_API_TOKEN` 相同的 Bearer Token

```bash
cd web
npm install
npm run dev
```

浏览器访问 Vite 提示的地址即可。

## 运行模式说明

### Telegram Bot

- 默认启动 Telegram Bot
- 通过 `/start` 进入主菜单
- 功能包括：今日更新、订阅剧集、待整理、同步更新、管理与备份

### HTTP API

- 仅当 `WEB_ENABLED=true` 且 `WEB_API_TOKEN` 已设置时启用
- 所有 `/api/*` 请求需要 `Authorization: Bearer <token>`
- `/api/health` 支持未认证健康检查

## HTTP API 一览

> 以下接口均为 `GET/POST/PUT/DELETE`，默认前缀 `/api`

- `GET /health`：健康检查（无需认证）
- `GET /dashboard`：获取任务看板数据
- `GET /today`：获取今日播出剧集
- `GET /search?q=xxx`：搜索剧集
- `POST /subscribe`：订阅剧集（body: `{"tmdb_id": 123}`）
- `DELETE /subscribe/:id`：取消订阅（软删除）
- `GET /library`：获取订阅列表
- `POST /tasks/:id/complete`：完成任务（ORGANIZE 会归档剧集）
- `POST /tasks/:id/postpone`：任务延期到明天
- `PUT /shows/:id/resource-time`：更新资源时间（body: `{"resource_time":"20:00"}`）
- `POST /backup`：触发一次备份

### 示例：调用 API

```bash
curl -H "Authorization: Bearer $WEB_API_TOKEN" http://localhost:18080/api/dashboard
```

## 数据与备份

- SQLite 数据库由程序自动建表
- 备份默认存放在 `BACKUP_DIR` 指定目录
- Scheduler 每周日 03:00 自动备份

## 常见问题

- TMDB 请求失败：确认 `TMDB_API_KEY` 已设置
- Web API 401：确认 `WEB_API_TOKEN` 与请求头一致
- Bot 无响应：确认 `TELEGRAM_BOT_TOKEN` 与 `TELEGRAM_CHAT_ID` 正确
