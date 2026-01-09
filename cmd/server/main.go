package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"tv-tracker/internal/handler"
	"tv-tracker/internal/logger"
	"tv-tracker/internal/notify"
	"tv-tracker/internal/repository"
	"tv-tracker/internal/service"
	"tv-tracker/internal/tmdb"
)

// Config holds the application configuration
type Config struct {
	TMDBAPIKey        string
	TelegramBotToken  string
	TelegramChatID    int64 // 管理员个人 Chat ID
	TelegramChannelID int64 // 频道 ID，用于发送日报
	DBPath            string
	BackupDir         string
	ReportTime        string // Format: "HH:MM"
	ShutdownTimeout   time.Duration
	WEBEnabled        bool
	WEBListenAddr     string
	WEBAPIToken       string
}

func main() {
	// Parse CLI flags
	reportMode := flag.Bool("report", false, "Send daily report and exit")
	flag.Parse()

	// Initialize logger first
	isDev := getEnv("ENV", "production") == "development"
	if err := logger.InitLogger(isDev); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// Load configuration and validate after logger is initialized
	config := loadConfigAndValidate()

	// Initialize database
	db, err := repository.NewSQLiteDB(config.DBPath)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Initialize database schema
	if err := db.InitSchema(); err != nil {
		logger.Fatal("Failed to initialize database schema", zap.Error(err))
	}

	logger.Info("Database initialized", zap.String("path", config.DBPath))

	// Initialize repositories
	showRepo := repository.NewTVShowRepository(db)
	episodeRepo := repository.NewEpisodeRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	cacheRepo := repository.NewTMDBCacheRepository(db)

	// Initialize TMDB client
	tmdbClient := tmdb.NewClient(config.TMDBAPIKey)

	// Initialize services
	cacheSvc := service.NewTMDBCacheService(tmdbClient, cacheRepo)
	subManager := service.NewSubscriptionManager(tmdbClient, cacheSvc, showRepo, episodeRepo)
	taskGenerator := service.NewTaskGenerator(tmdbClient, cacheSvc, showRepo, episodeRepo, taskRepo)
	taskBoard := service.NewTaskBoardService(taskRepo, showRepo)
	backupSvc := service.NewBackupService(config.DBPath, config.BackupDir)
	showSync := service.NewShowSyncService(cacheSvc, taskGenerator, showRepo, episodeRepo)

	disableBot, _ := strconv.ParseBool(getEnv("DISABLE_BOT", "false"))

	// Initialize Telegram Bot
	var bot *notify.TelegramBot
	if disableBot {
		logger.Info("Telegram bot disabled", zap.Bool("disable_bot", true))
	} else {
		if config.TelegramBotToken == "" || config.TelegramChatID == 0 {
			logger.Fatal("Telegram bot not configured. Set TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_ID environment variables.")
		}

		deps := notify.Dependencies{
			TMDB:        tmdbClient,
			SubMgr:      subManager,
			TaskGen:     taskGenerator,
			TaskBoard:   taskBoard,
			EpisodeRepo: episodeRepo,
			BackupSvc:   backupSvc,
		}

		// 如果没有配置频道 ID，则使用管理员 ID 发送日报
		channelID := config.TelegramChannelID
		if channelID == 0 {
			channelID = config.TelegramChatID
		}

		newBot, err := notify.NewTelegramBot(config.TelegramBotToken, config.TelegramChatID, channelID, deps)
		if err != nil {
			logger.Fatal("Failed to create Telegram bot", zap.Error(err))
		}
		bot = newBot
		logger.Info("Telegram bot initialized", zap.Int64("chat_id", config.TelegramChatID))
	}

	// CLI mode: send daily report and exit
	if *reportMode {
		if bot == nil {
			logger.Fatal("Report mode requires Telegram bot; set DISABLE_BOT=false")
		}
		logger.Info("Sending daily report...")
		if err := bot.SendDailyReport(); err != nil {
			logger.Fatal("Failed to send daily report", zap.Error(err))
		}
		logger.Info("Daily report sent successfully!")
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup

	// Initialize scheduler (requires bot for daily report)
	var scheduler *service.Scheduler
	if bot != nil {
		scheduler = service.NewScheduler(bot, backupSvc, config.ReportTime)
		scheduler.Start()
		logger.Info("Scheduler started", zap.String("report_time", config.ReportTime))
	} else {
		logger.Info("Scheduler disabled because Telegram bot is disabled")
	}

	// Optional HTTP server
	var httpServer *http.Server
	if config.WEBEnabled {
		if config.WEBAPIToken == "" {
			logger.Fatal("WEB_ENABLED=true but WEB_API_TOKEN not set")
		}

		router := gin.Default()
		httpHandler := handler.NewHTTPHandler(
			tmdbClient,
			subManager,
			taskBoard,
			episodeRepo,
			showRepo,
			backupSvc,
			showSync,
			config.WEBAPIToken,
		)
		httpHandler.RegisterRoutes(router)

		httpServer = &http.Server{
			Addr:    config.WEBListenAddr,
			Handler: router,
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			logger.Info("HTTP API listening", zap.String("address", config.WEBListenAddr))
			if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Error("HTTP server error", zap.Error(err))
			}
		}()
	} else {
		logger.Info("HTTP API disabled", zap.Bool("web_enabled", false))
	}

	// Telegram bot
	if bot != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			logger.Info("TV Tracker bot started", zap.Int64("chat_id", config.TelegramChatID))
			bot.Start()
		}()
	}

	// Wait for shutdown signal
	<-ctx.Done()
	logger.Info("Received shutdown signal")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), config.ShutdownTimeout)
	defer cancel()

	if scheduler != nil {
		scheduler.Stop()
	}
	if bot != nil {
		bot.Stop()
	}
	if httpServer != nil {
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Error("HTTP server shutdown error", zap.Error(err))
		}
	}

	wg.Wait()
	logger.Info("Shutdown complete")
}

