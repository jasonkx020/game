import { signRequest } from './signature'
import {
  accessToken,
  clearSession,
  refreshToken as storedRefreshToken,
  writeSession,
} from './session'

const appId = import.meta.env.VITE_APP_ID || 'cocos-dev'
const appSecret = import.meta.env.VITE_HMAC_SECRET || 'dev-hmac-secret-change-me'
// Empty = same origin (nginx gateway :8090, embedded admin :8082 via gateway, or Vite proxy).
const apiBase = import.meta.env.VITE_API_BASE || ''
const authBase = import.meta.env.VITE_AUTH_BASE || ''

export interface ApiError {
  code: number
  message: string
  request_id?: string
}

/** Access JWT missing/expired — NOT business 403 (permission ≠ session dead). */
function isAccessAuthFailure(err: ApiError, httpStatus: number): boolean {
  if (httpStatus === 401) return true
  return err.code === 1001 || err.code === 401
}

/**
 * Hard reject from /admin/auth/refresh → clear session & re-login.
 * Do not use for ordinary business API errors (including 403).
 */
function isRefreshHardReject(err: unknown): boolean {
  if (!err || typeof err !== 'object') return false
  const code = (err as ApiError).code
  return code === 1001 || code === 401 || code === 403 || code === 400
}

function isAuthBootstrapPath(path: string): boolean {
  return (
    path.includes('/admin/auth/refresh') ||
    path.includes('/admin/auth/login') ||
    path.includes('/admin/auth/logout')
  )
}

/**
 * Lowest-level signed HTTP call: HMAC + optional Bearer, no refresh.
 * All higher helpers go through `request`, which owns refresh.
 */
async function signedFetch<T>(
  base: string,
  method: string,
  path: string,
  body?: unknown,
): Promise<{ ok: true; data: T } | { ok: false; status: number; err: ApiError }> {
  const bodyStr = body !== undefined ? JSON.stringify(body) : ''
  const signPath = path.startsWith('/v1') ? path : `/v1${path}`
  const headers = await signRequest({
    method,
    path: signPath,
    body: bodyStr || undefined,
    token: accessToken(),
    appId,
    appSecret,
  })

  const res = await fetch(`${base}${signPath}`, {
    method,
    headers: {
      ...headers,
      ...(bodyStr ? { 'Content-Type': 'application/json' } : {}),
    },
    body: bodyStr || undefined,
  })

  let data: unknown
  try {
    data = await res.json()
  } catch {
    const err: ApiError = {
      code: res.ok ? 500 : res.status,
      message: res.ok ? 'invalid JSON response' : `HTTP ${res.status}`,
    }
    return { ok: false, status: res.status || 500, err }
  }

  if (!res.ok) {
    const err = (data && typeof data === 'object' ? data : {}) as ApiError
    if (typeof err.code !== 'number') err.code = res.status
    if (!err.message) err.message = `HTTP ${res.status}`
    return { ok: false, status: res.status, err }
  }

  return { ok: true, data: data as T }
}

/** Single-flight refresh shared by concurrent 401s. */
let refreshInFlight: Promise<void> | null = null

async function refreshAccessToken(): Promise<void> {
  if (refreshInFlight) return refreshInFlight

  refreshInFlight = (async () => {
    const access = accessToken()
    const refreshTok = storedRefreshToken()
    if (!access || !refreshTok) {
      throw { code: 1001, message: 'access_token and refresh_token required' } as ApiError
    }

    const maxAttempts = 3
    let lastErr: unknown
    for (let attempt = 0; attempt < maxAttempts; attempt++) {
      const result = await signedFetch<{
        access_token: string
        refresh_token: string
        user_id: number
        nickname: string
        role: string
      }>(authBase, 'POST', '/admin/auth/refresh', {
        access_token: access,
        refresh_token: refreshTok,
      })

      if (result.ok) {
        const res = result.data
        writeSession({
          accessToken: res.access_token,
          refreshToken: res.refresh_token,
          user: {
            user_id: res.user_id,
            nickname: res.nickname,
            role: res.role,
          },
        })
        return
      }

      lastErr = result.err
      if (isRefreshHardReject(result.err)) {
        throw result.err
      }
      if (attempt === maxAttempts - 1) throw result.err
      await new Promise((r) => setTimeout(r, 200 * (attempt + 1)))
    }
    throw lastErr
  })().finally(() => {
    refreshInFlight = null
  })

  return refreshInFlight
}

function redirectLogin(): void {
  if (typeof window !== 'undefined' && !window.location.pathname.startsWith('/login')) {
    window.location.assign('/login')
  }
}

/**
 * Base request used by every API call (admin-api + auth-service).
 * On access expiry (401/1001): refresh once (single-flight) then retry.
 * Business 403 never clears tokens; only refresh hard-reject does.
 */
async function request<T>(
  base: string,
  method: string,
  path: string,
  body?: unknown,
): Promise<T> {
  const signPath = path.startsWith('/v1') ? path : `/v1${path}`
  const first = await signedFetch<T>(base, method, path, body)
  if (first.ok) return first.data

  const canRefresh =
    !isAuthBootstrapPath(signPath) && isAccessAuthFailure(first.err, first.status)

  if (!canRefresh) {
    throw first.err
  }

  try {
    await refreshAccessToken()
  } catch (refreshErr) {
    if (isRefreshHardReject(refreshErr)) {
      clearSession()
      redirectLogin()
      throw refreshErr
    }
    // Network / 5xx on refresh: keep tokens
    throw first.err
  }

  const second = await signedFetch<T>(base, method, path, body)
  if (second.ok) return second.data

  if (isAccessAuthFailure(second.err, second.status)) {
    clearSession()
    redirectLogin()
  }
  throw second.err
}

export const api = {
  get: <T>(path: string) => request<T>(apiBase, 'GET', path),
  post: <T>(path: string, body?: unknown) => request<T>(apiBase, 'POST', path, body),
  patch: <T>(path: string, body?: unknown) => request<T>(apiBase, 'PATCH', path, body),
  delete: <T>(path: string) => request<T>(apiBase, 'DELETE', path),
}

/** Auth endpoints on independent auth-service — same base request (refresh included). */
export const authApi = {
  get: <T>(path: string) => request<T>(authBase, 'GET', path),
  post: <T>(path: string, body?: unknown) => request<T>(authBase, 'POST', path, body),
}

/** Manual refresh; normally unnecessary — `request` refreshes automatically. */
export async function refreshSession(): Promise<void> {
  await refreshAccessToken()
}
