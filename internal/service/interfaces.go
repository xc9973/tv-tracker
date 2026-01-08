package service

// ReportSender defines capability to send daily reports.
type ReportSender interface {
	SendDailyReport() error
}
