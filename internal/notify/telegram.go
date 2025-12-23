package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"tv-tracker/internal/models"
)

// TelegramNotifier handles Telegram notifications
type TelegramNotifier struct {
	botToken   string
	chatID     string
	httpClient *http.Client
	baseURL    string
}

// NewTelegramNotifier creates a new TelegramNotifier
func NewTelegramNotifier(botToken, chatID string) *TelegramNotifier {
	return &TelegramNotifier{
		botToken: botToken,
		chatID:   chatID,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://api.telegram.org",
	}
}

// telegramMessage represents the request body for sending a message
type telegramMessage struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

// telegramResponse represents the response from Telegram API
type telegramResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
}

// SendMessage sends a message to Telegram
// Requirements: 9.4, 9.5
func (n *TelegramNotifier) SendMessage(text string) error {
	if n.botToken == "" || n.chatID == "" {
		return fmt.Errorf("telegram notifier not configured: missing bot token or chat ID")
	}

	url := fmt.Sprintf("%s/bot%s/sendMessage", n.baseURL, n.botToken)

	msg := telegramMessage{
		ChatID:    n.chatID,
		Text:      text,
		ParseMode: "HTML",
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	resp, err := n.httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		// Log error and continue operation (Requirement 9.5)
		return fmt.Errorf("failed to send telegram message: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var telegramResp telegramResponse
	if err := json.Unmarshal(respBody, &telegramResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !telegramResp.OK {
		return fmt.Errorf("telegram API error: %s", telegramResp.Description)
	}

	return nil
}

// SendDailyReport generates and sends a daily update report
// Requirements: 9.1, 9.2, 9.3
func (n *TelegramNotifier) SendDailyReport(tasks []models.Task) error {
	report := FormatDailyReport(tasks)
	return n.SendMessage(report)
}

// FormatDailyReport formats tasks into a daily report message
// Requirements: 9.1, 9.2, 9.3
// Exported for testing purposes
func FormatDailyReport(tasks []models.Task) string {
	today := time.Now().Format("2006-01-02")
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("📺 <b>今日更新日报</b> (%s)\n\n", today))

	// Filter only UPDATE tasks (daily report is for episode updates)
	var updateTasks []models.Task
	for _, task := range tasks {
		if task.TaskType == models.TaskTypeUpdate {
			updateTasks = append(updateTasks, task)
		}
	}

	// Requirement 9.3: When there are no updates today
	if len(updateTasks) == 0 {
		sb.WriteString("今日暂无剧集更新 🎬")
		return sb.String()
	}

	// Sort tasks by resource time for better readability
	sort.Slice(updateTasks, func(i, j int) bool {
		return updateTasks[i].ResourceTime < updateTasks[j].ResourceTime
	})

	// Requirement 9.2: Format message with show name, episode info, and resource time
	for i, task := range updateTasks {
		// Extract episode info from description (format: "新剧集更新: SxxExx - Episode Name")
		episodeInfo := extractEpisodeInfo(task.Description)

		sb.WriteString(fmt.Sprintf("%d. <b>%s</b>\n", i+1, task.TVShowName))
		sb.WriteString(fmt.Sprintf("   📍 %s\n", episodeInfo))
		sb.WriteString(fmt.Sprintf("   ⏰ 预计资源时间: %s\n", task.ResourceTime))

		if i < len(updateTasks)-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// extractEpisodeInfo extracts episode info from task description
func extractEpisodeInfo(description string) string {
	// Description format: "新剧集更新: SxxExx - Episode Name"
	if strings.HasPrefix(description, "新剧集更新: ") {
		return strings.TrimPrefix(description, "新剧集更新: ")
	}
	return description
}
