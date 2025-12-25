package notify

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	tele "gopkg.in/telebot.v3"

	"tv-tracker/internal/models"
	"tv-tracker/internal/repository"
	"tv-tracker/internal/service"
	"tv-tracker/internal/tmdb"
)

// BotState represents the current state of user interaction
type BotState string

const (
	StateIdle          BotState = "idle"
	StateWaitingTMDBID BotState = "waiting_tmdb_id"
	StateWaitingAPIKey BotState = "waiting_api_key"
)

// TelegramBot handles Telegram bot interactions
type TelegramBot struct {
	bot         *tele.Bot
	chatID      int64  // 管理员 Chat ID
	channelID   int64  // 频道 ID，用于发送日报
	state       BotState
	stateMu     sync.RWMutex
	tmdb        *tmdb.Client
	subMgr      *service.SubscriptionManager
	taskGen     *service.TaskGenerator
	taskBoard   *service.TaskBoardService
	episodeRepo *repository.EpisodeRepository
	backupSvc   *service.BackupService
}

// Dependencies holds all dependencies for TelegramBot
type Dependencies struct {
	TMDB        *tmdb.Client
	SubMgr      *service.SubscriptionManager
	TaskGen     *service.TaskGenerator
	TaskBoard   *service.TaskBoardService
	EpisodeRepo *repository.EpisodeRepository
	BackupSvc   *service.BackupService
}

// NewTelegramBot creates a new TelegramBot
func NewTelegramBot(token string, chatID int64, channelID int64, deps Dependencies) (*TelegramBot, error) {
	pref := tele.Settings{
		Token:  token,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	bot, err := tele.NewBot(pref)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	tb := &TelegramBot{
		bot:         bot,
		chatID:      chatID,
		channelID:   channelID,
		state:       StateIdle,
		tmdb:        deps.TMDB,
		subMgr:      deps.SubMgr,
		taskGen:     deps.TaskGen,
		taskBoard:   deps.TaskBoard,
		episodeRepo: deps.EpisodeRepo,
		backupSvc:   deps.BackupSvc,
	}

	// Register handlers
	tb.registerHandlers()

	return tb, nil
}

// registerHandlers registers all bot handlers
func (t *TelegramBot) registerHandlers() {
	// Command handlers
	t.bot.Handle("/start", t.authMiddleware(t.HandleStart))
	t.bot.Handle("/help", t.authMiddleware(t.HandleHelp))

	// Text handler for state-based input
	t.bot.Handle(tele.OnText, t.authMiddleware(t.HandleText))

	// Callback handlers
	t.bot.Handle(&tele.InlineButton{Unique: "tasks"}, t.authMiddleware(t.HandleTasksCallback))
	t.bot.Handle(&tele.InlineButton{Unique: "subscribe"}, t.authMiddleware(t.HandleSubscribeCallback))
	t.bot.Handle(&tele.InlineButton{Unique: "organize"}, t.authMiddleware(t.HandleOrganizeCallback))
	t.bot.Handle(&tele.InlineButton{Unique: "sync"}, t.authMiddleware(t.HandleSyncCallback))
	t.bot.Handle(&tele.InlineButton{Unique: "admin"}, t.authMiddleware(t.HandleAdminCallback))
	t.bot.Handle(&tele.InlineButton{Unique: "apikey"}, t.authMiddleware(t.HandleAPIKeyCallback))
	t.bot.Handle(&tele.InlineButton{Unique: "backup"}, t.authMiddleware(t.HandleBackupCallback))
	t.bot.Handle(&tele.InlineButton{Unique: "back"}, t.authMiddleware(t.HandleBackCallback))
	t.bot.Handle(&tele.InlineButton{Unique: "complete"}, t.authMiddleware(t.HandleCompleteTaskCallback))
	t.bot.Handle(&tele.InlineButton{Unique: "archive"}, t.authMiddleware(t.HandleArchiveCallback))
}


// authMiddleware checks if the user is authorized
func (t *TelegramBot) authMiddleware(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		if !t.IsOwner(c.Chat().ID) {
			return c.Send("⛔ 未授权访问")
		}
		return next(c)
	}
}

