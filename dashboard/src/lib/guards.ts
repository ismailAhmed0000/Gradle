import { redirect } from '@tanstack/react-router'
import { getToken } from './auth'

export function requireAuth() {
  if (getToken() === null) {
    throw redirect({ to: '/login' })
  }
}

export function redirectIfAuthed() {
  if (getToken() !== null) {
    throw redirect({ to: '/' })
  }
}
