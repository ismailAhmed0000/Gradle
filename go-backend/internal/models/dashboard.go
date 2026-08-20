package models

type DailyActivity struct {
	Date         string `json:"date"`
	Weekday      string `json:"weekday"`
	PagesScanned int    `json:"pages_scanned"`
}

type DashboardSummary struct {
	TodayPagesScanned    int             `json:"today_pages_scanned"`
	WeeklyActivity       []DailyActivity `json:"weekly_activity"`
	CompositedPercentage float64         `json:"composited_percentage"`
	SubmissionsThisWeek  int             `json:"submissions_this_week"`
	PagesScannedThisWeek int             `json:"pages_scanned_this_week"`
}
