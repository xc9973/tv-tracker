# Requirements Document

## Introduction

TV Tracker 是一个个人使用的 Telegram Bot 应用，部署在服务器上通过 Telegram 消息交互，用于管理 Emby 媒体库的影视剧订阅和更新追踪。系统集成 TMDB API 获取元数据，自动在剧集更新日生成"更新提醒"任务，并通过 Telegram Bot 发送每日更新日报。当剧集完结时提醒用户进行本地文件整理归档。

## Glossary

- **TV_Tracker**: 影视剧订阅追踪系统的主应用（Telegram Bot）
- **TMDB_Client**: 与 TMDB API 交互的客户端模块
- **Subscription_Manager**: 管理用户剧集订阅的模块
- **Task_Generator**: 根据剧集状态生成任务的模块
- **Bot**: Telegram Bot，用户通过发送消息/命令与系统交互
- **Notifier**: Telegram 通知模块
- **TVShow**: 订阅的剧集数据模型
- **Task**: 待办任务数据模型
- **UPDATE_Task**: 剧集更新提醒类型的任务（提醒用户有新剧集可下载/入库）
- **ORGANIZE_Task**: 整理归档类型的任务
- **Resource_Time**: 资源预计可用时间（根据国家/地区自动推断）

## Requirements

### Requirement 1: TMDB 剧集搜索

**User Story:** As a user, I want to search for TV shows via TMDB, so that I can find and subscribe to shows I want to track.

#### Acceptance Criteria

1. WHEN a user enters a search keyword, THE TMDB_Client SHALL call the TMDB /search/tv API and return matching results
2. WHEN search results are returned, THE TV_Tracker SHALL display show name, poster, and first air date for each result
3. WHEN the TMDB API returns an error, THE TMDB_Client SHALL return a descriptive error message to the user
4. WHEN no results are found, THE TV_Tracker SHALL display a "no results found" message

### Requirement 2: 剧集订阅管理

**User Story:** As a user, I want to subscribe to TV shows, so that the system can track updates for me.

#### Acceptance Criteria

1. WHEN a user clicks subscribe on a search result, THE Subscription_Manager SHALL fetch detailed show info from TMDB /tv/{id} API
2. WHEN subscribing to a show, THE Subscription_Manager SHALL store tmdb_id, name, status, and next_air_date in the database
3. WHEN a show is already subscribed, THE TV_Tracker SHALL prevent duplicate subscription and notify the user
4. WHEN viewing the library, THE TV_Tracker SHALL display all subscribed shows with their current status

### Requirement 3: 数据同步

**User Story:** As a user, I want to sync my subscriptions with TMDB, so that I have the latest show information.

#### Acceptance Criteria

1. WHEN a user clicks the sync button, THE Task_Generator SHALL iterate through all non-archived subscriptions
2. WHEN syncing a show, THE TMDB_Client SHALL fetch the latest data from TMDB /tv/{id} API
3. WHEN sync completes, THE TV_Tracker SHALL update the local database with the latest show status and next_air_date
4. IF the TMDB API is unavailable during sync, THEN THE TV_Tracker SHALL log the error and continue with remaining shows

### Requirement 4: 剧集更新提醒自动生成

**User Story:** As a user, I want the system to automatically create update reminders when new episodes air, so that I know when to download new content for my Emby library.

#### Acceptance Criteria

1. WHEN a show's next_episode_to_air has air_date equal to today, THE Task_Generator SHALL create an UPDATE_Task
2. WHEN creating an UPDATE_Task, THE Task_Generator SHALL include the episode identifier (SxxExx format) in the task description
3. WHEN an UPDATE_Task already exists for the same show and episode, THE Task_Generator SHALL NOT create a duplicate task
4. WHEN a show's air_date is in the past and no task exists, THE Task_Generator SHALL create an UPDATE_Task for the missed episode

### Requirement 5: 完结整理任务生成

**User Story:** As a user, I want to be notified when a show ends, so that I can organize and archive my local files.

#### Acceptance Criteria

1. WHEN a show's status changes to "Ended" or "Canceled", THE Task_Generator SHALL create an ORGANIZE_Task
2. WHEN creating an ORGANIZE_Task, THE Task_Generator SHALL include a message indicating the show has ended and needs archiving
3. WHEN an ORGANIZE_Task already exists for a show, THE Task_Generator SHALL NOT create a duplicate task
4. WHILE a show is archived (is_archived = True), THE Task_Generator SHALL skip it during sync operations

### Requirement 6: 任务完成处理

**User Story:** As a user, I want to mark tasks as complete, so that I can track my progress.