// loadConfigAndValidate loads configuration from environment variables and validates required fields
func loadConfigAndValidate() *Config {
	chatID, _ := strconv.ParseInt(getEnv("TELEGRAM_CHAT_ID", "0"), 10, 64)
	channelID, _ := strconv.ParseInt(getEnv("TELEGRAM_CHANNEL_ID", "0"), 10, 64)

	webEnabled, _ := strconv.ParseBool(getEnv("WEB_ENABLED", "false"))

	// Parse shutdown timeout (default: 5 seconds)
	shutdownTimeoutSecs, _ := strconv.Atoi(getEnv("SHUTDOWN_TIMEOUT", "5"))
	if shutdownTimeoutSecs <= 0 {
		shutdownTimeoutSecs = 5
	}

	config := &Config{
		TMDBAPIKey:        getEnv("TMDB_API_KEY", ""),
		TelegramBotToken:  getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramChatID:    chatID,
		TelegramChannelID: channelID,
		DBPath:            getEnv("DB_PATH", "tv_tracker.db"),
		BackupDir:         getEnv("BACKUP_DIR", "backups"),
		ReportTime:        getEnv("REPORT_TIME", "08:00"),
		ShutdownTimeout:   time.Duration(shutdownTimeoutSecs) * time.Second,
		WEBEnabled:        webEnabled,
		WEBListenAddr:     getEnv("WEB_LISTEN_ADDR", ":18080"),
		WEBAPIToken:       getEnv("WEB_API_TOKEN", ""),
	}

	// Validate required configuration using structured logger
	if config.TMDBAPIKey == "" {
		logger.Fatal("TMDB_API_KEY is required but not set",
			zap.String("hint", "Please set the TMDB_API_KEY environment variable"))
	}
	if config.WEBEnabled && config.WEBAPIToken == "" {
		logger.Fatal("WEB_API_TOKEN is required when WEB_ENABLED=true",
			zap.Bool("web_enabled", config.WEBEnabled),
			zap.String("hint", "Please set the WEB_API_TOKEN environment variable"))
	}

	return config
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
