# TV Tracker 架构文档

## 项目概述

TV Tracker 是一个电视剧追踪管理系统,帮助用户订阅、追踪和管理美剧更新。系统由 Go 后端、React 前端、SQLite 数据库和 Telegram Bot 组成。

## 技术栈

### 后端
- **语言**: Go 1.23
- **Web 框架**: Gin
- **数据库**: SQLite 3
- **外部 API**: TMDB (The Movie Database)
- **通知**: Telegram Bot API

### 前端
- **框架**: React 18 + TypeScript
- **构建工具**: Vite
- **路由**: React Router
- **HTTP 客户端**: Axios
- **样式**: CSS Modules

### 部署
- **容器化**: Docker + Docker Compose
- **反向代理**: Nginx
- **数据持久化**: Docker Volume

## 系统架构

```
┌─────────────────────────────────────────────────────────────┐
│                        用户交互层                              │
├─────────────────────────────────────────────────────────────┤
│  Web 前端 (React)          │    Telegram Bot                 │
│  - 订阅管理                 │    - 命令交互                    │
│  - 任务看板                 │    - 每日推送                    │
│  - 今日更新                 │    - 内联按钮                    │
└──────────┬──────────────────┴─────────────┬─────────────────┘
           │                                 │
           │ HTTP API                        │ Bot API
           │ (Bearer Token Auth)             │
           ▼                                 ▼
┌─────────────────────────────────────────────────────────────┐
│                      HTTP Handler 层                          │
├─────────────────────────────────────────────────────────────┤
│  handler/http.go                                             │
│  - API 路由注册                                               │
│  - 请求鉴权 (authMiddleware)                                  │
│  - 请求处理与响应                                              │
└──────────┬──────────────────────────────────────────────────┘
           │
           │ 调用服务层
           ▼
┌─────────────────────────────────────────────────────────────┐
│                       Service 层                              │
├─────────────────────────────────────────────────────────────┤
│  SubscriptionManager   │  TaskBoardService  │  TaskGenerator │
│  - 订阅管理             │  - 任务面板         │  - 任务生成    │
│  - TMDB 数据同步        │  - 任务完成/延期    │  - 集数同步    │
│                        │                    │                │
│  Scheduler            │  BackupService      │  TimeService   │
│  - 定时任务            │  - 数据库备份        │  - 时间工具    │
│  - 每日报告            │  - 备份清理          │                │
└──────────┬────────────────────────────────────┬─────────────┘
           │                                     │
           │ 调用仓储层                          │ 外部 API
           ▼                                     ▼
┌──────────────────────────────┐    ┌──────────────────────┐
│      Repository 层            │    │    External API      │
├──────────────────────────────┤    ├──────────────────────┤
│  TVShowRepository            │    │  TMDB API Client     │
│  - 剧集 CRUD                  │    │  - 搜索电视剧         │
│                              │    │  - 获取详情           │
│  EpisodeRepository           │    │  - 获取季/集信息      │
│  - 集数 CRUD                  │    │                      │
│  - 今日更新查询                │    │  Telegram API        │
│                              │    │  - 发送消息           │
│  TaskRepository              │    │  - 处理回调           │
│  - 任务 CRUD                  │    │  - 内联键盘           │
│  - 任务状态管理                │    └──────────────────────┘
└──────────┬───────────────────┘
           │
           │ 数据持久化
           ▼
┌─────────────────────────────────────────────────────────────┐
│                       数据库层                                 │
├─────────────────────────────────────────────────────────────┤
│  SQLite Database (tv_tracker.db)                             │
│  - tv_shows: 电视剧信息                                        │
│  - episodes: 集数信息                                          │
│  - tasks: 观看任务                                             │
└─────────────────────────────────────────────────────────────┘
```

## 核心模块说明

### 1. 入口层 (cmd/server)

**main.go**
- 程序入口,负责初始化所有组件
- 解析命令行参数与环境变量
- 创建数据库连接
- 初始化各层服务
- 启动 Web 服务器 / Telegram Bot / 定时调度器

### 2. 数据模型层 (internal/models)

**models.go**
```go
- TVShow: 电视剧信息 (TMDB ID, 名称, 季数, 状态, 资源时间等)
- Episode: 集数信息 (所属剧集, 季/集号, 标题, 播出日期等)
- Task: 观看任务 (关联集数, 状态, 计划日期等)
```

### 3. 仓储层 (internal/repository)

**sqlite.go**
- 数据库连接管理
- 表结构初始化
- 事务支持

