# Implementation Plan: TV Tracker

## Overview

基于 Go + telebot + SQLite 实现影视剧订阅追踪系统，通过 Telegram Bot 交互。采用增量开发方式，每个任务构建在前一个任务之上。

## Tasks

- [x] 1. 项目初始化和数据模型
  - [x] 1.1 创建 Go 项目结构和依赖配置
    - 创建 go.mod，添加 telebot、go-sqlite3、gopter、testify 依赖
    - 创建目录结构：cmd/server、internal/models、internal/repository、internal/service、internal/notify、internal/tmdb
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

- [x] 9. Telegram Bot 实现
  - [x] 9.1 实现 TelegramBot (`internal/notify/telegram.go`)
    - NewTelegramBot 构造函数
    - HandleStart: /start 命令 - 显示主菜单按钮
    - HandleHelp: /help 命令 - 帮助信息
    - HandleText: 处理文本输入（根据当前状态）
    - 按钮回调：HandleTasksCallback, HandleSubscribeCallback, HandleOrganizeCallback, HandleSyncCallback, HandleAdminCallback, HandleAPIKeyCallback, HandleBackupCallback, HandleBackCallback
    - 任务完成回调：HandleCompleteTaskCallback (✅ 已完成), HandleArchiveCallback (✅ 已归档)
    - 状态管理：StateIdle, StateWaitingTMDBID, StateWaitingAPIKey
    - 键盘生成：MainMenuKeyboard, AdminMenuKeyboard, BackButtonKeyboard, TaskListKeyboard
    - IsOwner: 权限检查（只响应配置的 Chat ID）
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5, 7.6, 7.7, 7.8, 7.9, 7.10_

  - [x] 9.2 编写属性测试：任务渲染完整性
    - **Property 15: Task Rendering Completeness**
    - **Validates: Requirements 7.9**

- [x] 10. 日报功能实现
  - [x] 10.1 实现日报格式化
    - FormatDailyReport 方法：生成每日更新日报消息
    - 包含剧名、集数信息 (SxxExx)、资源时间
    - _Requirements: 9.1, 9.2, 9.3_

  - [x] 10.2 编写属性测试：日报包含所有今日剧集
    - **Property 19: Daily Report Contains All Today's Episodes**
    - **Validates: Requirements 9.1, 9.2**

- [x] 11. 数据库备份服务实现
  - [x] 11.1 实现 BackupService (`internal/service/backup.go`)
    - Backup 方法：执行数据库备份
    - GetLastBackupTime 方法：获取上次备份时间
    - CleanOldBackups 方法：清理旧备份（保留最近 4 个）
    - _Requirements: 11.1, 11.2, 11.3_

  - [x] 11.2 实现 Scheduler (`internal/service/scheduler.go`)
    - ScheduleDailyReport 方法：每天早上定时发送日报
    - ScheduleWeeklyBackup 方法：每周自动备份
    - Start 方法：启动所有定时任务
    - _Requirements: 9.1, 11.1_

  - [x] 11.3 集成备份到 Admin 命令
    - 在 /admin 中显示上次备份时间
    - 添加手动备份按钮回调
    - _Requirements: 11.4_

- [x] 12. 应用入口和配置
  - [x] 12.1 实现主程序入口 (`cmd/server/main.go`)
    - 加载配置 (TMDB API Key, Telegram Bot Token, Chat ID, Report Time)
    - 初始化数据库
    - 初始化服务
    - 启动 Telegram Bot (长轮询模式)
    - 启动 Scheduler（日报定时任务 + 每周备份）
    - 支持 CLI 模式：`--report` 参数直接发送日报（用于手动触发）
    - _Requirements: 8.3, 9.1, 11.1_

- [x] 13. Final Checkpoint - 完整功能验证
  - 确保所有测试通过
  - 验证完整工作流：/dy → /gx → /tasks → /wj → /admin
  - 验证每周备份功能
  - 如有问题请询问用户

## Notes

- 每个任务都引用了具体的需求条款以便追溯
- Checkpoint 任务用于阶段性验证
- 属性测试验证核心正确性属性
- 单元测试验证具体示例和边界情况
