import { createFileRoute, Link } from '@tanstack/react-router'
import { useState } from 'react'
import { useEnrollStudent, useStudent } from '../../features/students/api'
import { useSubjects } from '../../features/subjects/api'
import { useIsAdmin } from '../../features/auth/api'
import { Badge } from '../../components/Badge'
import { requireAuth } from '../../lib/guards'

export const Route = createFileRoute('/students/$studentId')({
  beforeLoad: requireAuth,
  component: StudentDetailPage,
})

function EnrollForm({ studentId, enrolledIds }: { studentId: string; enrolledIds: Set<string> }) {
  const subjects = useSubjects()
  const enroll = useEnrollStudent()
  const [subjectId, setSubjectId] = useState('')

  const available = subjects.data?.filter((s) => !enrolledIds.has(s.id)) ?? []

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!subjectId) return
    enroll.mutate({ studentId, subjectId }, { onSuccess: () => setSubjectId('') })
  }

  if (subjects.data && available.length === 0) {
    return <p className="text-sm text-slate-500">Enrolled in every subject you've created.</p>
  }

  return (
    <form onSubmit={handleSubmit} className="flex items-center gap-2">
      <select
        value={subjectId}
        onChange={(e) => setSubjectId(e.target.value)}
        className="rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none"
      >
        <option value="" disabled>
          Select a subject…
        </option>
        {available.map((s) => (
          <option key={s.id} value={s.id}>
            {s.name}
          </option>
        ))}
      </select>
      <button
        type="submit"
        disabled={enroll.isPending || !subjectId}
        className="rounded-md bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-500 disabled:opacity-50"
      >
        {enroll.isPending ? 'Enrolling…' : 'Enroll'}
      </button>
      {enroll.isError && <p className="text-sm text-red-600">{enroll.error.message}</p>}
    </form>
  )
}

function StudentDetailPage() {
  const { studentId } = Route.useParams()
  const student = useStudent(studentId)
  const isAdmin = useIsAdmin()

  if (student.isLoading) return <p className="text-slate-500">Loading…</p>
  if (student.isError) return <p className="text-red-600">{student.error.message}</p>
  if (!student.data) return null

  const s = student.data
  const enrolledIds = new Set(s.subjects.map((sub) => sub.id))

  return (
    <div className="space-y-8">
      <div>
        <Link to="/students" className="text-sm text-indigo-600 hover:underline">
          ← All students
        </Link>
        <h1 className="mt-2 text-2xl font-semibold">{s.name}</h1>
        {s.email && <p className="mt-1 text-sm text-slate-500">{s.email}</p>}
      </div>

      <section>
        <h2 className="mb-3 text-sm font-medium text-slate-700">Enrolled subjects</h2>
        <div className="mb-4 flex flex-wrap gap-2">
          {s.subjects.length === 0 ? (
            <p className="text-sm text-slate-500">Not enrolled in any subject yet.</p>
          ) : (
            s.subjects.map((sub) => (
              <span
                key={sub.id}
                className="rounded-full bg-slate-100 px-3 py-1 text-xs font-medium text-slate-700"
              >
                {sub.name}
              </span>
            ))
          )}
        </div>
        {isAdmin && <EnrollForm studentId={studentId} enrolledIds={enrolledIds} />}
      </section>

      <section>
        <h2 className="mb-3 text-sm font-medium text-slate-700">
          Assignments &amp; answers ({s.submissions.length})
        </h2>
        {s.submissions.length === 0 ? (
          <p className="text-sm text-slate-500">
            No assignments yet — enroll this student in a subject with assignments.
          </p>
        ) : (
          <div className="overflow-hidden rounded-lg border border-slate-200 bg-white">
            <table className="w-full text-left text-sm">
              <thead className="border-b border-slate-200 bg-slate-50 text-slate-500">
                <tr>
                  <th className="px-4 py-3 font-medium">Assignment</th>
                  <th className="px-4 py-3 font-medium">Subject</th>
                  <th className="px-4 py-3 font-medium">Answers done</th>
                  <th className="px-4 py-3 font-medium">Status</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {s.submissions.map((work) => (
                  <tr key={work.assignment_id} className="hover:bg-slate-50">
                    <td className="px-4 py-3">
                      <Link
                        to="/assignments/$assignmentId"
                        params={{ assignmentId: work.assignment_id }}
                        className="font-medium text-indigo-600 hover:underline"
                      >
                        {work.assignment_title}
                      </Link>
                    </td>
                    <td className="px-4 py-3 text-slate-600">{work.subject_name ?? '—'}</td>
                    <td className="px-4 py-3 text-slate-600">
                      {work.answer_regions_done} / {work.answer_regions_total}
                    </td>
                    <td className="px-4 py-3">
                      {work.status ? (
                        <Badge status={work.status} />
                      ) : (
                        <Badge status="not started" />
                      )}
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