**tvshow_repository.go**
- Create: 创建新剧集订阅
- GetByTMDBID: 根据 TMDB ID 查询
- GetByID: 根据主键查询
- GetAllActive: 获取所有活跃订阅
- Update: 更新剧集信息
- Archive: 归档剧集

**episode_repository.go**
- Create/BatchCreate: 创建集数记录
- GetByShowAndSeason: 查询某季所有集数
- GetTodayEpisodesWithShowInfo: 查询今日更新(含剧集名)
- Update: 更新集数信息

**task_repository.go**
- Create: 创建观看任务
- GetPendingTasks: 获取待完成任务
- GetTasksByDate: 查询指定日期任务
- CompleteTask: 标记任务完成
- PostponeTask: 延期任务到明天
- UpdateStatus: 更新任务状态

### 4. 服务层 (internal/service)

**subscription.go - 订阅管理**
- Subscribe(tmdbID): 订阅新剧集
  1. 调用 TMDB API 获取剧集详情
  2. 保存剧集信息到数据库
  3. 同步最新季集数信息
  4. 创建观看任务
- Unsubscribe(showID): 取消订阅(归档)
- GetAllSubscriptions(): 获取所有订阅

**task_generator.go - 任务生成器**
- SyncShow(showID): 同步单个剧集
  - 获取 TMDB 最新季信息
  - 对比本地数据库
  - 新增缺失的集数和任务
- SyncAll(): 同步所有订阅剧集
- GenerateTasksForEpisode(): 为集数生成任务

**task_board.go - 任务看板**
- GetDashboardData(): 获取面板数据
  - 今日任务
  - 延期任务
  - 未来任务
- CompleteTask(taskID): 完成任务
- PostponeTask(taskID): 延期任务

**scheduler.go - 定时调度**
- 每日定时发送报告 (默认 08:00)
- 每周日凌晨 3:00 备份数据库
- calculateNextReportTime(): 计算下次报告时间
- calculateNextBackupTime(): 计算下次备份时间

**backup.go - 备份服务**
- Backup(): 创建数据库备份
  - 文件名格式: `tv_tracker_YYYYMMDD_HHMMSS.db`
  - 保存到 `data/backups/` 目录
- CleanOldBackups(keepDays): 清理旧备份
  - 默认保留最近 7 天

### 5. HTTP 处理层 (internal/handler)

**http.go**

核心结构:
```go
type HTTPHandler struct {
    tmdbClient  *tmdb.Client
    subMgr      *service.SubscriptionManager
    taskBoard   *service.TaskBoardService
    episodeRepo *repository.EpisodeRepository
    showRepo    *repository.TVShowRepository
    backupSvc   *service.BackupService
    apiToken    string
}
```

API 路由:
```
GET  /api/health                  - 健康检查 (无需鉴权)
GET  /api/dashboard               - 获取任务面板
GET  /api/today                   - 获取今日更新
GET  /api/search?q=<query>        - 搜索电视剧
POST /api/subscribe               - 订阅剧集
DELETE /api/subscribe/:id         - 取消订阅
GET  /api/library                 - 获取片库列表
POST /api/tasks/:id/complete      - 完成任务
POST /api/tasks/:id/postpone      - 延期任务
PUT  /api/shows/:id/resource-time - 更新资源时间
POST /api/backup                  - 手动备份
```

鉴权机制:
- 使用 Bearer Token 认证
- Token 通过环境变量 `WEB_API_TOKEN` 配置
- authMiddleware 拦截所有需要鉴权的请求

### 6. Telegram Bot (internal/notify)

**telegram.go**

命令:
- `/start` - 欢迎消息与主菜单
- `/today` - 查看今日更新
- `/tasks` - 查看待办任务
- `/sync` - 手动同步更新
- `/backup` - 手动备份数据库

回调处理:
- `sync` - 同步所有剧集
- `organize` - 整理任务(清理已播完)
- `complete_<taskID>` - 完成任务
- `postpone_<taskID>` - 延期任务

每日推送:
- 定时发送今日更新列表
- 支持发送到频道或私聊
- 格式化消息包含剧集名、集数、标题

### 7. 外部 API 客户端 (internal/tmdb)

**client.go**

功能:
- SearchTV(query): 搜索电视剧
- GetTVDetails(tmdbID): 获取剧集详情
- GetSeasonDetails(tmdbID, seasonNum): 获取季详情
- 自动处理 API Key 和请求限流

### 8. 前端架构 (web/src)