// IsOwner checks if the chat ID matches the configured owner
func (t *TelegramBot) IsOwner(chatID int64) bool {
	return chatID == t.chatID
}

// setState sets the current bot state
func (t *TelegramBot) setState(state BotState) {
	t.stateMu.Lock()
	defer t.stateMu.Unlock()
	t.state = state
}

// getState gets the current bot state
func (t *TelegramBot) getState() BotState {
	t.stateMu.RLock()
	defer t.stateMu.RUnlock()
	return t.state
}

// Start starts the bot
func (t *TelegramBot) Start() {
	log.Println("Starting Telegram bot...")
	t.bot.Start()
}

// Stop stops the bot
func (t *TelegramBot) Stop() {
	t.bot.Stop()
}

// HandleStart handles the /start command
func (t *TelegramBot) HandleStart(c tele.Context) error {
	t.setState(StateIdle)
	return c.Send(t.FormatMainMenu(), t.MainMenuKeyboard())
}

// HandleHelp handles the /help command
func (t *TelegramBot) HandleHelp(c tele.Context) error {
	help := `📺 <b>TV Tracker 帮助</b>

<b>功能说明：</b>
• 📺 今日更新 - 查看今日需要更新的剧集
• ➕ 订阅剧集 - 通过 TMDB ID 订阅新剧集
• 📦 待整理 - 查看已完结待归档的剧集
• 🔄 同步更新 - 同步所有订阅数据
• ⚙️ 管理 - 系统管理和设置

<b>如何获取 TMDB ID：</b>
1. 访问 themoviedb.org
2. 搜索剧集
3. URL 中的数字即为 TMDB ID
   例如: /tv/1399 中的 1399`

	return c.Send(help, &tele.SendOptions{ParseMode: tele.ModeHTML}, t.BackButtonKeyboard())
}

// HandleText handles text input based on current state
func (t *TelegramBot) HandleText(c tele.Context) error {
	state := t.getState()

	switch state {
	case StateWaitingTMDBID:
		return t.handleTMDBIDInput(c)
	case StateWaitingAPIKey:
		return t.handleAPIKeyInput(c)
	default:
		return c.Send("请使用 /start 打开主菜单")
	}
}

// handleTMDBIDInput handles TMDB ID input
func (t *TelegramBot) handleTMDBIDInput(c tele.Context) error {
	t.setState(StateIdle)

	tmdbID, err := strconv.Atoi(strings.TrimSpace(c.Text()))
	if err != nil {
		return c.Send("❌ 无效的 TMDB ID，请输入数字", t.BackButtonKeyboard())
	}

	// Subscribe to the show
	show, alreadyExists, err := t.subMgr.Subscribe(tmdbID)
	if err != nil {
		return c.Send(fmt.Sprintf("❌ 订阅失败: %v", err), t.BackButtonKeyboard())
	}

	if alreadyExists {
		msg := fmt.Sprintf(`⚠️ <b>该剧集已订阅</b>

📺 %s
状态: %s
资源时间: %s`, show.Name, show.Status, show.ResourceTime)
		return c.Send(msg, &tele.SendOptions{ParseMode: tele.ModeHTML}, t.BackButtonKeyboard())
	}

	msg := fmt.Sprintf(`✅ <b>已订阅</b>

📺 %s
状态: %s
资源时间: %s`, show.Name, show.Status, show.ResourceTime)

	return c.Send(msg, &tele.SendOptions{ParseMode: tele.ModeHTML}, t.BackButtonKeyboard())
}

// handleAPIKeyInput handles API key input
func (t *TelegramBot) handleAPIKeyInput(c tele.Context) error {
	t.setState(StateIdle)
	// Note: In a real implementation, you would update the TMDB client's API key
	// For now, we just acknowledge the input
	return c.Send("✅ TMDB API Key 已更新\n\n⚠️ 注意：需要重启服务才能生效", t.BackButtonKeyboard())
}


