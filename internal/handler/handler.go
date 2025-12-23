package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"tv-tracker/internal/models"
	"tv-tracker/internal/notify"
	"tv-tracker/internal/service"
	"tv-tracker/internal/tmdb"
)

// Handler handles all HTTP requests
type Handler struct {
	tmdbClient      *tmdb.Client
	subManager      *service.SubscriptionManager
	taskGenerator   *service.TaskGenerator
	taskBoard       *service.TaskBoardService
	notifier        *notify.TelegramNotifier
}

// NewHandler creates a new Handler
func NewHandler(
	tmdbClient *tmdb.Client,
	subManager *service.SubscriptionManager,
	taskGenerator *service.TaskGenerator,
	taskBoard *service.TaskBoardService,
	notifier *notify.TelegramNotifier,
) *Handler {
	return &Handler{
		tmdbClient:    tmdbClient,
		subManager:    subManager,
		taskGenerator: taskGenerator,
		taskBoard:     taskBoard,
		notifier:      notifier,
	}
}

// RegisterRoutes registers all API routes
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.GET("/dashboard", h.GetDashboard)
		api.GET("/search", h.SearchTV)
		api.POST("/subscribe", h.Subscribe)
		api.POST("/sync", h.Sync)
		api.POST("/tasks/:id/complete", h.CompleteTask)
		api.GET("/library", h.GetLibrary)
		api.POST("/report", h.SendReport)
	}
}

// GetDashboard returns the dashboard data with pending tasks
// GET /api/dashboard
// Requirements: 7.1, 7.2
func (h *Handler) GetDashboard(c *gin.Context) {
	data, err := h.taskBoard.GetDashboardData()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get dashboard data: " + err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, data)
}


// SearchTV searches for TV shows via TMDB API
// GET /api/search?q=<query>
// Requirements: 1.1, 1.4
func (h *Handler) SearchTV(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Search query is required",
		})
		return
	}

	results, err := h.tmdbClient.SearchTV(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to search TV shows: " + err.Error(),
		})
		return
	}

	// Return empty array if no results (Requirement 1.4)
	if results == nil {
		results = []tmdb.SearchResult{}
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
	})
}

// SubscribeRequest represents the request body for subscribing to a show
type SubscribeRequest struct {
	TMDBID int `json:"tmdb_id" binding:"required"`
}

// Subscribe subscribes to a TV show
// POST /api/subscribe
// Requirements: 2.3
func (h *Handler) Subscribe(c *gin.Context) {
	var req SubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: tmdb_id is required",
		})
		return
	}

	// Check if already subscribed (Requirement 2.3)
	if h.subManager.IsSubscribed(req.TMDBID) {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "Already subscribed to this show",
			"message": "该剧集已订阅",
		})
		return
	}

	show, err := h.subManager.Subscribe(req.TMDBID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to subscribe: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Successfully subscribed",
		"show":    show,
	})
}

// Sync triggers a manual sync of all subscriptions
// POST /api/sync
func (h *Handler) Sync(c *gin.Context) {
	result, err := h.taskGenerator.SyncAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to sync: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Sync completed",
		"result":  result,
	})
}

// CompleteTask marks a task as completed
// POST /api/tasks/:id/complete
// Requirements: 7.1, 7.2
func (h *Handler) CompleteTask(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid task ID",
		})
		return
	}

	if err := h.taskBoard.CompleteTask(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to complete task: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Task completed successfully",
	})
}

// GetLibrary returns all subscribed shows
// GET /api/library
func (h *Handler) GetLibrary(c *gin.Context) {
	shows, err := h.subManager.GetAllSubscriptions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get library: " + err.Error(),
		})
		return
	}

	// Ensure non-nil slice for JSON serialization
	if shows == nil {
		shows = []models.TVShow{}
	}

	c.JSON(http.StatusOK, gin.H{
		"shows": shows,
	})
}

// SendReport sends the daily report via Telegram
// POST /api/report
func (h *Handler) SendReport(c *gin.Context) {
	if h.notifier == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Telegram notifier not configured",
		})
		return
	}

	// Get all pending update tasks for the report
	data, err := h.taskBoard.GetDashboardData()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get tasks for report: " + err.Error(),
		})
		return
	}

	if err := h.notifier.SendDailyReport(data.UpdateTasks); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to send report: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Report sent successfully",
	})
}