**目录结构**:
```
web/src/
├── main.tsx          - 应用入口
├── App.tsx           - 路由配置
├── components/       - 通用组件
│   └── Layout.tsx    - 页面布局(导航栏)
├── pages/            - 页面组件
│   ├── Today.tsx     - 今日更新
│   ├── Dashboard.tsx - 任务看板
│   ├── Library.tsx   - 片库管理
│   └── Search.tsx    - 搜索订阅
└── services/
    └── api.ts        - API 客户端封装
```

**api.ts 核心功能**:
- Axios 实例配置
- 自动添加 Bearer Token
- 统一错误处理
- 类型安全的 API 调用

**页面功能**:
1. **Today** - 展示当天更新的集数
2. **Dashboard** - 分类显示任务(今日/延期/未来)
3. **Library** - 管理订阅剧集,设置资源时间
4. **Search** - 搜索并订阅新剧集

## 数据流示例

### 订阅新剧集流程

```
1. 用户在前端搜索剧集
   └─> GET /api/search?q=<query>
       └─> tmdbClient.SearchTV()
           └─> TMDB API 返回搜索结果

2. 用户点击订阅
   └─> POST /api/subscribe {tmdb_id: 12345}
       └─> subMgr.Subscribe(12345)
           ├─> tmdbClient.GetTVDetails(12345)
           ├─> showRepo.Create(show)
           ├─> tmdbClient.GetSeasonDetails() for 最新季
           ├─> episodeRepo.BatchCreate(episodes)
           └─> taskGen.GenerateTasksForEpisode() 创建任务

3. 返回订阅成功
   └─> 前端刷新片库列表
```

### 每日同步与推送流程

```
1. Scheduler 定时触发 (每日 08:00)
   └─> taskGen.SyncAll()
       └─> 遍历所有订阅剧集
           ├─> tmdbClient.GetSeasonDetails() 获取最新信息
           ├─> 对比本地数据库
           ├─> episodeRepo.BatchCreate() 新增集数
           └─> taskRepo.Create() 创建新任务

2. 发送每日报告
   └─> telegramBot.SendDailyReport()
       ├─> episodeRepo.GetTodayEpisodesWithShowInfo()
       ├─> FormatDailyReport() 格式化消息
       └─> 发送到 Telegram 频道/私聊
```

## 配置说明

### 环境变量

| 变量名 | 必填 | 默认值 | 说明 |
|-------|------|--------|------|
| `TMDB_API_KEY` | 是 | - | TMDB API 密钥 |
| `TELEGRAM_BOT_TOKEN` | Bot 模式必填 | - | Telegram Bot Token |
| `TELEGRAM_CHAT_ID` | Bot 模式必填 | - | 管理员 Chat ID |
| `TELEGRAM_CHANNEL_ID` | 否 | - | 推送频道 ID |
| `REPORT_TIME` | 否 | `09:00` | 每日报告时间 |
| `WEB_ENABLED` | 否 | `false` | 启用 Web 服务 |
| `WEB_LISTEN_ADDR` | 否 | `:18080` | Web 监听地址 |
| `WEB_API_TOKEN` | Web 模式必填 | - | API 鉴权 Token |
| `DB_PATH` | 否 | `./data/tv_tracker.db` | 数据库路径 |
| `BACKUP_DIR` | 否 | `./data/backups` | 备份目录 |
| `DISABLE_BOT` | 否 | `false` | 禁用 Telegram Bot |

### Docker 部署架构

```
┌─────────────────────────────────────────────┐
│         Docker Compose 服务拓扑              │
├─────────────────────────────────────────────┤
│                                             │
│  tv-tracker-web (Nginx)                     │
│  ├─ 端口: 80:80 (对外暴露)                   │
│  ├─ 功能: 提供 React 静态资源                │
│  └─ 代理: /api/* → tv-tracker-api:18080     │
│       │                                     │
│       │ HTTP Proxy                          │
│       ▼                                     │
│  tv-tracker-api (Go Backend)                │
│  ├─ 端口: 18080 (仅容器内部)                 │
│  ├─ 功能: HTTP API + Telegram Bot           │
│  └─ 卷: ./data → /app/data                  │
│                                             │
└─────────────────────────────────────────────┘
```

## 数据库设计

### tv_shows 表
```sql
CREATE TABLE tv_shows (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tmdb_id INTEGER UNIQUE NOT NULL,
    name TEXT NOT NULL,
    total_seasons INTEGER,
    status TEXT,
    origin_country TEXT,
    resource_time TEXT,        -- 资源更新时间 (如 "周四凌晨3点")
    is_archived BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);
```

