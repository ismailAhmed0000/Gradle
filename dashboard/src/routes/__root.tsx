import { QueryClient } from '@tanstack/react-query'
import { createRootRouteWithContext, Link, Outlet } from '@tanstack/react-router'
import { useMe, logout } from '../features/auth/api'
import { useAuth } from '../lib/useAuth'

export const Route = createRootRouteWithContext<{ queryClient: QueryClient }>()({
  component: RootLayout,
})

const NAV_ITEMS = [
  { to: '/', label: 'Dashboard' },
  { to: '/students', label: 'Students' },
  { to: '/subjects', label: 'Subjects' },
  { to: '/assignments', label: 'Assignments' },
] as const

// The Outlet is kept at the same position in the tree regardless of auth
// state — branching into two differently-shaped trees would remount it on
// login/logout and drop whatever navigation was already in flight.
function RootLayout() {
  const { isAuthenticated } = useAuth()
  const { data: user } = useMe(isAuthenticated)

  return (
    <div className="flex min-h-screen bg-slate-50 text-slate-900">
      {isAuthenticated && (
        <aside className="flex w-56 shrink-0 flex-col border-r border-slate-200 bg-white">
          <div className="px-5 py-5 text-lg font-semibold">Gradle</div>
          <nav className="flex flex-1 flex-col gap-1 px-3">
            {NAV_ITEMS.map((item) => (
              <Link
                key={item.to}
                to={item.to}
                activeOptions={{ exact: item.to === '/' }}
                className="rounded-md px-3 py-2 text-sm font-medium text-slate-600 hover:bg-slate-100"
                activeProps={{ className: 'bg-indigo-50 text-indigo-600' }}
              >
                {item.label}
              </Link>
            ))}
          </nav>
        </aside>
      )}

      <div className="flex flex-1 flex-col">
        {isAuthenticated && (
          <header className="border-b border-slate-200 bg-white">
            <div className="flex items-center justify-end gap-4 px-6 py-4 text-sm text-slate-500">
              {user && <span>{user.email}</span>}
              <button
                type="button"
                onClick={() => logout()}
                className="rounded-md border border-slate-300 px-3 py-1.5 text-slate-700 hover:bg-slate-100"
              >
                Log out
              </button>
            </div>
          </header>
        )}
        <main className="flex-1 px-6 py-8">
          <div className="mx-auto max-w-5xl">
            <Outlet />
          </div>
        </main>
      </div>
    </div>
  )
}
