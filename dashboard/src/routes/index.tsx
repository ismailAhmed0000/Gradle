import { createFileRoute, Link } from '@tanstack/react-router'
import type { LinkComponentProps } from '@tanstack/react-router'
import { useDashboardSummary } from '../features/dashboard/api'
import { requireAuth } from '../lib/guards'
import type { components } from '../api/schema'

export const Route = createFileRoute('/')({
  beforeLoad: requireAuth,
  component: DashboardPage,
})

function StatCard({
  label,
  value,
  to,
}: {
  label: string
  value: number
  to?: LinkComponentProps['to']
}) {
  const content = (
    <>
      <p className="text-sm text-slate-500">{label}</p>
      <p className="mt-1 text-2xl font-semibold">{value}</p>
    </>
  )

  if (to) {
    return (
      <Link
        to={to}
        className="block rounded-lg border border-slate-200 bg-white p-4 transition-colors hover:border-indigo-300 hover:bg-indigo-50/50"
      >
        {content}
      </Link>
    )
  }

  return <div className="rounded-lg border border-slate-200 bg-white p-4">{content}</div>
}

const SERIES = {
  pagesScanned: { label: 'Pages scanned', color: '#2a78d6' },
  pendingSubmissions: { label: 'Pending submissions', color: '#eb6834' },
  pendingToGrade: { label: 'Pending to grade', color: '#1baf7a' },
} as const

const CHART_WIDTH = 700
const CHART_HEIGHT = 160
const PADDING_LEFT = 28
const PADDING_BOTTOM = 20

function WeeklyActivityChart({
  data,
}: {
  data: components['schemas']['DailyActivity'][]
}) {
  const plotWidth = CHART_WIDTH - PADDING_LEFT
  const plotHeight = CHART_HEIGHT - PADDING_BOTTOM
  const slotWidth = plotWidth / data.length

  const maxValue = Math.max(
    1,
    ...data.map((d) => d.pages_scanned),
    ...data.map((d) => d.pending_submissions),
    ...data.map((d) => d.pending_to_grade),
  )

  const xCenter = (i: number) => PADDING_LEFT + slotWidth * (i + 0.5)
  const yFor = (value: number) => plotHeight - (value / maxValue) * plotHeight

  const linePoints = (key: 'pending_submissions' | 'pending_to_grade') =>
    data.map((d, i) => `${xCenter(i)},${yFor(d[key])}`).join(' ')

  const gridLines = [0, 0.5, 1]

  return (
    <div>
      <svg
        viewBox={`0 0 ${CHART_WIDTH} ${CHART_HEIGHT}`}
        className="w-full"
        style={{ height: CHART_HEIGHT }}
        role="img"
        aria-label="Weekly activity: pages scanned, pending submissions, and pending to grade"
      >
        {gridLines.map((fraction) => (
          <line
            key={fraction}
            x1={PADDING_LEFT}
            x2={CHART_WIDTH}
            y1={plotHeight * (1 - fraction)}
            y2={plotHeight * (1 - fraction)}
            stroke="#e1e0d9"
            strokeWidth={1}
          />
        ))}
        <text x={0} y={4} fontSize={9} fill="#898781">
          {maxValue}
        </text>
        <text x={0} y={plotHeight + 4} fontSize={9} fill="#898781">
          0
        </text>

        {data.map((day, i) => {
          const barWidth = slotWidth * 0.36
          const barHeight = (day.pages_scanned / maxValue) * plotHeight
          const x = xCenter(i) - barWidth / 2
          return (
            <rect
              key={day.date}
              x={x}
              y={plotHeight - barHeight}
              width={barWidth}
              height={Math.max(barHeight, day.pages_scanned > 0 ? 2 : 0)}
              rx={2}
              fill={SERIES.pagesScanned.color}
            >
              <title>{`${day.weekday}: ${day.pages_scanned} pages scanned`}</title>
            </rect>
          )
        })}

        <polyline
          points={linePoints('pending_submissions')}
          fill="none"
          stroke={SERIES.pendingSubmissions.color}
          strokeWidth={2}
          strokeLinecap="round"
          strokeLinejoin="round"
        />
        <polyline
          points={linePoints('pending_to_grade')}
          fill="none"
          stroke={SERIES.pendingToGrade.color}
          strokeWidth={2}
          strokeLinecap="round"
          strokeLinejoin="round"
        />

        {data.map((day, i) => (
          <circle
            key={`pending-submissions-${day.date}`}
            cx={xCenter(i)}
            cy={yFor(day.pending_submissions)}
            r={3.5}
            fill="#fcfcfb"
            stroke={SERIES.pendingSubmissions.color}
            strokeWidth={2}
          >
            <title>{`${day.weekday}: ${day.pending_submissions} pending submissions`}</title>
          </circle>
        ))}
        {data.map((day, i) => (
          <circle
            key={`pending-to-grade-${day.date}`}
            cx={xCenter(i)}
            cy={yFor(day.pending_to_grade)}
            r={3.5}
            fill="#fcfcfb"
            stroke={SERIES.pendingToGrade.color}
            strokeWidth={2}
          >
            <title>{`${day.weekday}: ${day.pending_to_grade} pending to grade`}</title>
          </circle>
        ))}

        {data.map((day, i) => (
          <text
            key={day.date}
            x={xCenter(i)}
            y={CHART_HEIGHT - 4}
            fontSize={10}
            fill="#898781"
            textAnchor="middle"
          >
            {day.weekday}
          </text>
        ))}
      </svg>

      <div className="mt-3 flex flex-wrap gap-x-5 gap-y-1">
        {Object.values(SERIES).map((series) => (
          <div key={series.label} className="flex items-center gap-1.5 text-xs text-slate-600">
            <span
              className="inline-block h-2.5 w-2.5 rounded-sm"
              style={{ backgroundColor: series.color }}
            />
            {series.label}
          </div>
        ))}
      </div>
    </div>
  )
}

function DashboardPage() {
  const { data, isLoading, isError, error } = useDashboardSummary()

  if (isLoading) return <p className="text-slate-500">Loading…</p>
  if (isError) return <p className="text-red-600">{error.message}</p>
  if (!data) return null

  return (
    <div className="space-y-8">
      <h1 className="text-2xl font-semibold">Dashboard</h1>

      <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
        <StatCard label="Total students" value={data.total_students} to="/students" />
        <StatCard label="Total subjects" value={data.total_subjects} to="/subjects" />
        <StatCard label="Pending submissions" value={data.pending_submissions} to="/assignments" />
        <StatCard label="Pending to grade" value={data.pending_to_grade} to="/assignments" />
      </div>

      <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
        <StatCard label="Pages scanned today" value={data.today_pages_scanned} />
        <StatCard label="Submissions this week" value={data.submissions_this_week} />
        <StatCard label="Pending this week" value={data.pending_this_week} />
        <StatCard label="Pages scanned this week" value={data.pages_scanned_this_week} />
      </div>

      <div className="rounded-lg border border-slate-200 bg-white p-6">
        <h2 className="mb-4 text-sm font-medium text-slate-700">Weekly activity</h2>
        <WeeklyActivityChart data={data.weekly_activity} />
      </div>
    </div>
  )
}
