import createClient from 'openapi-fetch'
import type { paths } from './schema'
import { clearToken, getToken } from '../lib/auth'

const baseUrl = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080'

export const api = createClient<paths>({ baseUrl })

api.use({
  onRequest({ request }) {
    const token = getToken()
    if (token) {
      request.headers.set('Authorization', `Bearer ${token}`)
    }
    return request
  },
  onResponse({ response }) {
    if (response.status === 401) {
      clearToken()
    }
    return response
  },
})

export class ApiError extends Error {
  constructor(message: string) {
    super(message)
    this.name = 'ApiError'
  }
}
