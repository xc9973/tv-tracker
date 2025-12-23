package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"tv-tracker/internal/handler"
	"tv-tracker/internal/notify"
	"tv-tracker/internal/repository"
	"tv-tracker/internal/service"
	"tv-tracker/internal/tmdb"
)

// Config holds the application configuration
type Config struct {
	TMDBAPIKey       string
	TelegramBotToken string
	TelegramChatID   string
	DBPath           string
	Port             string
}

func main() {
	// Parse CLI flags
	reportMode := flag.Bool("report", false, "Send daily report and exit (for cron jobs)")
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

	// Initialize Telegram notifier (optional - may not be configured)
	var notifier *notify.TelegramNotifier
	if config.TelegramBotToken != "" && config.TelegramChatID != "" {
		notifier = notify.NewTelegramNotifier(config.TelegramBotToken, config.TelegramChatID)
	}

	// CLI mode: send daily report and exit
	// Requirements: 9.4 - Support CLI mode for cron jobs
	if *reportMode {
		if notifier == nil {
			log.Fatal("Telegram notifier not configured. Set TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_ID environment variables.")
		}

		// Get dashboard data for the report
		data, err := taskBoard.GetDashboardData()
		if err != nil {
			log.Fatalf("Failed to get dashboard data: %v", err)
		}

		// Send daily report
		if err := notifier.SendDailyReport(data.UpdateTasks); err != nil {
			log.Fatalf("Failed to send daily report: %v", err)
		}

		fmt.Println("Daily report sent successfully!")
		return
	}

	// Web server mode
	// Initialize handler
	h := handler.NewHandler(tmdbClient, subManager, taskGenerator, taskBoard, notifier)

	// Setup Gin router
	router := gin.Default()

	// Serve static files from web/dist
	webDistPath := filepath.Join("web", "dist")
	if _, err := os.Stat(webDistPath); err == nil {
		router.Static("/assets", filepath.Join(webDistPath, "assets"))
		router.StaticFile("/", filepath.Join(webDistPath, "index.html"))
		router.StaticFile("/vite.svg", filepath.Join(webDistPath, "vite.svg"))
		
		// Handle SPA routing - serve index.html for non-API routes
		router.NoRoute(func(c *gin.Context) {
			c.File(filepath.Join(webDistPath, "index.html"))
		})
	}

	// Register API routes
	h.RegisterRoutes(router)

	// Start server
	addr := ":" + config.Port
	log.Printf("Starting TV Tracker server on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// loadConfig loads configuration from environment variables
// Requirements: 8.3 - Load configuration on application start
func loadConfig() *Config {
	config := &Config{
		TMDBAPIKey:       getEnv("TMDB_API_KEY", ""),
		TelegramBotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramChatID:   getEnv("TELEGRAM_CHAT_ID", ""),
		DBPath:           getEnv("DB_PATH", "tv_tracker.db"),
		Port:             getEnv("PORT", "8080"),
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
