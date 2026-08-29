import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '../../api/client'
import { unwrap } from '../../api/unwrap'

export function useGradeSubmission(submissionId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (body: { grade?: string | null; feedback?: string | null }) => {
      const result = await api.PATCH('/api/submissions/{id}/grade', {
        params: { path: { id: submissionId } },
        body,
      })
      return unwrap(result)
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['submissions', submissionId] }),
  })
}
