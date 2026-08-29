import { createFileRoute } from '@tanstack/react-router'
import { useDashboardSummary } from '../features/dashboard/api'
import { requireAuth } from '../lib/guards'

export const Route = createFileRoute('/')({
  beforeLoad: requireAuth,
  component: DashboardPage,
})

function StatCard({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-lg border border-slate-200 bg-white p-4">
      <p className="text-sm text-slate-500">{label}</p>
      <p className="mt-1 text-2xl font-semibold">{value}</p>
    </div>
  )
}

function DashboardPage() {
  const { data, isLoading, isError, error } = useDashboardSummary()

  if (isLoading) return <p className="text-slate-500">Loading…</p>
  if (isError) return <p className="text-red-600">{error.message}</p>
  if (!data) return null

  const maxPages = Math.max(1, ...data.weekly_activity.map((d) => d.pages_scanned))

  return (
    <div className="space-y-8">
      <h1 className="text-2xl font-semibold">Dashboard</h1>

      <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
        <StatCard label="Pages scanned today" value={data.today_pages_scanned} />
        <StatCard label="Submissions this week" value={data.submissions_this_week} />
        <StatCard label="Pending this week" value={data.pending_this_week} />
        <StatCard label="Pages scanned this week" value={data.pages_scanned_this_week} />
      </div>

      <div className="rounded-lg border border-slate-200 bg-white p-6">
        <h2 className="mb-4 text-sm font-medium text-slate-700">Weekly activity</h2>
        <div className="flex items-end gap-4" style={{ height: 160 }}>
          {data.weekly_activity.map((day) => (
            <div key={day.date} className="flex flex-1 flex-col items-center gap-2">
              <div className="flex flex-1 w-full items-end">
                <div
                  className="w-full rounded-t bg-indigo-500"
                  style={{
                    height: `${(day.pages_scanned / maxPages) * 100}%`,
                    minHeight: day.pages_scanned > 0 ? 4 : 0,
                  }}
                  title={`${day.pages_scanned} pages`}
                />
              </div>
              <span className="text-xs text-slate-400">{day.weekday}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
