import { createFileRoute, Link } from '@tanstack/react-router'
import { useState } from 'react'
import { useCreateStudent, useStudents } from '../../features/students/api'
import { requireAuth } from '../../lib/guards'

export const Route = createFileRoute('/students/')({
  beforeLoad: requireAuth,
  component: StudentsPage,
})

function NewStudentForm({ onDone }: { onDone: () => void }) {
  const createStudent = useCreateStudent()
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    createStudent.mutate({ name, email: email || null }, { onSuccess: onDone })
  }

  return (
    <form
      onSubmit={handleSubmit}
      className="grid gap-4 rounded-lg border border-slate-200 bg-white p-4 sm:grid-cols-3"
    >
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
        <label className="mb-1 block text-sm font-medium text-slate-700">
          Email (optional)
        </label>
        <input
          type="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          className="w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none"
        />
      </div>
      <div className="flex items-end gap-2">
        <button
          type="submit"
          disabled={createStudent.isPending}
          className="rounded-md bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-500 disabled:opacity-50"
        >
          {createStudent.isPending ? 'Adding…' : 'Add student'}
        </button>
        <button
          type="button"
          onClick={onDone}
          className="rounded-md border border-slate-300 px-3 py-2 text-sm text-slate-700 hover:bg-slate-100"
        >
          Cancel
        </button>
      </div>
      {createStudent.isError && (
        <p className="text-sm text-red-600 sm:col-span-3">{createStudent.error.message}</p>
      )}
    </form>
  )
}

function StudentsPage() {
  const { data, isLoading, isError, error } = useStudents()
  const [showForm, setShowForm] = useState(false)

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Students</h1>
        {!showForm && (
          <button
            type="button"
            onClick={() => setShowForm(true)}
            className="rounded-md bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-500"
          >
            New student
          </button>
        )}
      </div>

      {showForm && <NewStudentForm onDone={() => setShowForm(false)} />}

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