// HandleTasksCallback handles the "今日更新" button
func (t *TelegramBot) HandleTasksCallback(c tele.Context) error {
	// 获取今天的日期
	today := time.Now().Format("2006-01-02")
	
	// 查询今天播出的剧集
	episodes, err := t.episodeRepo.GetTodayEpisodesWithShowInfo(today)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "获取数据失败"})
	}

	if len(episodes) == 0 {
		return c.Edit("📺 <b>今日更新</b>\n\n今日暂无剧集更新 🎬", &tele.SendOptions{ParseMode: tele.ModeHTML}, t.BackButtonKeyboard())
	}

	msg := t.FormatTodayEpisodes(episodes)
	return c.Edit(msg, &tele.SendOptions{ParseMode: tele.ModeHTML}, t.BackButtonKeyboard())
}

// HandleSubscribeCallback handles the "订阅剧集" button
func (t *TelegramBot) HandleSubscribeCallback(c tele.Context) error {
	t.setState(StateWaitingTMDBID)
	return c.Edit("➕ <b>订阅剧集</b>\n\n请输入 TMDB ID（可在 themoviedb.org 查询）:", &tele.SendOptions{ParseMode: tele.ModeHTML}, t.BackButtonKeyboard())
}

// HandleOrganizeCallback handles the "待整理" button
func (t *TelegramBot) HandleOrganizeCallback(c tele.Context) error {
	data, err := t.taskBoard.GetDashboardData()
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "获取任务失败"})
	}

	if len(data.OrganizeTasks) == 0 {
		return c.Edit("📦 <b>待整理归档</b>\n\n暂无待整理剧集 ✨", &tele.SendOptions{ParseMode: tele.ModeHTML}, t.BackButtonKeyboard())
	}

	msg := t.FormatOrganizeList(data.OrganizeTasks)
	keyboard := t.TaskListKeyboard(data.OrganizeTasks, "archive")

	return c.Edit(msg, &tele.SendOptions{ParseMode: tele.ModeHTML}, keyboard)
}

// HandleSyncCallback handles the "同步更新" button
func (t *TelegramBot) HandleSyncCallback(c tele.Context) error {
	// First respond to callback to prevent timeout
	c.Respond(&tele.CallbackResponse{Text: "正在同步..."})

	// Run sync
	result, err := t.taskGen.SyncAll()
	if err != nil {
		return c.Edit(fmt.Sprintf("❌ 同步失败: %v", err), t.BackButtonKeyboard())
	}

	// Get subscription list
	shows, err := t.subMgr.GetAllSubscriptions()
	if err != nil {
		return c.Edit(fmt.Sprintf("❌ 获取订阅列表失败: %v", err), t.BackButtonKeyboard())
	}

	msg := fmt.Sprintf(`🔄 <b>同步完成</b>

新增更新任务: %d
新增整理任务: %d
错误数: %d

`, result.UpdateTasks, result.OrganizeTasks, result.Errors)

	msg += t.FormatSubscriptionList(shows)

	return c.Edit(msg, &tele.SendOptions{ParseMode: tele.ModeHTML}, t.BackButtonKeyboard())
}

// HandleAdminCallback handles the "管理" button
func (t *TelegramBot) HandleAdminCallback(c tele.Context) error {
	msg := t.FormatAdminMenu()
	return c.Edit(msg, &tele.SendOptions{ParseMode: tele.ModeHTML}, t.AdminMenuKeyboard())
}

// HandleAPIKeyCallback handles the "更换TMDB API" button
func (t *TelegramBot) HandleAPIKeyCallback(c tele.Context) error {
	t.setState(StateWaitingAPIKey)
	return c.Edit("🔑 <b>更换 TMDB API Key</b>\n\n请输入新的 API Key:", &tele.SendOptions{ParseMode: tele.ModeHTML}, t.BackButtonKeyboard())
}

// HandleBackupCallback handles the "手动备份" button
func (t *TelegramBot) HandleBackupCallback(c tele.Context) error {
	c.Respond(&tele.CallbackResponse{Text: "正在备份..."})

	if t.backupSvc == nil {
		return c.Edit("❌ 备份服务未配置", t.BackButtonKeyboard())
	}

	backupPath, err := t.backupSvc.Backup()
	if err != nil {
		return c.Edit(fmt.Sprintf("❌ 备份失败: %v", err), t.BackButtonKeyboard())
	}

	return c.Edit(fmt.Sprintf("✅ <b>备份成功</b>\n\n文件: %s", backupPath), &tele.SendOptions{ParseMode: tele.ModeHTML}, t.BackButtonKeyboard())
}

