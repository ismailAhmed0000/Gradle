import { Link } from '@tanstack/react-router'
import { useState } from 'react'
import { useSubject } from './api'
import { useEnrollStudent, useStudents } from '../students/api'
import { useIsAdmin } from '../auth/api'

function AddStudentDropdown({
  subjectId,
  enrolledIds,
}: {
  subjectId: string
  enrolledIds: Set<string>
}) {
  const students = useStudents()
  const enroll = useEnrollStudent()
  const [studentId, setStudentId] = useState('')

  const available = students.data?.filter((s) => !enrolledIds.has(s.id)) ?? []

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!studentId) return
    enroll.mutate({ studentId, subjectId }, { onSuccess: () => setStudentId('') })
  }

  if (students.data && available.length === 0) {
    return <p className="text-sm text-slate-500">Every student is already enrolled here.</p>
  }

  return (
    <form onSubmit={handleSubmit} className="flex items-center gap-2">
      <select
        value={studentId}
        onChange={(e) => setStudentId(e.target.value)}
        className="rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none"
      >
        <option value="" disabled>
          Select a student…
        </option>
        {available.map((s) => (
          <option key={s.id} value={s.id}>
            {s.name}
          </option>
        ))}
      </select>
      <button
        type="submit"
        disabled={enroll.isPending || !studentId}
        className="rounded-md bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-500 disabled:opacity-50"
      >
        {enroll.isPending ? 'Adding…' : 'Add to subject'}
      </button>
      {enroll.isError && <p className="text-sm text-red-600">{enroll.error.message}</p>}
    </form>
  )
}

export function SubjectStudentsPanel({ subjectId }: { subjectId: string }) {
  const subject = useSubject(subjectId)
  const isAdmin = useIsAdmin()

  if (subject.isLoading) return <p className="text-sm text-slate-500">Loading…</p>
  if (subject.isError) return <p className="text-sm text-red-600">{subject.error.message}</p>
  if (!subject.data) return null

  const enrolledIds = new Set(subject.data.students.map((s) => s.id))

  return (
    <div className="space-y-3">
      {subject.data.students.length === 0 ? (
        <p className="text-sm text-slate-500">No students enrolled in this subject yet.</p>
      ) : (
        <ul className="divide-y divide-slate-100">
          {subject.data.students.map((student) => (
            <li key={student.id} className="flex items-center justify-between py-2 text-sm">
              <Link
                to="/students/$studentId"
                params={{ studentId: student.id }}
                className="font-medium text-indigo-600 hover:underline"
              >
                {student.name}
              </Link>
              <span className="text-slate-500">{student.email ?? '—'}</span>
            </li>
          ))}
        </ul>
      )}

      {isAdmin && <AddStudentDropdown subjectId={subjectId} enrolledIds={enrolledIds} />}
    </div>
  )
}
