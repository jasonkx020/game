/** Token persistence for the HTTP layer (no Pinia dependency). */

export interface SessionUser {
  user_id: number
  nickname: string
  role: string
}

export interface SessionTokens {
  accessToken: string
  refreshToken: string
  user: SessionUser | null
}

type SessionListener = (session: SessionTokens | null) => void

let listener: SessionListener | null = null

/** Auth store (or tests) registers to mirror localStorage into reactive state. */
export function bindSessionListener(fn: SessionListener | null): void {
  listener = fn
}

export function readSession(): SessionTokens | null {
  const accessToken = localStorage.getItem('access_token') || ''
  const refreshToken = localStorage.getItem('refresh_token') || ''
  if (!accessToken && !refreshToken) return null
  let user: SessionUser | null = null
  const raw = localStorage.getItem('user')
  if (raw) {
    try {
      user = JSON.parse(raw) as SessionUser
    } catch {
      user = null
    }
  }
  return { accessToken, refreshToken, user }
}

export function writeSession(session: SessionTokens): void {
  localStorage.setItem('access_token', session.accessToken)
  localStorage.setItem('refresh_token', session.refreshToken)
  if (session.user) localStorage.setItem('user', JSON.stringify(session.user))
  else localStorage.removeItem('user')
  listener?.(session)
}

export function clearSession(): void {
  localStorage.removeItem('access_token')
  localStorage.removeItem('refresh_token')
  localStorage.removeItem('user')
  listener?.(null)
}

export function accessToken(): string | undefined {
  return localStorage.getItem('access_token') || undefined
}

export function refreshToken(): string | undefined {
  return localStorage.getItem('refresh_token') || undefined
}
