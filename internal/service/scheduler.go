package service

import (
	"fmt"
	"log"
	"time"
)

// ReportSender interface for sending daily reports
type ReportSender interface {
	SendDailyReport() error
}

// Scheduler handles scheduled tasks
// Requirements: 9.1, 11.1
type Scheduler struct {
	reportSender ReportSender
	backupSvc    *BackupService
	reportTime   string // Format: "HH:MM"
	stopChan     chan struct{}
}

// NewScheduler creates a new Scheduler
func NewScheduler(reportSender ReportSender, backupSvc *BackupService, reportTime string) *Scheduler {
	return &Scheduler{
		reportSender: reportSender,
		backupSvc:    backupSvc,
		reportTime:   reportTime,
		stopChan:     make(chan struct{}),
	}
}

// Start starts all scheduled tasks
func (s *Scheduler) Start() {
	go s.runDailyReportScheduler()
	go s.runWeeklyBackupScheduler()
	log.Printf("Scheduler started - Daily report at %s, Weekly backup on Sundays at 03:00", s.reportTime)
}

// Stop stops all scheduled tasks
func (s *Scheduler) Stop() {
	close(s.stopChan)
}

// runDailyReportScheduler runs the daily report scheduler
// Requirements: 9.1
func (s *Scheduler) runDailyReportScheduler() {
	for {
		// Calculate time until next report
		nextRun := s.calculateNextReportTime()
		duration := time.Until(nextRun)

		log.Printf("Next daily report scheduled at %s (in %v)", nextRun.Format("2006-01-02 15:04:05"), duration.Round(time.Minute))

		select {
		case <-time.After(duration):
			log.Println("Sending daily report...")
			if err := s.reportSender.SendDailyReport(); err != nil {
				log.Printf("Failed to send daily report: %v", err)
			} else {
				log.Println("Daily report sent successfully")
			}
		case <-s.stopChan:
			return
		}
	}
}

// runWeeklyBackupScheduler runs the weekly backup scheduler
// Requirements: 11.1
func (s *Scheduler) runWeeklyBackupScheduler() {
	for {
		// Calculate time until next Sunday at 03:00
		nextRun := s.calculateNextBackupTime()
		duration := time.Until(nextRun)

		log.Printf("Next backup scheduled at %s (in %v)", nextRun.Format("2006-01-02 15:04:05"), duration.Round(time.Hour))

		select {
		case <-time.After(duration):
			log.Println("Running weekly backup...")
			backupPath, err := s.backupSvc.Backup()
			if err != nil {
				log.Printf("Failed to create backup: %v", err)
			} else {
				log.Printf("Backup created successfully: %s", backupPath)
			}
		case <-s.stopChan:
			return
		}
	}
}


// calculateNextReportTime calculates the next time to send the daily report
func (s *Scheduler) calculateNextReportTime() time.Time {
	now := time.Now()

	// Parse report time
	hour, minute := 8, 0 // Default to 08:00
	if s.reportTime != "" {
		fmt.Sscanf(s.reportTime, "%d:%d", &hour, &minute)
	}

	// Create today's report time
	reportTime := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())

	// If we've already passed today's report time, schedule for tomorrow
	if now.After(reportTime) {
		reportTime = reportTime.Add(24 * time.Hour)
	}

	return reportTime
}

// calculateNextBackupTime calculates the next Sunday at 03:00
func (s *Scheduler) calculateNextBackupTime() time.Time {
	now := time.Now()

	// Find next Sunday
	daysUntilSunday := (7 - int(now.Weekday())) % 7
	if daysUntilSunday == 0 {
		// Today is Sunday, check if we've passed 03:00
		backupTime := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, now.Location())
		if now.After(backupTime) {
			// Already passed, schedule for next Sunday
			daysUntilSunday = 7
		}
	}

	nextSunday := now.AddDate(0, 0, daysUntilSunday)
	return time.Date(nextSunday.Year(), nextSunday.Month(), nextSunday.Day(), 3, 0, 0, 0, now.Location())
}
