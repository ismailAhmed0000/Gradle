import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '../../api/client'
import { unwrap } from '../../api/unwrap'

export function useStudents() {
  return useQuery({
    queryKey: ['students'],
    queryFn: async () => unwrap(await api.GET('/api/students')),
  })
}

export function useStudent(id: string) {
  return useQuery({
    queryKey: ['students', id],
    queryFn: async () =>
      unwrap(await api.GET('/api/students/{id}', { params: { path: { id } } })),
    enabled: id.length > 0,
  })
}

export function useCreateStudent() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (body: { name: string; email?: string | null }) => {
      const result = await api.POST('/api/students', { body })
      return unwrap(result)
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['students'] }),
  })
}

export function useEnrollStudent(studentId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (body: { subject_id: string }) => {
      const result = await api.POST('/api/students/{id}/enroll', {
        params: { path: { id: studentId } },
        body,
      })
      return unwrap(result)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['students', studentId] })
      queryClient.invalidateQueries({ queryKey: ['students'] })
    },
  })
}