#### Acceptance Criteria

1. WHEN a user marks an UPDATE_Task as complete, THE Task_Board SHALL set is_completed to True
2. WHEN a user marks an ORGANIZE_Task as complete, THE Task_Board SHALL set is_completed to True AND set the associated TVShow.is_archived to True
3. WHEN a show is archived, THE TV_Tracker SHALL exclude it from future sync operations
4. WHEN viewing the task board, THE TV_Tracker SHALL display completed and pending tasks separately

### Requirement 7: Telegram Bot 交互

**User Story:** As a user, I want to interact with the system via Telegram Bot with button-based menu, so that I can manage my Emby library updates easily from my phone.

#### Acceptance Criteria

1. WHEN sending `/start` command, THE Bot SHALL display a main menu with inline buttons for all functions
2. WHEN clicking "📺 今日更新" button, THE Bot SHALL reply with all pending UPDATE_Tasks, each with a "✅ 已完成" button
3. WHEN clicking "✅ 已完成" button on an UPDATE_Task, THE Bot SHALL mark the task as completed and update the display
4. WHEN clicking "➕ 订阅剧集" button, THE Bot SHALL prompt user to input TMDB ID, then subscribe to the show
5. WHEN clicking "📦 待整理" button, THE Bot SHALL reply with all ended/canceled shows, each with a "✅ 已归档" button
6. WHEN clicking "✅ 已归档" button on an ORGANIZE_Task, THE Bot SHALL mark the task as completed, archive the show, and update the display
7. WHEN clicking "🔄 同步更新" button, THE Bot SHALL sync all active subscriptions and reply with the subscription list
8. WHEN clicking "⚙️ 管理" button, THE Bot SHALL display admin menu with system status, TMDB API config, and backup options
9. WHEN displaying tasks, THE Bot SHALL show the associated show name, task description, and resource time
10. WHEN the Bot starts, THE Bot SHALL only respond to messages from the configured Chat ID (owner only)

### Requirement 11: 数据库备份

**User Story:** As a user, I want the database to be backed up automatically, so that I don't lose my data.

#### Acceptance Criteria

1. THE TV_Tracker SHALL automatically backup the SQLite database once per week
2. WHEN creating a backup, THE TV_Tracker SHALL save the backup file with a timestamp in the filename
3. THE TV_Tracker SHALL retain the last 4 weekly backups and delete older ones
4. WHEN sending `/admin` command, THE Bot SHALL show the last backup time and allow manual backup trigger

### Requirement 8: 数据持久化

**User Story:** As a user, I want my subscriptions and tasks to be saved, so that I don't lose my data.

#### Acceptance Criteria

1. THE TV_Tracker SHALL store all TVShow records in a SQLite database
2. THE TV_Tracker SHALL store all Task records in a SQLite database with foreign key reference to TVShow
3. WHEN the application starts, THE TV_Tracker SHALL load existing data from the SQLite database
4. WHEN data is modified, THE TV_Tracker SHALL persist changes to the SQLite database immediately

### Requirement 9: Telegram 日报通知

**User Story:** As a user, I want to receive daily update reports via Telegram automatically every morning, so that I know what shows are airing today without manually checking.

#### Acceptance Criteria

1. THE TV_Tracker SHALL automatically send a daily report to the configured Telegram channel every morning at a configured time
2. WHEN generating a daily report, THE Notifier SHALL query all episodes with air_date equal to today
3. WHEN there are updates today, THE Notifier SHALL format a message containing show name, episode info (SxxExx), and resource time
4. WHEN there are no updates today, THE Notifier SHALL send a message indicating no updates
5. WHEN sending a Telegram message, THE Notifier SHALL use the configured Bot Token and Chat ID
6. IF the Telegram API fails, THEN THE Notifier SHALL log the error and continue operation

### Requirement 10: 资源时间自动推断

**User Story:** As a user, I want the system to automatically estimate when resources will be available based on the show's origin country, so that I know when to check for downloads.

#### Acceptance Criteria

1. WHEN subscribing to a US/UK/CA show, THE Subscription_Manager SHALL set resource_time to "18:00"
2. WHEN subscribing to a CN/TW show, THE Subscription_Manager SHALL set resource_time to "20:00"
3. WHEN subscribing to a JP/KR show, THE Subscription_Manager SHALL set resource_time to "23:00"
4. WHEN subscribing to a show from other countries, THE Subscription_Manager SHALL set resource_time to "待定"
5. WHEN displaying tasks, THE Task_Board SHALL show the resource_time alongside the episode info
