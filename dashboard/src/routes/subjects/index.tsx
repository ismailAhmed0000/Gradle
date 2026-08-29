import { createFileRoute } from '@tanstack/react-router'
import { useState } from 'react'
import { useCreateSubject, useSubjects } from '../../features/subjects/api'
import { requireAuth } from '../../lib/guards'

export const Route = createFileRoute('/subjects/')({
  beforeLoad: requireAuth,
  component: SubjectsPage,
})

function NewSubjectForm({ onDone }: { onDone: () => void }) {
  const createSubject = useCreateSubject()
  const [name, setName] = useState('')

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    createSubject.mutate({ name }, { onSuccess: onDone })
  }

  return (
    <form
      onSubmit={handleSubmit}
      className="flex items-start gap-3 rounded-lg border border-slate-200 bg-white p-4"
    >
      <div className="flex-1">
        <label className="mb-1 block text-sm font-medium text-slate-700">Name</label>
        <input
          type="text"
          required
          autoFocus
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="e.g. Mathematics"
          className="w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none"
        />
        {createSubject.isError && (
          <p className="mt-1 text-sm text-red-600">{createSubject.error.message}</p>
        )}
      </div>
      <div className="mt-6 flex gap-2">
        <button
          type="submit"
          disabled={createSubject.isPending}
          className="rounded-md bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-500 disabled:opacity-50"
        >
          {createSubject.isPending ? 'Creating…' : 'Create'}
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

function SubjectsPage() {
  const { data, isLoading, isError, error } = useSubjects()
  const [showForm, setShowForm] = useState(false)

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Subjects</h1>
        {!showForm && (
          <button
            type="button"
            onClick={() => setShowForm(true)}
            className="rounded-md bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-500"
          >
            New subject
          </button>
        )}
      </div>

      {showForm && <NewSubjectForm onDone={() => setShowForm(false)} />}

      {isLoading && <p className="text-slate-500">Loading…</p>}
      {isError && <p className="text-red-600">{error.message}</p>}

      {data && data.length === 0 && <p className="text-slate-500">No subjects yet.</p>}

      {data && data.length > 0 && (
        <ul className="divide-y divide-slate-100 rounded-lg border border-slate-200 bg-white">
          {data.map((subject) => (
            <li key={subject.id} className="px-4 py-3 text-sm text-slate-700">
              {subject.name}
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
