const STATUS_STYLES: Record<string, string> = {
  pending: 'bg-slate-100 text-slate-700',
  processing: 'bg-amber-100 text-amber-700',
  generating: 'bg-amber-100 text-amber-700',
  submitted: 'bg-amber-100 text-amber-700',
  composited: 'bg-emerald-100 text-emerald-700',
  graded: 'bg-emerald-100 text-emerald-700',
  done: 'bg-emerald-100 text-emerald-700',
  expired: 'bg-slate-200 text-slate-600',
  failed: 'bg-red-100 text-red-700',
}

export function Badge({ status }: { status: string }) {
  return (
    <span
      className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${
        STATUS_STYLES[status] ?? 'bg-slate-100 text-slate-700'
      }`}
    >
      {status}
    </span>
  )
}
