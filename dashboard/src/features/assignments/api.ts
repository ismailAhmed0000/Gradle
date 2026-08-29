import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '../../api/client'
import { unwrap } from '../../api/unwrap'

export function useAssignments() {
  return useQuery({
    queryKey: ['assignments'],
    queryFn: async () => unwrap(await api.GET('/api/assignments')),
  })
}

export function useAssignment(id: string) {
  return useQuery({
    queryKey: ['assignments', id],
    queryFn: async () =>
      unwrap(await api.GET('/api/assignments/{id}', { params: { path: { id } } })),
    enabled: id.length > 0,
  })
}

export function useCreateAssignment() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (body: { title: string; subject_id: string; due_date?: string | null }) => {
      const result = await api.POST('/api/assignments', { body })
      return unwrap(result)
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['assignments'] }),
  })
}

export function useUploadAssignmentFile() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({ assignmentId, file }: { assignmentId: string; file: File }) => {
      const body = new FormData()
      body.append('file', file)
      const result = await api.POST('/api/assignments/{id}/files', {
        params: { path: { id: assignmentId } },
        // openapi-fetch passes FormData through untouched and lets the
        // browser set the multipart Content-Type + boundary itself.
        body: body as unknown as { file: string },
      })
      return unwrap(result)
    },
    onSuccess: (_data, variables) =>
      queryClient.invalidateQueries({ queryKey: ['assignments', variables.assignmentId] }),
  })
}