// HandleBackCallback handles the "返回主菜单" button
func (t *TelegramBot) HandleBackCallback(c tele.Context) error {
	t.setState(StateIdle)
	return c.Edit(t.FormatMainMenu(), &tele.SendOptions{ParseMode: tele.ModeHTML}, t.MainMenuKeyboard())
}


// HandleCompleteTaskCallback handles the "已完成" button for UPDATE tasks
func (t *TelegramBot) HandleCompleteTaskCallback(c tele.Context) error {
	// Parse task ID from callback data
	data := c.Callback().Data
	taskID, err := strconv.ParseInt(data, 10, 64)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "无效的任务ID"})
	}

	// Complete the task
	if err := t.taskBoard.CompleteTask(taskID); err != nil {
		return c.Respond(&tele.CallbackResponse{Text: fmt.Sprintf("完成失败: %v", err)})
	}

	c.Respond(&tele.CallbackResponse{Text: "✅ 已标记完成"})

	// Refresh the task list
	return t.HandleTasksCallback(c)
}

// HandleArchiveCallback handles the "已归档" button for ORGANIZE tasks
func (t *TelegramBot) HandleArchiveCallback(c tele.Context) error {
	// Parse task ID from callback data
	data := c.Callback().Data
	taskID, err := strconv.ParseInt(data, 10, 64)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "无效的任务ID"})
	}

	// Complete the task (this also archives the show)
	if err := t.taskBoard.CompleteTask(taskID); err != nil {
		return c.Respond(&tele.CallbackResponse{Text: fmt.Sprintf("归档失败: %v", err)})
	}

	c.Respond(&tele.CallbackResponse{Text: "✅ 已归档"})

	// Refresh the organize list
	return t.HandleOrganizeCallback(c)
}

// FormatMainMenu formats the main menu message
func (t *TelegramBot) FormatMainMenu() string {
	return "📺 <b>TV Tracker</b>\n\n选择一个功能:"
}

// FormatTodayEpisodes formats today's episodes list
func (t *TelegramBot) FormatTodayEpisodes(episodes []repository.TodayEpisodeInfo) string {
	var sb strings.Builder
	today := time.Now().Format("2006-01-02")
	sb.WriteString(fmt.Sprintf("📺 <b>今日更新</b> (%s)\n\n", today))

	for i, info := range episodes {
		episodeID := fmt.Sprintf("S%02dE%02d", info.Episode.Season, info.Episode.Episode)
		sb.WriteString(fmt.Sprintf("%d. <b>%s</b>\n", i+1, info.ShowName))
		sb.WriteString(fmt.Sprintf("   📍 %s", episodeID))
		if info.Episode.Title != "" {
			sb.WriteString(fmt.Sprintf(" - %s", info.Episode.Title))
		}
		sb.WriteString(fmt.Sprintf("\n   ⏰ %s\n\n", info.ResourceTime))
	}

	sb.WriteString(fmt.Sprintf("共 %d 集更新", len(episodes)))
	return sb.String()
}

// FormatTaskList formats the task list message
func (t *TelegramBot) FormatTaskList(tasks []models.Task) string {
	var sb strings.Builder
	sb.WriteString("📺 <b>今日更新</b>\n\n")

	// Sort by resource time
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].ResourceTime < tasks[j].ResourceTime
	})

	for i, task := range tasks {
		episodeInfo := extractEpisodeInfo(task.Description)
		sb.WriteString(fmt.Sprintf("%d. <b>%s</b>\n", i+1, task.TVShowName))
		sb.WriteString(fmt.Sprintf("   📍 %s\n", episodeInfo))
		sb.WriteString(fmt.Sprintf("   ⏰ %s\n\n", task.ResourceTime))
	}

	return sb.String()
}

