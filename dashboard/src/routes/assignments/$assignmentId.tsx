import { createFileRoute, Link } from '@tanstack/react-router'
import { useAssignment } from '../../features/assignments/api'
import { useSubmissionsForAssignment } from '../../features/submissions/api'
import { Badge } from '../../components/Badge'
import { requireAuth } from '../../lib/guards'

export const Route = createFileRoute('/assignments/$assignmentId')({
  beforeLoad: requireAuth,
  component: AssignmentDetailPage,
})

function AssignmentDetailPage() {
  const { assignmentId } = Route.useParams()
  const assignment = useAssignment(assignmentId)
  const submissions = useSubmissionsForAssignment(assignmentId)

  if (assignment.isLoading) return <p className="text-slate-500">Loading…</p>
  if (assignment.isError) return <p className="text-red-600">{assignment.error.message}</p>
  if (!assignment.data) return null

  const a = assignment.data

  return (
    <div className="space-y-8">
      <div>
        <Link to="/assignments" className="text-sm text-indigo-600 hover:underline">
          ← All assignments
        </Link>
        <div className="mt-2 flex items-center gap-3">
          <h1 className="text-2xl font-semibold">{a.title}</h1>
          <Badge status={a.status} />
        </div>
        <p className="mt-1 text-sm text-slate-500">
          {a.subject_name ?? 'No subject'} · Teacher: {a.teacher_email}
          {a.due_date && <> · Due {new Date(a.due_date).toLocaleDateString()}</>}
        </p>
      </div>

      <section>
        <h2 className="mb-3 text-sm font-medium text-slate-700">
          Files ({a.assignment_files.length})
        </h2>
        {a.assignment_files.length === 0 ? (
          <p className="text-sm text-slate-500">No files uploaded.</p>
        ) : (
          <ul className="divide-y divide-slate-100 rounded-lg border border-slate-200 bg-white">
            {a.assignment_files.map((file) => (
              <li key={file.id} className="flex items-center justify-between px-4 py-3 text-sm">
                <span className="text-slate-700">
                  {file.file_path} · {file.page_count} page{file.page_count === 1 ? '' : 's'}
                </span>
                {file.download_url && (
                  <a
                    href={file.download_url}
                    target="_blank"
                    rel="noreferrer"
                    className="text-indigo-600 hover:underline"
                  >
                    Download
                  </a>
                )}
              </li>
            ))}
          </ul>
        )}
      </section>

      <section>
        <h2 className="mb-3 text-sm font-medium text-slate-700">
          Questions ({a.questions.length})
        </h2>
        {a.questions.length === 0 ? (
          <p className="text-sm text-slate-500">No questions defined.</p>
        ) : (
          <div className="flex flex-wrap gap-2">
            {a.questions.map((q) => (
              <span
                key={q.id}
                className="rounded-md border border-slate-200 bg-white px-2.5 py-1 text-xs text-slate-600"
              >
                Q{q.question_number}
                {!q.has_defined_region && ' (no region)'}
              </span>
            ))}
          </div>
        )}
      </section>

      <section>
        <h2 className="mb-3 text-sm font-medium text-slate-700">Submissions</h2>
        {submissions.isLoading && <p className="text-sm text-slate-500">Loading…</p>}
        {submissions.isError && (
          <p className="text-sm text-red-600">{submissions.error.message}</p>
        )}
        {submissions.data && submissions.data.length === 0 && (
          <p className="text-sm text-slate-500">No submissions yet.</p>
        )}
        {submissions.data && submissions.data.length > 0 && (
          <div className="overflow-hidden rounded-lg border border-slate-200 bg-white">
            <table className="w-full text-left text-sm">
              <thead className="border-b border-slate-200 bg-slate-50 text-slate-500">
                <tr>
                  <th className="px-4 py-3 font-medium">Student</th>
                  <th className="px-4 py-3 font-medium">Pages</th>
                  <th className="px-4 py-3 font-medium">Regions done</th>
                  <th className="px-4 py-3 font-medium">Status</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {submissions.data.map((s) => (
                  <tr key={s.id} className="hover:bg-slate-50">
                    <td className="px-4 py-3">
                      <Link
                        to="/submissions/$submissionId"
                        params={{ submissionId: s.id }}
                        className="font-medium text-indigo-600 hover:underline"
                      >
                        {s.student_name}
                      </Link>
                    </td>
                    <td className="px-4 py-3 text-slate-600">{s.page_count}</td>
                    <td className="px-4 py-3 text-slate-600">
                      {s.answer_regions_done} / {s.answer_regions_total}
                    </td>
                    <td className="px-4 py-3">
                      <Badge status={s.status} />
                    </td>
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
