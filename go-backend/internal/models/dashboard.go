package models

type DailyActivity struct {
	Date         string `json:"date"`
	Weekday      string `json:"weekday"`
	PagesScanned int    `json:"pages_scanned"`
}

type DashboardSummary struct {
	TodayPagesScanned    int             `json:"today_pages_scanned"`
	WeeklyActivity       []DailyActivity `json:"weekly_activity"`
	SubmittedThisWeek    int             `json:"submitted_this_week"`
	PendingThisWeek      int             `json:"pending_this_week"`
	SubmissionsThisWeek  int             `json:"submissions_this_week"`
	PagesScannedThisWeek int             `json:"pages_scanned_this_week"`
}
