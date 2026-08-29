import { useQuery } from '@tanstack/react-query'
import { api } from '../../api/client'
import { unwrap } from '../../api/unwrap'

export function useSubmissionsForAssignment(assignmentId: string) {
  return useQuery({
    queryKey: ['assignments', assignmentId, 'submissions'],
    queryFn: async () =>
      unwrap(
        await api.GET('/api/assignments/{id}/submissions', {
          params: { path: { id: assignmentId } },
        }),
      ),
    enabled: assignmentId.length > 0,
  })
}

export function useSubmission(id: string) {
  return useQuery({
    queryKey: ['submissions', id],
    queryFn: async () =>
      unwrap(await api.GET('/api/submissions/{id}', { params: { path: { id } } })),
    enabled: id.length > 0,
  })
}

export function useSubmissionComposited(id: string, enabled: boolean) {
  return useQuery({
    queryKey: ['submissions', id, 'composited'],
    queryFn: async () =>
      unwrap(
        await api.GET('/api/submissions/{id}/composited', { params: { path: { id } } }),
      ),
    enabled: enabled && id.length > 0,
    refetchInterval: (query) =>
      query.state.data?.status === 'generating' || query.state.data?.status === 'pending'
        ? 3000
        : false,
  })
}
