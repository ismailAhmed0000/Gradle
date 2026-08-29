import { createFileRoute } from '@tanstack/react-router'
import { useState } from 'react'
import { useCreateSubject, useSubjects } from '../../features/subjects/api'
import { SubjectStudentsPanel } from '../../features/subjects/SubjectStudentsPanel'
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

function ChevronIcon({ open }: { open: boolean }) {
  return (
    <svg
      viewBox="0 0 20 20"
      fill="currentColor"
      className={`h-4 w-4 shrink-0 text-slate-400 transition-transform ${open ? 'rotate-180' : ''}`}
    >
      <path
        fillRule="evenodd"
        d="M5.23 7.21a.75.75 0 011.06.02L10 11.168l3.71-3.938a.75.75 0 111.08 1.04l-4.25 4.5a.75.75 0 01-1.08 0l-4.25-4.5a.75.75 0 01.02-1.06z"
        clipRule="evenodd"
      />
    </svg>
  )
}

function SubjectsPage() {
  const { data, isLoading, isError, error } = useSubjects()
  const [showForm, setShowForm] = useState(false)
  const [expandedId, setExpandedId] = useState<string | null>(null)

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
          {data.map((subject) => {
            const isOpen = expandedId === subject.id
            return (
              <li key={subject.id}>
                <button
                  type="button"
                  onClick={() => setExpandedId(isOpen ? null : subject.id)}
                  className="flex w-full items-center justify-between px-4 py-3 text-left text-sm font-medium text-slate-700 hover:bg-slate-50"
                >
                  {subject.name}
                  <ChevronIcon open={isOpen} />
                </button>
                {isOpen && (
                  <div className="border-t border-slate-100 bg-slate-50/50 px-4 py-3">
                    <SubjectStudentsPanel subjectId={subject.id} />
                  </div>
                )}
              </li>
            )
          })}
        </ul>
      )}
    </div>
  )
}
