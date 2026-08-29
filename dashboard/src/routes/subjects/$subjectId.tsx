import { createFileRoute, Link } from '@tanstack/react-router'
import { useSubject } from '../../features/subjects/api'
import { SubjectStudentsPanel } from '../../features/subjects/SubjectStudentsPanel'
import { requireAuth } from '../../lib/guards'

export const Route = createFileRoute('/subjects/$subjectId')({
  beforeLoad: requireAuth,
  component: SubjectDetailPage,
})

function SubjectDetailPage() {
  const { subjectId } = Route.useParams()
  const subject = useSubject(subjectId)

  if (subject.isLoading) return <p className="text-slate-500">Loading…</p>
  if (subject.isError) return <p className="text-red-600">{subject.error.message}</p>
  if (!subject.data) return null

  return (
    <div className="space-y-8">
      <div>
        <Link to="/subjects" className="text-sm text-indigo-600 hover:underline">
          ← All subjects
        </Link>
        <h1 className="mt-2 text-2xl font-semibold">{subject.data.name}</h1>
      </div>

      <section className="rounded-lg border border-slate-200 bg-white p-4">
        <h2 className="mb-3 text-sm font-medium text-slate-700">
          Students ({subject.data.students.length})
        </h2>
        <SubjectStudentsPanel subjectId={subjectId} />
      </section>
    </div>
  )
}
