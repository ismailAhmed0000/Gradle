import { createFileRoute, Link } from '@tanstack/react-router'
import { useState } from 'react'
import { useCreateStudent, useEnrollStudent, useStudents } from '../../features/students/api'
import { useSubjects } from '../../features/subjects/api'
import { useIsAdmin } from '../../features/auth/api'
import { SidePanel } from '../../components/SidePanel'
import { requireAuth } from '../../lib/guards'

export const Route = createFileRoute('/students/')({
  beforeLoad: requireAuth,
  component: StudentsPage,
})

function NewStudentPanel({ open, onClose }: { open: boolean; onClose: () => void }) {
  const subjects = useSubjects()
  const createStudent = useCreateStudent()
  const enroll = useEnrollStudent()
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [subjectId, setSubjectId] = useState('')

  function reset() {
    setName('')
    setEmail('')
    setSubjectId('')
  }

  function handleClose() {
    reset()
    onClose()
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    createStudent.mutate(
      { name, email: email || null },
      {
        onSuccess: (student) => {
          enroll.mutate({ studentId: student.id, subjectId }, { onSuccess: handleClose })
        },
      },
    )
  }

  const isPending = createStudent.isPending || enroll.isPending
  const errorMessage = createStudent.error?.message ?? enroll.error?.message

  return (
    <SidePanel open={open} onClose={handleClose} title="New student">
      <form onSubmit={handleSubmit} className="flex h-full flex-col">
        <div className="flex-1 space-y-4">
          <div>
            <label className="mb-1 block text-sm font-medium text-slate-700">Name</label>
            <input
              type="text"
              required
              autoFocus
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none"
            />
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-slate-700">Email</label>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none"
            />
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-slate-700">Subject</label>
            <select
              required
              value={subjectId}
              onChange={(e) => setSubjectId(e.target.value)}
              className="w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none"
            >
              <option value="" disabled>
                Select a subject…
              </option>
              {subjects.data?.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.name}
                </option>
              ))}
            </select>
            {subjects.data?.length === 0 && (
              <p className="mt-1 text-xs text-slate-400">
                No subjects yet —{' '}
                <Link to="/subjects" className="text-indigo-600 hover:underline">
                  create one first
                </Link>
                .
              </p>
            )}
          </div>

          {errorMessage && <p className="text-sm text-red-600">{errorMessage}</p>}
        </div>

        <div className="flex gap-2 border-t border-slate-200 pt-4">
          <button
            type="submit"
            disabled={isPending}
            className="rounded-md bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-500 disabled:opacity-50"
          >
            {isPending ? 'Adding…' : 'Add student'}
          </button>
          <button
            type="button"
            onClick={handleClose}
            className="rounded-md border border-slate-300 px-3 py-2 text-sm text-slate-700 hover:bg-slate-100"
          >
            Cancel
          </button>
        </div>
      </form>
    </SidePanel>
  )
}

function StudentsPage() {
  const { data, isLoading, isError, error } = useStudents()
  const [showPanel, setShowPanel] = useState(false)
  const isAdmin = useIsAdmin()

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Students</h1>
        {isAdmin && (
          <button
            type="button"
            onClick={() => setShowPanel(true)}
            className="rounded-md bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-500"
          >
            New student
          </button>
        )}
      </div>

      {isAdmin && <NewStudentPanel open={showPanel} onClose={() => setShowPanel(false)} />}

      {isLoading && <p className="text-slate-500">Loading…</p>}
      {isError && <p className="text-red-600">{error.message}</p>}

      {data && data.length === 0 && <p className="text-slate-500">No students yet.</p>}

      {data && data.length > 0 && (
        <div className="overflow-hidden rounded-lg border border-slate-200 bg-white">
          <table className="w-full text-left text-sm">
            <thead className="border-b border-slate-200 bg-slate-50 text-slate-500">
              <tr>
                <th className="px-4 py-3 font-medium">Name</th>
                <th className="px-4 py-3 font-medium">Email</th>
                <th className="px-4 py-3 font-medium">Subjects</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {data.map((student) => (
                <tr key={student.id} className="hover:bg-slate-50">
                  <td className="px-4 py-3">
                    <Link
                      to="/students/$studentId"
                      params={{ studentId: student.id }}
                      className="font-medium text-indigo-600 hover:underline"
                    >
                      {student.name}
                    </Link>
                  </td>
                  <td className="px-4 py-3 text-slate-600">{student.email ?? '—'}</td>
                  <td className="px-4 py-3 text-slate-600">
                    {student.subjects.length === 0
                      ? '—'
                      : student.subjects.map((s) => s.name).join(', ')}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