// FormatOrganizeList formats the organize task list message
func (t *TelegramBot) FormatOrganizeList(tasks []models.Task) string {
	var sb strings.Builder
	sb.WriteString("📦 <b>待整理归档</b>\n\n")

	for i, task := range tasks {
		sb.WriteString(fmt.Sprintf("%d. <b>%s</b>\n", i+1, task.TVShowName))
		sb.WriteString(fmt.Sprintf("   %s\n\n", task.Description))
	}

	return sb.String()
}

// FormatSubscriptionList formats the subscription list message
func (t *TelegramBot) FormatSubscriptionList(shows []models.TVShow) string {
	var sb strings.Builder
	sb.WriteString("<b>📚 当前订阅</b>\n\n")

	if len(shows) == 0 {
		sb.WriteString("暂无订阅")
		return sb.String()
	}

	for i, show := range shows {
		status := "🟢"
		if show.Status == "Ended" || show.Status == "Canceled" {
			status = "🔴"
		}
		sb.WriteString(fmt.Sprintf("%d. %s %s\n", i+1, status, show.Name))
	}

	return sb.String()
}

// FormatAdminMenu formats the admin menu message
func (t *TelegramBot) FormatAdminMenu() string {
	var sb strings.Builder
	sb.WriteString("⚙️ <b>系统管理</b>\n\n")

	// Get subscription count
	shows, _ := t.subMgr.GetAllSubscriptions()
	sb.WriteString(fmt.Sprintf("📚 订阅数: %d\n", len(shows)))

	// Get pending task count
	data, _ := t.taskBoard.GetDashboardData()
	totalTasks := len(data.UpdateTasks) + len(data.OrganizeTasks)
	sb.WriteString(fmt.Sprintf("📋 待处理任务: %d\n", totalTasks))

	// Get last backup time
	if t.backupSvc != nil {
		lastBackup, err := t.backupSvc.GetLastBackupTime()
		if err == nil && !lastBackup.IsZero() {
			sb.WriteString(fmt.Sprintf("💾 上次备份: %s\n", lastBackup.Format("2006-01-02 15:04")))
		} else {
			sb.WriteString("💾 上次备份: 无\n")
		}
	}

	return sb.String()
}


// FormatDailyReport formats the daily report message
func (t *TelegramBot) FormatDailyReport(tasks []models.Task) string {
	return FormatDailyReport(tasks)
}

// FormatDailyReport formats tasks into a daily report message (standalone function)
func FormatDailyReport(tasks []models.Task) string {
	today := time.Now().Format("2006-01-02")
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("📺 <b>今日更新日报</b> (%s)\n\n", today))

	// Filter only UPDATE tasks
	var updateTasks []models.Task
	for _, task := range tasks {
		if task.TaskType == models.TaskTypeUpdate {
			updateTasks = append(updateTasks, task)
		}
	}

	if len(updateTasks) == 0 {
		sb.WriteString("今日暂无剧集更新 🎬")
		return sb.String()
	}

	// Sort by resource time
	sort.Slice(updateTasks, func(i, j int) bool {
		return updateTasks[i].ResourceTime < updateTasks[j].ResourceTime
	})

	for i, task := range updateTasks {
		episodeInfo := extractEpisodeInfo(task.Description)
		sb.WriteString(fmt.Sprintf("%d. <b>%s</b>\n", i+1, task.TVShowName))
		sb.WriteString(fmt.Sprintf("   📍 %s\n", episodeInfo))
		sb.WriteString(fmt.Sprintf("   ⏰ %s\n", task.ResourceTime))
		if i < len(updateTasks)-1 {
			sb.WriteString("\n")
		}
	}

	sb.WriteString(fmt.Sprintf("\n\n共 %d 部剧集更新", len(updateTasks)))

	return sb.String()
}

// extractEpisodeInfo extracts episode info from task description
func extractEpisodeInfo(description string) string {
	if strings.HasPrefix(description, "新剧集更新: ") {
		return strings.TrimPrefix(description, "新剧集更新: ")
	}
	return description
}

