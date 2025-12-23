# Implementation Plan: TV Tracker

## Overview

基于 Go + Gin + SQLite 实现影视剧订阅追踪系统，前端使用 React + TypeScript。采用增量开发方式，每个任务构建在前一个任务之上。

## Tasks

- [x] 1. 项目初始化和数据模型
  - [x] 1.1 创建 Go 项目结构和依赖配置
    - 创建 go.mod，添加 gin、go-sqlite3、gopter、testify 依赖
    - 创建目录结构：cmd/server、internal/models、internal/repository、internal/service、internal/handler、internal/tmdb
    - _Requirements: 8.1_

  - [x] 1.2 实现数据模型 (`internal/models/models.go`)
    - 定义 TaskType 常量 (UPDATE, ORGANIZE)
    - 定义 TVShow 结构体（包含 origin_country, resource_time 字段）
    - 定义 Task 结构体（包含 resource_time 字段）
    - 定义 Episode 结构体
    - _Requirements: 8.1, 8.2, 10.5_

  - [x] 1.3 编写属性测试：TVShow 持久化往返
    - **Property 16: TVShow Persistence Round-Trip**
    - **Validates: Requirements 8.1, 8.4**

- [x] 2. 数据库层实现
  - [x] 2.1 实现 SQLite 数据库初始化 (`internal/repository/sqlite.go`)
    - 创建数据库连接
    - 实现表创建 (tv_shows, tasks, episodes)
    - 创建索引
    - _Requirements: 8.1, 8.2_

  - [x] 2.2 实现 TVShowRepository
    - Create, GetByTMDBID, GetAllActive, GetAll, Update, Archive 方法
    - _Requirements: 2.2, 8.1, 8.4_

  - [x] 2.3 实现 EpisodeRepository
    - Upsert, GetByTMDBID, GetByAirDate, DeleteByTMDBID 方法
    - _Requirements: 8.1, 9.1_

  - [x] 2.4 实现 TaskRepository
    - Create, GetPendingByType, GetByShowAndEpisode, ExistsOrganizeTask, Complete, GetAllPending, GetByID 方法
    - _Requirements: 8.2_

  - [x] 2.5 编写属性测试：Task 外键完整性
    - **Property 17: Task Foreign Key Integrity**
    - **Validates: Requirements 8.2**

- [x] 3. Checkpoint - 数据层验证
  - 确保所有测试通过，如有问题请询问用户

- [x] 4. TMDB 客户端实现
  - [x] 4.1 实现 TMDB Client (`internal/tmdb/client.go`)
    - NewClient 构造函数
    - SearchTV 方法：调用 /search/tv API
    - GetTVDetails 方法：调用 /tv/{id} API
    - GetSeasonEpisodes 方法：调用 /tv/{id}/season/{season} API
    - 定义 SearchResult、TVDetails、EpisodeInfo、SeasonDetail 结构体
    - _Requirements: 1.1, 2.1, 3.2_

  - [x] 4.2 编写属性测试：API 错误处理
    - **Property 2: API Error Handling**
    - **Validates: Requirements 1.3**

  - [x] 4.3 编写属性测试：搜索结果结构验证
    - **Property 1: TMDB Search Returns Valid Results**
    - **Validates: Requirements 1.1, 1.2**

- [x] 5. 订阅管理服务实现
  - [x] 5.1 实现 SubscriptionManager (`internal/service/subscription.go`)
    - Subscribe 方法：获取 TMDB 详情，存储剧集，同步最新一季剧集到 episodes 表
    - IsSubscribed 方法：检查是否已订阅
    - GetAllSubscriptions 方法：获取所有订阅
    - Unsubscribe 方法：取消订阅
    - InferResourceTime 方法：根据国家推断资源时间
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 10.1, 10.2, 10.3, 10.4_

  - [x] 5.2 编写属性测试：订阅数据往返
    - **Property 3: Subscription Data Round-Trip**
    - **Validates: Requirements 2.2**

  - [x] 5.3 编写属性测试：订阅幂等性
    - **Property 4: Subscription Idempotence**
    - **Validates: Requirements 2.3**

  - [x] 5.4 编写属性测试：资源时间推断
    - **Property 18: Resource Time Inference**
    - **Validates: Requirements 10.1, 10.2, 10.3, 10.4**

