import { createFileRoute, Link } from '@tanstack/react-router'
import { useSubmission, useSubmissionComposited } from '../../features/submissions/api'
import { Badge } from '../../components/Badge'
import { requireAuth } from '../../lib/guards'

export const Route = createFileRoute('/submissions/$submissionId')({
  beforeLoad: requireAuth,
  component: SubmissionDetailPage,
})

function SubmissionDetailPage() {
  const { submissionId } = Route.useParams()
  const submission = useSubmission(submissionId)
  const composited = useSubmissionComposited(
    submissionId,
    submission.data?.status === 'composited',
  )

  if (submission.isLoading) return <p className="text-slate-500">Loading…</p>
  if (submission.isError) return <p className="text-red-600">{submission.error.message}</p>
  if (!submission.data) return null

  const s = submission.data

  return (
    <div className="space-y-8">
      <div>
        <Link
          to="/assignments/$assignmentId"
          params={{ assignmentId: s.assignment_id }}
          className="text-sm text-indigo-600 hover:underline"
        >
          ← Back to assignment
        </Link>
        <div className="mt-2 flex items-center gap-3">
          <h1 className="text-2xl font-semibold">{s.student_name}</h1>
          <Badge status={s.status} />
        </div>
      </div>

      {s.status === 'composited' && (
        <section className="rounded-lg border border-slate-200 bg-white p-4">
          <h2 className="mb-2 text-sm font-medium text-slate-700">Graded PDF</h2>
          {composited.isLoading && <p className="text-sm text-slate-500">Checking…</p>}
          {composited.data?.download_url && (
            <a
              href={composited.data.download_url}
              target="_blank"
              rel="noreferrer"
              className="text-sm text-indigo-600 hover:underline"
            >
              Download composited PDF
            </a>
          )}
          {composited.data?.error_message && (
            <p className="text-sm text-red-600">{composited.data.error_message}</p>
          )}
        </section>
      )}

      <section>
        <h2 className="mb-3 text-sm font-medium text-slate-700">
          Pages ({s.pages.length})
        </h2>
        {s.pages.length === 0 ? (
          <p className="text-sm text-slate-500">No pages scanned yet.</p>
        ) : (
          <ul className="divide-y divide-slate-100 rounded-lg border border-slate-200 bg-white">
            {s.pages.map((page) => (
              <li key={page.id} className="px-4 py-3 text-sm text-slate-700">
                Page {page.page_number} · {page.raw_image_path}
              </li>
            ))}
          </ul>
        )}
      </section>

      <section>
        <h2 className="mb-3 text-sm font-medium text-slate-700">
          Answer regions ({s.answer_regions.length})
        </h2>
        {s.answer_regions.length === 0 ? (
          <p className="text-sm text-slate-500">No answer regions yet.</p>
        ) : (
          <div className="overflow-hidden rounded-lg border border-slate-200 bg-white">
            <table className="w-full text-left text-sm">
              <thead className="border-b border-slate-200 bg-slate-50 text-slate-500">
                <tr>
                  <th className="px-4 py-3 font-medium">Question</th>
                  <th className="px-4 py-3 font-medium">Status</th>
                  <th className="px-4 py-3 font-medium">Error</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {s.answer_regions.map((region) => (
                  <tr key={region.id}>
                    <td className="px-4 py-3 text-slate-700">Q{region.question_number}</td>
                    <td className="px-4 py-3">
                      <Badge status={region.status} />
                    </td>
                    <td className="px-4 py-3 text-red-600">{region.error_message ?? '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  )
}
