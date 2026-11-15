import { signRequest } from './signature'

const appId = import.meta.env.VITE_APP_ID || 'cocos-dev'
const appSecret = import.meta.env.VITE_HMAC_SECRET || 'dev-hmac-secret-change-me'
const apiBase = import.meta.env.VITE_API_BASE || ''
const authBase = import.meta.env.VITE_AUTH_BASE || 'http://localhost:8081'

export interface ApiError {
  code: number
  message: string
  request_id?: string
}

function getToken(): string | undefined {
  return localStorage.getItem('access_token') ?? undefined
}

async function request<T>(
  base: string,
  method: string,
  path: string,
  body?: unknown,
): Promise<T> {
  const bodyStr = body !== undefined ? JSON.stringify(body) : ''
  const signPath = path.startsWith('/v1') ? path : `/v1${path}`
  const headers = await signRequest({
    method,
    path: signPath,
    body: bodyStr || undefined,
    token: getToken(),
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
    if (!res.ok) {
      throw { code: res.status, message: `HTTP ${res.status}` } as ApiError
    }
    throw { code: 500, message: 'invalid JSON response' } as ApiError
  }
  if (!res.ok) {
    const err = (data && typeof data === 'object' ? data : {}) as ApiError
    if (typeof err.code !== 'number') err.code = res.status
    if (!err.message) err.message = `HTTP ${res.status}`
    throw err
  }
  return data as T
}

export const api = {
  get: <T>(path: string) => request<T>(apiBase, 'GET', path),
  post: <T>(path: string, body?: unknown) => request<T>(apiBase, 'POST', path, body),
  patch: <T>(path: string, body?: unknown) => request<T>(apiBase, 'PATCH', path, body),
  delete: <T>(path: string) => request<T>(apiBase, 'DELETE', path),
}

/** Auth endpoints on independent auth-service */
export const authApi = {
  get: <T>(path: string) => request<T>(authBase, 'GET', path),
  post: <T>(path: string, body?: unknown) => request<T>(authBase, 'POST', path, body),
}