- [x] 6. 任务生成服务实现
  - [x] 6.1 实现 TaskGenerator (`internal/service/task_generator.go`)
    - SyncAll 方法：遍历未归档订阅，同步最新季剧集，生成任务
    - checkEpisodeUpdate 方法：检查是否需要生成 UPDATE_Task
    - checkShowEnded 方法：检查是否需要生成 ORGANIZE_Task
    - FormatEpisodeID 函数：格式化为 SxxExx
    - _Requirements: 3.1, 3.3, 4.1, 4.2, 4.4, 5.1_

  - [x] 6.2 编写属性测试：剧集 ID 格式
    - **Property 9: Episode ID Format**
    - **Validates: Requirements 4.2**

  - [x] 6.3 编写属性测试：同步只处理活跃剧集
    - **Property 5: Sync Processes Only Active Shows**
    - **Validates: Requirements 3.1, 5.4, 6.3**

  - [x] 6.4 编写属性测试：UPDATE_Task 幂等性
    - **Property 10: UPDATE_Task Idempotence**
    - **Validates: Requirements 4.3**

  - [x] 6.5 编写属性测试：ORGANIZE_Task 幂等性
    - **Property 12: ORGANIZE_Task Idempotence**
    - **Validates: Requirements 5.3**

- [x] 7. Checkpoint - 核心服务验证
  - 确保所有测试通过，如有问题请询问用户

- [x] 8. 任务看板服务实现
  - [x] 8.1 实现 TaskBoardService (`internal/service/task_board.go`)
    - GetDashboardData 方法：获取待处理任务
    - CompleteTask 方法：完成任务，ORGANIZE 任务同时归档剧集
    - _Requirements: 6.1, 6.2, 6.4, 7.1, 7.2_

  - [x] 8.2 编写属性测试：UPDATE_Task 完成
    - **Property 13: UPDATE_Task Completion**
    - **Validates: Requirements 6.1**

  - [x] 8.3 编写属性测试：ORGANIZE_Task 完成级联归档
    - **Property 14: ORGANIZE_Task Completion Cascades to Archive**
    - **Validates: Requirements 6.2**

- [x] 9. HTTP Handler 实现
  - [x] 9.1 实现 Handler (`internal/handler/handler.go`)
    - GET /api/dashboard - 获取看板数据
    - GET /api/search - 搜索 TMDB API
    - POST /api/subscribe - 订阅剧集
    - POST /api/sync - 手动同步
    - POST /api/tasks/:id/complete - 完成任务
    - GET /api/library - 获取片库数据
    - POST /api/report - 发送 Telegram 日报
    - _Requirements: 1.1, 1.4, 2.3, 7.1, 7.2, 7.3_

  - [x] 9.2 编写属性测试：任务渲染完整性
    - **Property 15: Task Rendering Completeness**
    - **Validates: Requirements 7.3**

- [x] 10. Telegram 通知服务实现
  - [x] 10.1 实现 TelegramNotifier (`internal/notify/telegram.go`)
    - NewTelegramNotifier 构造函数
    - SendMessage 方法：发送消息到 Telegram
    - SendDailyReport 方法：生成并发送每日更新日报
    - _Requirements: 9.1, 9.2, 9.3, 9.4, 9.5_

  - [x] 10.2 编写属性测试：日报包含所有今日剧集
    - **Property 19: Daily Report Contains All Today's Episodes**
    - **Validates: Requirements 9.1, 9.2**

- [x] 11. 前端 React 应用实现
  - [x] 11.1 初始化 React 项目
    - 使用 Vite 创建 React + TypeScript 项目
    - 配置代理到 Go 后端
    - 安装依赖：axios、react-router-dom
    - _Requirements: 7.1, 7.2_

  - [x] 11.2 创建基础组件和布局
    - App.tsx - 路由配置
    - Layout.tsx - 基础布局组件
    - 全局样式
    - _Requirements: 7.1, 7.4_

  - [x] 11.3 实现任务看板页面 (Dashboard.tsx)
    - 展示今日更新任务列表（按资源时间排序）
    - 展示待整理任务列表
    - 同步按钮
    - 发送日报按钮
    - 任务完成按钮
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 10.5_

  - [x] 11.4 实现搜索页面 (Search.tsx)
    - 搜索输入框
    - 搜索结果展示（剧名、海报、首播日期）
    - 订阅按钮
    - _Requirements: 1.2, 1.4_

  - [x] 11.5 实现片库页面 (Library.tsx)
    - 展示所有订阅剧集
    - 显示剧集状态
    - _Requirements: 2.4_

- [x] 12. 应用入口和配置
  - [x] 12.1 实现主程序入口 (`cmd/server/main.go`)
    - 加载配置 (TMDB API Key, Telegram Bot Token, Chat ID)
    - 初始化数据库
    - 初始化服务
    - 启动 Gin 服务器
    - 支持 CLI 模式：`--report` 参数直接发送日报（用于 cron 定时任务）
    - _Requirements: 8.3, 9.4_

- [x] 13. Final Checkpoint - 完整功能验证
  - 确保所有测试通过
  - 验证完整工作流：搜索 → 订阅 → 同步 → 任务完成 → 发送日报
  - 如有问题请询问用户

## Notes

- 每个任务都引用了具体的需求条款以便追溯
- Checkpoint 任务用于阶段性验证
- 属性测试验证核心正确性属性
- 单元测试验证具体示例和边界情况