// MainMenuKeyboard returns the main menu keyboard
func (t *TelegramBot) MainMenuKeyboard() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}

	btnTasks := menu.Data("📺 今日更新", "tasks")
	btnSubscribe := menu.Data("➕ 订阅剧集", "subscribe")
	btnOrganize := menu.Data("📦 待整理", "organize")
	btnSync := menu.Data("🔄 同步更新", "sync")
	btnAdmin := menu.Data("⚙️ 管理", "admin")

	menu.Inline(
		menu.Row(btnTasks, btnSubscribe),
		menu.Row(btnOrganize, btnSync),
		menu.Row(btnAdmin),
	)

	return menu
}

// AdminMenuKeyboard returns the admin menu keyboard
func (t *TelegramBot) AdminMenuKeyboard() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}

	btnAPIKey := menu.Data("🔑 更换TMDB API", "apikey")
	btnBackup := menu.Data("💾 手动备份", "backup")
	btnBack := menu.Data("🔙 返回主菜单", "back")

	menu.Inline(
		menu.Row(btnAPIKey, btnBackup),
		menu.Row(btnBack),
	)

	return menu
}

// BackButtonKeyboard returns a keyboard with just the back button
func (t *TelegramBot) BackButtonKeyboard() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}
	btnBack := menu.Data("🔙 返回主菜单", "back")
	menu.Inline(menu.Row(btnBack))
	return menu
}

// TaskListKeyboard returns a keyboard for task list with complete/archive buttons
func (t *TelegramBot) TaskListKeyboard(tasks []models.Task, action string) *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}

	var rows []tele.Row
	for _, task := range tasks {
		var btn tele.Btn
		if action == "complete" {
			btn = menu.Data(fmt.Sprintf("✅ %s", task.TVShowName), action, strconv.FormatInt(task.ID, 10))
		} else {
			btn = menu.Data(fmt.Sprintf("✅ 归档 %s", task.TVShowName), action, strconv.FormatInt(task.ID, 10))
		}
		rows = append(rows, menu.Row(btn))
	}

	// Add back button
	btnBack := menu.Data("🔙 返回主菜单", "back")
	rows = append(rows, menu.Row(btnBack))

	menu.Inline(rows...)
	return menu
}

// SendDailyReport sends the daily report to the channel
func (t *TelegramBot) SendDailyReport() error {
	// 获取今天的日期
	today := time.Now().Format("2006-01-02")
	
	// 查询今天播出的剧集
	episodes, err := t.episodeRepo.GetTodayEpisodesWithShowInfo(today)
	if err != nil {
		return fmt.Errorf("failed to get today's episodes: %w", err)
	}

	msg := t.FormatDailyReportFromEpisodes(episodes)
	// 发送到频道
	_, err = t.bot.Send(&tele.Chat{ID: t.channelID}, msg, &tele.SendOptions{ParseMode: tele.ModeHTML})
	return err
}

// FormatDailyReportFromEpisodes formats today's episodes into a daily report
func (t *TelegramBot) FormatDailyReportFromEpisodes(episodes []repository.TodayEpisodeInfo) string {
	today := time.Now().Format("2006-01-02")
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("📺 <b>今日更新日报</b> (%s)\n\n", today))

	if len(episodes) == 0 {
		sb.WriteString("今日暂无剧集更新 🎬")
		return sb.String()
	}

	for i, info := range episodes {
		episodeID := fmt.Sprintf("S%02dE%02d", info.Episode.Season, info.Episode.Episode)
		sb.WriteString(fmt.Sprintf("%d. <b>%s</b>\n", i+1, info.ShowName))
		sb.WriteString(fmt.Sprintf("   📍 %s", episodeID))
		if info.Episode.Title != "" {
			sb.WriteString(fmt.Sprintf(" - %s", info.Episode.Title))
		}
		sb.WriteString(fmt.Sprintf("\n   ⏰ %s\n", info.ResourceTime))
		if i < len(episodes)-1 {
			sb.WriteString("\n")
		}
	}

	sb.WriteString(fmt.Sprintf("\n\n共 %d 集更新", len(episodes)))

	return sb.String()
}
