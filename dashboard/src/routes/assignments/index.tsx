import { createFileRoute, Link } from '@tanstack/react-router'
import { useState } from 'react'
import { useAssignments, useCreateAssignment } from '../../features/assignments/api'
import { useSubjects } from '../../features/subjects/api'
import { Badge } from '../../components/Badge'
import { requireAuth } from '../../lib/guards'

export const Route = createFileRoute('/assignments/')({
  beforeLoad: requireAuth,
  component: AssignmentsPage,
})

function NewAssignmentForm({ onDone }: { onDone: () => void }) {
  const subjects = useSubjects()
  const createAssignment = useCreateAssignment()
  const [title, setTitle] = useState('')
  const [subjectId, setSubjectId] = useState('')
  const [dueDate, setDueDate] = useState('')

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    createAssignment.mutate(
      {
        title,
        subject_id: subjectId,
        due_date: dueDate ? new Date(dueDate).toISOString() : null,
      },
      { onSuccess: onDone },
    )
  }

  return (
    <form
      onSubmit={handleSubmit}
      className="space-y-4 rounded-lg border border-slate-200 bg-white p-4"
    >
      <div className="grid gap-4 sm:grid-cols-3">
        <div className="sm:col-span-1">
          <label className="mb-1 block text-sm font-medium text-slate-700">Title</label>
          <input
            type="text"
            required
            value={title}
            onChange={(e) => setTitle(e.target.value)}
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
        <div>
          <label className="mb-1 block text-sm font-medium text-slate-700">
            Due date (optional)
          </label>
          <input
            type="date"
            value={dueDate}
            onChange={(e) => setDueDate(e.target.value)}
            className="w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none"
          />
        </div>
      </div>
      {createAssignment.isError && (
        <p className="text-sm text-red-600">{createAssignment.error.message}</p>
      )}
      <div className="flex gap-2">
        <button
          type="submit"
          disabled={createAssignment.isPending}
          className="rounded-md bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-500 disabled:opacity-50"
        >
          {createAssignment.isPending ? 'Creating…' : 'Create assignment'}
        </button>
        <button
          type="button"
          onClick={onDone}
          className="rounded-md border border-slate-300 px-3 py-2 text-sm text-slate-700 hover:bg-slate-100"
        >
          Cancel
        </button>
      </div>
    </form>
  )
}

function AssignmentsPage() {
  const { data, isLoading, isError, error } = useAssignments()
  const [showForm, setShowForm] = useState(false)

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Assignments</h1>
        {!showForm && (
          <button
            type="button"
            onClick={() => setShowForm(true)}
            className="rounded-md bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-500"
          >
            New assignment
          </button>
        )}
      </div>

      {showForm && <NewAssignmentForm onDone={() => setShowForm(false)} />}

      {isLoading && <p className="text-slate-500">Loading…</p>}
      {isError && <p className="text-red-600">{error.message}</p>}

      {data && data.length === 0 && (
        <p className="text-slate-500">No assignments yet.</p>
      )}

      {data && data.length > 0 && (
        <div className="overflow-hidden rounded-lg border border-slate-200 bg-white">
          <table className="w-full text-left text-sm">
            <thead className="border-b border-slate-200 bg-slate-50 text-slate-500">
              <tr>
                <th className="px-4 py-3 font-medium">Title</th>
                <th className="px-4 py-3 font-medium">Subject</th>
                <th className="px-4 py-3 font-medium">Due date</th>
                <th className="px-4 py-3 font-medium">Status</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {data.map((assignment) => (
                <tr key={assignment.id} className="hover:bg-slate-50">
                  <td className="px-4 py-3">
                    <Link
                      to="/assignments/$assignmentId"
                      params={{ assignmentId: assignment.id }}
                      className="font-medium text-indigo-600 hover:underline"
                    >
                      {assignment.title}
                    </Link>
                  </td>
                  <td className="px-4 py-3 text-slate-600">
                    {assignment.subject_name ?? '—'}
                  </td>
                  <td className="px-4 py-3 text-slate-600">
                    {assignment.due_date
                      ? new Date(assignment.due_date).toLocaleDateString()
                      : '—'}
                  </td>
                  <td className="px-4 py-3">
                    <Badge status={assignment.status} />
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