### episodes 表
```sql
CREATE TABLE episodes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    show_id INTEGER NOT NULL,
    season INTEGER NOT NULL,
    episode INTEGER NOT NULL,
    title TEXT,
    air_date DATE,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    FOREIGN KEY (show_id) REFERENCES tv_shows(id),
    UNIQUE(show_id, season, episode)
);
```

### tasks 表
```sql
CREATE TABLE tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    episode_id INTEGER NOT NULL,
    status TEXT DEFAULT 'pending',  -- pending, completed, postponed
    planned_date DATE,
    completed_at TIMESTAMP,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    FOREIGN KEY (episode_id) REFERENCES episodes(id)
);
```

## 测试架构

### 属性测试 (tests/property)

使用 `gopter` 进行基于属性的测试:

- **handler_test.go**: HTTP 处理器测试
- **persistence_test.go**: 数据持久化测试
- **subscription_test.go**: 订阅管理测试
- **task_board_test.go**: 任务面板测试
- **task_generator_test.go**: 任务生成器测试
- **telegram_test.go**: Telegram Bot 测试
- **tmdb_test.go**: TMDB 客户端测试

测试覆盖:
- 并发安全性
- 数据一致性
- 边界条件
- 异常处理

## 部署模式

### 1. Docker Compose (推荐)
- 前后端分离
- Nginx 反向代理
- 数据持久化

### 2. 单体二进制
- 编译为单个可执行文件
- 适合开发环境
- 支持 CLI 模式

### 3. 仅 Bot 模式
- 不启用 Web 界面
- 纯 Telegram 交互
- 轻量级部署

## 安全机制

1. **API 鉴权**: Bearer Token 认证
2. **常量时间比较**: 防止 Timing 攻击
3. **输入验证**: 严格的参数校验
4. **SQL 注入防护**: 使用参数化查询
5. **CORS 配置**: Nginx 层面控制

## 性能优化

1. **数据库索引**: TMDB ID, 播出日期, 任务状态
2. **批量操作**: BatchCreate 减少数据库往返
3. **定时任务**: 避免实时同步阻塞用户请求
4. **Nginx 缓存**: 静态资源 7 天缓存
5. **连接复用**: HTTP Keep-Alive

## 未来扩展方向

1. **多用户支持**: 用户系统与权限管理
2. **通知渠道扩展**: 企业微信、钉钉、邮件
3. **推荐系统**: 基于观看历史推荐相似剧集
4. **统计分析**: 观看时长、偏好分析
5. **移动端应用**: React Native / Flutter
6. **搜索优化**: 全文搜索、模糊匹配
7. **字幕集成**: 字幕下载与管理
8. **观看进度**: 集内进度跟踪

## 常见问题排查

### 1. 前端 401 错误
- 检查 `.env` 中 `WEB_API_TOKEN` 与 `VITE_API_TOKEN` 是否一致
- 清除浏览器缓存重新构建前端

### 2. API 端口冲突
- 修改 `WEB_LISTEN_ADDR` 或 `docker-compose.yml` 端口映射
- 使用 `lsof -i :<port>` 查找占用端口的进程

### 3. 数据库锁定
- SQLite 不支持高并发写入
- 检查是否有长时间未提交的事务
- 考虑迁移到 PostgreSQL (需修改 Repository 层)

### 4. TMDB API 限流
- 默认每秒 40 次请求
- 添加重试机制与指数退避
- 缓存常用数据减少 API 调用

### 5. Telegram 推送失败
- 验证 Bot Token 与 Chat ID 正确性
- 检查网络是否能访问 Telegram API
- 查看日志确认错误详情

## 开发指南

### 添加新的 API 端点

1. 在 `handler/http.go` 的 `RegisterRoutes` 中注册路由
2. 实现处理函数 (遵循 Gin Handler 签名)
3. 调用相应的 Service 层方法
4. 返回统一的 JSON 响应格式

### 添加新的服务

1. 在 `service/` 目录创建新文件
2. 定义服务结构体和方法
3. 在 `cmd/server/main.go` 中初始化
4. 注入到 Handler 或其他服务

### 数据库迁移

当前使用简单的 SQL 初始化,未来可考虑:
- [golang-migrate](https://github.com/golang-migrate/migrate)
- [goose](https://github.com/pressly/goose)

## 贡献指南

1. Fork 项目并创建特性分支
2. 编写测试用例
3. 确保所有测试通过: `go test ./...`
4. 提交 Pull Request

## 许可证

本项目采用 MIT 许可证。
