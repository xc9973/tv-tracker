package handler

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"tv-tracker/internal/models"
	"tv-tracker/internal/repository"
	"tv-tracker/internal/service"
	"tv-tracker/internal/timeutil"
	"tv-tracker/internal/tmdb"
)

// HTTPHandler handles HTTP requests for the web interface
type HTTPHandler struct {
	tmdbClient  *tmdb.Client
	subMgr      *service.SubscriptionManager
	taskBoard   *service.TaskBoardService
	episodeRepo *repository.EpisodeRepository
	showRepo    *repository.TVShowRepository
	backupSvc   *service.BackupService
	apiToken    string
}

// NewHTTPHandler creates a new HTTPHandler
func NewHTTPHandler(
	tmdbClient *tmdb.Client,
	subMgr *service.SubscriptionManager,
	taskBoard *service.TaskBoardService,
	episodeRepo *repository.EpisodeRepository,
	showRepo *repository.TVShowRepository,
	backupSvc *service.BackupService,
	apiToken string,
) *HTTPHandler {
	return &HTTPHandler{
		tmdbClient:  tmdbClient,
		subMgr:      subMgr,
		taskBoard:   taskBoard,
		episodeRepo: episodeRepo,
		showRepo:    showRepo,
		backupSvc:   backupSvc,
		apiToken:    strings.TrimSpace(apiToken),
	}
}

// RegisterRoutes registers all HTTP routes
func (h *HTTPHandler) RegisterRoutes(r *gin.Engine) {
	// Serve simple web UI
	r.GET("/", func(c *gin.Context) {
		c.File("./web/simple/index.html")
	})

	api := r.Group("/api")
	api.Use(h.authMiddleware)

	// Health check must allow unauthenticated ping for probes
	r.GET("/api/health", h.Health)

	// Dashboard
	api.GET("/dashboard", h.GetDashboard)

	// Today's episodes
	api.GET("/today", h.GetTodayEpisodes)

	// Week calendar
	api.GET("/week", h.GetWeekEpisodes)

	// Search
	api.GET("/search", h.SearchTV)

	// Subscription
	api.POST("/subscribe", h.Subscribe)
	api.DELETE("/subscribe/:id", h.Unsubscribe)
	api.GET("/library", h.GetLibrary)

	// Tasks
	api.POST("/tasks/:id/complete", h.CompleteTask)
	api.POST("/tasks/:id/postpone", h.PostponeTask)

	// Resource time
	api.PUT("/shows/:id/resource-time", h.UpdateResourceTime)

	// Backups
	api.POST("/backup", func(c *gin.Context) {
		backupPath, err := h.backupSvc.Backup()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"backup_path": backupPath})
	})

}

// GetDashboard returns the dashboard data
func (h *HTTPHandler) GetDashboard(c *gin.Context) {
	data, err := h.taskBoard.GetDashboardData()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

// GetTodayEpisodes returns today's episodes
func (h *HTTPHandler) GetTodayEpisodes(c *gin.Context) {
	today := h.getParam(c, "date", "")
	if today == "" {
		today = "2006-01-02" // will be set below
	}

	// Get current date if not provided
	if today == "2006-01-02" {
		today = c.Query("date")
		if today == "" {
			now := timeutil.Now()
			today = now.Format("2006-01-02")
		}
	}

	episodes, err := h.episodeRepo.GetTodayEpisodesWithShowInfo(today)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if episodes == nil {
		episodes = []repository.TodayEpisodeInfo{}
	}

	c.JSON(http.StatusOK, gin.H{"episodes": episodes})
}

// GetWeekEpisodes returns this week's episodes grouped by date
func (h *HTTPHandler) GetWeekEpisodes(c *gin.Context) {
	now := timeutil.Now()

	// Calculate start of week (Monday)
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday = 7
	}
	startOfWeek := now.AddDate(0, 0, -(weekday - 1))

	// Build week data
	weekData := make(map[string][]repository.TodayEpisodeInfo)

	for i := 0; i < 7; i++ {
		date := startOfWeek.AddDate(0, 0, i)
		dateStr := date.Format("2006-01-02")

		episodes, err := h.episodeRepo.GetTodayEpisodesWithShowInfo(dateStr)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if episodes == nil {
			episodes = []repository.TodayEpisodeInfo{}
		}

		weekData[dateStr] = episodes
	}

	c.JSON(http.StatusOK, gin.H{
		"days":       weekData,
		"start_date": startOfWeek.Format("2006-01-02"),
	})
}

