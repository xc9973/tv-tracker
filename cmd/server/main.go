package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"tv-tracker/internal/handler"
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
	WEBEnabled        bool
	WEBListenAddr     string
	WEBAPIToken       string
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

	disableBot, _ := strconv.ParseBool(getEnv("DISABLE_BOT", "false"))

	// Initialize Telegram Bot
	var bot *notify.TelegramBot
	if disableBot {
		log.Println("DISABLE_BOT=true; Telegram bot disabled")
	} else {
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

		newBot, err := notify.NewTelegramBot(config.TelegramBotToken, config.TelegramChatID, channelID, deps)
		if err != nil {
			log.Fatalf("Failed to create Telegram bot: %v", err)
		}
		bot = newBot
	}

	// CLI mode: send daily report and exit
	if *reportMode {
		if bot == nil {
			log.Fatal("Report mode requires Telegram bot; set DISABLE_BOT=false")
		}
		log.Println("Sending daily report...")
		if err := bot.SendDailyReport(); err != nil {
			log.Fatalf("Failed to send daily report: %v", err)
		}
		fmt.Println("Daily report sent successfully!")
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
	} else {
		log.Println("Scheduler disabled because Telegram bot is disabled")
	}

	// Optional HTTP server
	var httpServer *http.Server
	if config.WEBEnabled {
		if config.WEBAPIToken == "" {
			log.Fatal("WEB_ENABLED=true but WEB_API_TOKEN not set")
		}

		router := gin.Default()
		httpHandler := handler.NewHTTPHandler(
			tmdbClient,
			subManager,
			taskBoard,
			episodeRepo,
			showRepo,
			backupSvc,
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
			log.Printf("HTTP API listening on %s", config.WEBListenAddr)
			if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("HTTP server error: %v", err)
			}
		}()
	} else {
		log.Println("WEB_ENABLED=false; HTTP API disabled")
	}

	// Telegram bot
	if bot != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Printf("TV Tracker bot started. Chat ID: %d", config.TelegramChatID)
			bot.Start()
		}()
	}

	// Wait for shutdown signal
	<-ctx.Done()
	log.Println("Shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if scheduler != nil {
		scheduler.Stop()
	}
	if bot != nil {
		bot.Stop()
	}
	if httpServer != nil {
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}
	}

	wg.Wait()
	log.Println("Shutdown complete")
}

// loadConfig loads configuration from environment variables
func loadConfig() *Config {
	chatID, _ := strconv.ParseInt(getEnv("TELEGRAM_CHAT_ID", "0"), 10, 64)
	channelID, _ := strconv.ParseInt(getEnv("TELEGRAM_CHANNEL_ID", "0"), 10, 64)

	webEnabled, _ := strconv.ParseBool(getEnv("WEB_ENABLED", "false"))

	config := &Config{
		TMDBAPIKey:        getEnv("TMDB_API_KEY", ""),
		TelegramBotToken:  getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramChatID:    chatID,
		TelegramChannelID: channelID,
		DBPath:            getEnv("DB_PATH", "tv_tracker.db"),
		BackupDir:         getEnv("BACKUP_DIR", "backups"),
		ReportTime:        getEnv("REPORT_TIME", "08:00"),
		WEBEnabled:        webEnabled,
		WEBListenAddr:     getEnv("WEB_LISTEN_ADDR", ":18080"),
		WEBAPIToken:       getEnv("WEB_API_TOKEN", ""),
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
