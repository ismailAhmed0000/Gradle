const TOKEN_KEY = 'gradle_dashboard_token'

const listeners = new Set<() => void>()

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token)
  listeners.forEach((listener) => listener())
}

export function clearToken(): void {
  localStorage.removeItem(TOKEN_KEY)
  listeners.forEach((listener) => listener())
}

export function subscribe(listener: () => void): () => void {
  listeners.add(listener)
  return () => listeners.delete(listener)
}
