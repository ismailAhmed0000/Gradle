import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '../../api/client'
import { unwrap } from '../../api/unwrap'

export function useSubjects() {
  return useQuery({
    queryKey: ['subjects'],
    queryFn: async () => unwrap(await api.GET('/api/subjects')),
  })
}

export function useCreateSubject() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (body: { name: string }) => {
      const result = await api.POST('/api/subjects', { body })
      return unwrap(result)
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['subjects'] }),
  })
}
