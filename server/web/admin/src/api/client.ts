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

  const data = await res.json()
  if (!res.ok) {
    throw data as ApiError
  }
  return data as T
}

export const api = {
  get: <T>(path: string) => request<T>(apiBase, 'GET', path),
  post: <T>(path: string, body?: unknown) => request<T>(apiBase, 'POST', path, body),
  delete: <T>(path: string) => request<T>(apiBase, 'DELETE', path),
}

/** Auth endpoints on independent auth-service */
export const authApi = {
  get: <T>(path: string) => request<T>(authBase, 'GET', path),
  post: <T>(path: string, body?: unknown) => request<T>(authBase, 'POST', path, body),
}
