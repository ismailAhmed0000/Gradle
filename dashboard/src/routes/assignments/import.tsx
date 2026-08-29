import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useEffect, useState } from 'react'
import {
  connectGoogleClassroom,
  useGoogleCourses,
  useGoogleCourseWork,
  useGoogleStatus,
  useImportCoursework,
} from '../../features/integrations/googleClassroom'
import { useSubjects } from '../../features/subjects/api'
import { requireAuth } from '../../lib/guards'

export const Route = createFileRoute('/assignments/import')({
  beforeLoad: requireAuth,
  component: ImportFromClassroomPage,
})

function importRedirectUri() {
  return `${window.location.origin}/assignments/import`
}

function ImportFromClassroomPage() {
  const navigate = useNavigate()
  const status = useGoogleStatus()
  const subjects = useSubjects()
  const [courseId, setCourseId] = useState('')
  const [subjectId, setSubjectId] = useState('')
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [connectError, setConnectError] = useState<string | null>(null)

  const courses = useGoogleCourses(status.data?.connected === true)
  const courseWork = useGoogleCourseWork(courseId)
  const importCoursework = useImportCoursework()

  // Google redirects back here with ?connected=1 after the teacher approves
  // access; refresh the connection status once and drop the query param.
  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    if (params.get('connected') === '1') {
      status.refetch()
      window.history.replaceState({}, '', '/assignments/import')
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  function toggle(id: string) {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  function handleConnect() {
    setConnectError(null)
    connectGoogleClassroom(importRedirectUri()).catch((err) =>
      setConnectError(err instanceof Error ? err.message : 'Failed to start connection'),
    )
  }

  function handleImport() {
    if (!courseId || !subjectId || selected.size === 0) return
    importCoursework.mutate(
      { course_id: courseId, subject_id: subjectId, coursework_ids: Array.from(selected) },
      { onSuccess: () => navigate({ to: '/assignments' }) },
    )
  }

  if (status.isLoading) {
    return <p className="text-slate-500">Loading…</p>
  }

  if (!status.data?.connected) {
    return (
      <div className="max-w-lg space-y-4">
        <h1 className="text-2xl font-semibold">Import from Google Classroom</h1>
        <p className="text-sm text-slate-600">
          Connect your Google account to pull in courses and coursework as read-only
          assignments. You'll be asked to grant access to your Classroom courses, roster, and
          Drive files attached to your coursework.
        </p>
        <button
          onClick={handleConnect}
          className="rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-500"
        >
          Connect Google Classroom
        </button>
        {connectError && <p className="text-sm text-red-600">{connectError}</p>}
      </div>
    )
  }

  return (
    <div className="max-w-2xl space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">Import from Google Classroom</h1>
        <p className="mt-1 text-sm text-slate-500">
          Connected as {status.data.google_email}
        </p>
      </div>

      <div>
        <label className="mb-1 block text-sm font-medium text-slate-700">Course</label>
        <select
          value={courseId}
          onChange={(e) => {
            setCourseId(e.target.value)
            setSelected(new Set())
          }}
          className="w-full rounded-md border border-slate-300 px-3 py-2 text-sm"
        >
          <option value="">Select a course…</option>
          {courses.data?.map((course) => (
            <option key={course.ID} value={course.ID}>
              {course.Name}
            </option>
          ))}
        </select>
        {courses.isError && <p className="mt-1 text-sm text-red-600">{courses.error.message}</p>}
      </div>

      {courseId && (
        <div>
          <label className="mb-1 block text-sm font-medium text-slate-700">
            Import into subject
          </label>
          <select
            value={subjectId}
            onChange={(e) => setSubjectId(e.target.value)}
            className="w-full rounded-md border border-slate-300 px-3 py-2 text-sm"
          >
            <option value="">Select a subject…</option>
            {subjects.data?.map((subject) => (
              <option key={subject.id} value={subject.id}>
                {subject.name}
              </option>
            ))}
          </select>
        </div>
      )}

      {courseId && (
        <div>
          <h2 className="mb-2 text-sm font-medium text-slate-700">Coursework</h2>
          {courseWork.isLoading && <p className="text-sm text-slate-500">Loading…</p>}
          {courseWork.isError && (
            <p className="text-sm text-red-600">{courseWork.error.message}</p>
          )}
          {courseWork.data && courseWork.data.length === 0 && (
            <p className="text-sm text-slate-500">No coursework found in this course.</p>
          )}
          {courseWork.data && courseWork.data.length > 0 && (
            <ul className="divide-y divide-slate-100 rounded-lg border border-slate-200 bg-white">
              {courseWork.data.map((cw) => (
                <li key={cw.ID} className="flex items-center gap-3 px-4 py-3 text-sm">
                  <input
                    type="checkbox"
                    checked={selected.has(cw.ID)}
                    disabled={cw.already_imported}
                    onChange={() => toggle(cw.ID)}
                  />
                  <span className={cw.already_imported ? 'text-slate-400' : 'text-slate-700'}>
                    {cw.Title}
                    {cw.already_imported && ' (already imported)'}
                  </span>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}

      <button
        onClick={handleImport}
        disabled={!subjectId || selected.size === 0 || importCoursework.isPending}
        className="rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-500 disabled:opacity-50"
      >
        {importCoursework.isPending ? 'Importing…' : `Import ${selected.size || ''} assignment(s)`}
      </button>
      {importCoursework.isError && (
        <p className="text-sm text-red-600">{importCoursework.error.message}</p>
      )}
    </div>
  )
}
