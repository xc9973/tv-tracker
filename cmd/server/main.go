package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"tv-tracker/internal/notify"
	"tv-tracker/internal/repository"
	"tv-tracker/internal/service"
	"tv-tracker/internal/tmdb"
)

// Config holds the application configuration
type Config struct {
	TMDBAPIKey        string
	TelegramBotToken  string
	TelegramChatID    int64  // 管理员个人 Chat ID
	TelegramChannelID int64  // 频道 ID，用于发送日报
	DBPath            string
	BackupDir         string
	ReportTime        string // Format: "HH:MM"
}

func main() {
	// Parse CLI flags
	reportMode := flag.Bool("report", false, "Send daily report and exit")
	flag.Parse()

	// Load configuration
	config := loadConfig()

	// Initialize database
	db, err := repository.NewSQLiteDB(config.DBPath)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize database schema
	if err := db.InitSchema(); err != nil {
		log.Fatalf("Failed to initialize database schema: %v", err)
	}

	// Initialize repositories
	showRepo := repository.NewTVShowRepository(db)
	episodeRepo := repository.NewEpisodeRepository(db)
	taskRepo := repository.NewTaskRepository(db)

	// Initialize TMDB client
	tmdbClient := tmdb.NewClient(config.TMDBAPIKey)

	// Initialize services
	subManager := service.NewSubscriptionManager(tmdbClient, showRepo, episodeRepo)
	taskGenerator := service.NewTaskGenerator(tmdbClient, showRepo, episodeRepo, taskRepo)
	taskBoard := service.NewTaskBoardService(taskRepo, showRepo)
	backupSvc := service.NewBackupService(config.DBPath, config.BackupDir)

	// Initialize Telegram Bot
	if config.TelegramBotToken == "" || config.TelegramChatID == 0 {
		log.Fatal("Telegram bot not configured. Set TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_ID environment variables.")
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

	bot, err := notify.NewTelegramBot(config.TelegramBotToken, config.TelegramChatID, channelID, deps)
	if err != nil {
		log.Fatalf("Failed to create Telegram bot: %v", err)
	}

	// CLI mode: send daily report and exit
	if *reportMode {
		log.Println("Sending daily report...")
		if err := bot.SendDailyReport(); err != nil {
			log.Fatalf("Failed to send daily report: %v", err)
		}
		fmt.Println("Daily report sent successfully!")
		return
	}

	// Initialize scheduler
	scheduler := service.NewScheduler(bot, backupSvc, config.ReportTime)
	scheduler.Start()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down...")
		scheduler.Stop()
		bot.Stop()
		os.Exit(0)
	}()

	// Start bot (blocking)
	log.Printf("TV Tracker bot started. Chat ID: %d", config.TelegramChatID)
	bot.Start()
}


// loadConfig loads configuration from environment variables
func loadConfig() *Config {
	chatID, _ := strconv.ParseInt(getEnv("TELEGRAM_CHAT_ID", "0"), 10, 64)
	channelID, _ := strconv.ParseInt(getEnv("TELEGRAM_CHANNEL_ID", "0"), 10, 64)

	config := &Config{
		TMDBAPIKey:        getEnv("TMDB_API_KEY", ""),
		TelegramBotToken:  getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramChatID:    chatID,
		TelegramChannelID: channelID,
		DBPath:            getEnv("DB_PATH", "tv_tracker.db"),
		BackupDir:         getEnv("BACKUP_DIR", "backups"),
		ReportTime:        getEnv("REPORT_TIME", "08:00"),
	}

	// Validate required configuration
	if config.TMDBAPIKey == "" {
		log.Println("Warning: TMDB_API_KEY not set. TMDB API calls will fail.")
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
