import { apiClient } from './client';

export type DailyActivity = {
  date: string;
  weekday: string;
  pages_scanned: number;
};

export type DashboardSummary = {
  today_pages_scanned: number;
  weekly_activity: DailyActivity[];
  submitted_this_week: number;
  pending_this_week: number;
  submissions_this_week: number;
  pages_scanned_this_week: number;
};

export async function fetchDashboardSummary(): Promise<DashboardSummary> {
  const response = await apiClient.get<DashboardSummary>('/api/dashboard');
  return response.data;
}
