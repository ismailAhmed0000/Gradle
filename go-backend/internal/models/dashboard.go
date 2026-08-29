package models

type DailyActivity struct {
	Date               string `json:"date"`
	Weekday            string `json:"weekday"`
	PagesScanned       int    `json:"pages_scanned"`
	PendingSubmissions int    `json:"pending_submissions"`
	PendingToGrade     int    `json:"pending_to_grade"`
}

type DashboardSummary struct {
	TodayPagesScanned    int             `json:"today_pages_scanned"`
	WeeklyActivity       []DailyActivity `json:"weekly_activity"`
	SubmittedThisWeek    int             `json:"submitted_this_week"`
	PendingThisWeek      int             `json:"pending_this_week"`
	SubmissionsThisWeek  int             `json:"submissions_this_week"`
	PagesScannedThisWeek int             `json:"pages_scanned_this_week"`
	TotalStudents        int             `json:"total_students"`
	TotalSubjects        int             `json:"total_subjects"`
	PendingSubmissions   int             `json:"pending_submissions"`
	PendingToGrade       int             `json:"pending_to_grade"`
}