// SearchTV searches for TV shows
func (h *HTTPHandler) SearchTV(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter is required"})
		return
	}

	results, err := h.tmdbClient.SearchTV(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}

// Subscribe subscribes to a TV show
func (h *HTTPHandler) Subscribe(c *gin.Context) {
	var req struct {
		TMDBID int `json:"tmdb_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	show, alreadyExists, err := h.subMgr.Subscribe(req.TMDBID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if alreadyExists {
		c.JSON(http.StatusConflict, gin.H{"error": "already subscribed", "show": show})
		return
	}

	c.JSON(http.StatusOK, gin.H{"show": show})
}

// Unsubscribe unsubscribes from a TV show
func (h *HTTPHandler) Unsubscribe(c *gin.Context) {
	id := h.getIntParam(c, "id")
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid show id"})
		return
	}

	if err := h.subMgr.Unsubscribe(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "unsubscribed"})
}

// GetLibrary returns all subscribed shows
func (h *HTTPHandler) GetLibrary(c *gin.Context) {
	shows, err := h.subMgr.GetAllSubscriptions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if shows == nil {
		shows = []models.TVShow{}
	}

	c.JSON(http.StatusOK, gin.H{"shows": shows})
}

// CompleteTask marks a task as completed
func (h *HTTPHandler) CompleteTask(c *gin.Context) {
	taskID := h.getIntParam(c, "id")
	if taskID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	if err := h.taskBoard.CompleteTask(taskID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "task completed"})
}

// PostponeTask postpones a task to tomorrow
func (h *HTTPHandler) PostponeTask(c *gin.Context) {
	taskID := h.getIntParam(c, "id")
	if taskID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	if err := h.taskBoard.PostponeTask(taskID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "task postponed to tomorrow"})
}

// UpdateResourceTime updates the resource time for a TV show
func (h *HTTPHandler) UpdateResourceTime(c *gin.Context) {
	id := h.getIntParam(c, "id")
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid show id"})
		return
	}

	var req struct {
		ResourceTime string `json:"resource_time" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	show, err := h.showRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if show == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "show not found"})
		return
	}

	show.ResourceTime = req.ResourceTime
	if err := h.showRepo.Update(show); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"show": show})
}

// Health returns health status
func (h *HTTPHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// authMiddleware enforces Bearer token authentication against the configured API token.
func (h *HTTPHandler) authMiddleware(c *gin.Context) {
	expected := strings.TrimSpace(h.getAPIToken())
	if expected == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "WEB_API_TOKEN not set"})
		c.Abort()
		return
	}

	authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid Authorization header"})
		c.Abort()
		return
	}

	if subtle.ConstantTimeCompare([]byte(parts[1]), []byte(expected)) != 1 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		c.Abort()
		return
	}

	c.Next()
}

// Helper functions

func (h *HTTPHandler) getParam(c *gin.Context, key, defaultValue string) string {
	value := c.Param(key)
	if value == "" {
		value = c.Query(key)
	}
	if value == "" {
		return defaultValue
	}
	return value
}

func (h *HTTPHandler) getAPIToken() string {
	return h.apiToken
}

func (h *HTTPHandler) getIntParam(c *gin.Context, key string) int64 {
	value := c.Param(key)
	if value == "" {
		value = c.Query(key)
	}
	if value == "" {
		return 0
	}

	var id int64
	if _, err := fmt.Sscanf(value, "%d", &id); err != nil {
		return 0
	}
	return id
}
