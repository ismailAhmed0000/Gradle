import { useMutation, useQuery } from '@tanstack/react-query'
import { api } from '../../api/client'
import { unwrap } from '../../api/unwrap'
import { setToken, clearToken } from '../../lib/auth'

export function useLogin() {
  return useMutation({
    mutationFn: async (body: { email: string; password: string }) => {
      const result = await api.POST('/api/auth/login', { body })
      return unwrap(result)
    },
    onSuccess: (data) => setToken(data.token),
  })
}

export function useRegister() {
  return useMutation({
    mutationFn: async (body: { email: string; password: string }) => {
      const result = await api.POST('/api/auth/register', { body })
      return unwrap(result)
    },
    onSuccess: (data) => setToken(data.token),
  })
}

export function useMe(enabled: boolean) {
  return useQuery({
    queryKey: ['me'],
    queryFn: async () => unwrap(await api.GET('/api/auth/me')),
    enabled,
    retry: false,
  })
}

export function useIsAdmin() {
  const { data } = useMe(true)
  return data?.role === 'admin'
}

export function logout() {
  clearToken()
}
