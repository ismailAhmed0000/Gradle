import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '../../api/client'
import { unwrap } from '../../api/unwrap'

export function useGoogleStatus() {
  return useQuery({
    queryKey: ['integrations', 'google', 'status'],
    queryFn: async () => unwrap(await api.GET('/api/integrations/google/status')),
  })
}

// Redirects the browser to Google's consent screen; Google redirects back to
// this same page (redirectUri) with ?connected=1 once the teacher approves.
export async function connectGoogleClassroom(redirectUri: string) {
  const result = await api.GET('/api/integrations/google/teacher/auth-url', {
    params: { query: { redirect_uri: redirectUri } },
  })
  const { url } = unwrap(result)
  window.location.href = url
}

export function useDisconnectGoogle() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async () => {
      await api.DELETE('/api/integrations/google')
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['integrations', 'google'] }),
  })
}

export function useGoogleCourses(enabled: boolean) {
  return useQuery({
    queryKey: ['integrations', 'google', 'courses'],
    queryFn: async () => unwrap(await api.GET('/api/integrations/google/courses')),
    enabled,
  })
}

export function useGoogleCourseWork(courseId: string) {
  return useQuery({
    queryKey: ['integrations', 'google', 'courses', courseId, 'coursework'],
    queryFn: async () =>
      unwrap(
        await api.GET('/api/integrations/google/courses/{id}/coursework', {
          params: { path: { id: courseId } },
        }),
      ),
    enabled: courseId.length > 0,
  })
}

export function useImportCoursework() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (body: {
      course_id: string
      subject_id: string
      coursework_ids: string[]
    }) => unwrap(await api.POST('/api/integrations/google/import', { body })),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['assignments'] }),
  })
}
