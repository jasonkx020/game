import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api, authApi } from '@/api/client'

export interface AuthUser {
  user_id: number
  nickname: string
  role: string
}

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('access_token') || '')
  const refreshToken = ref(localStorage.getItem('refresh_token') || '')
  const user = ref<AuthUser | null>(
    localStorage.getItem('user') ? JSON.parse(localStorage.getItem('user')!) : null,
  )

  const isLoggedIn = computed(() => !!token.value)

  function persist() {
    if (token.value) localStorage.setItem('access_token', token.value)
    else localStorage.removeItem('access_token')
    if (refreshToken.value) localStorage.setItem('refresh_token', refreshToken.value)
    else localStorage.removeItem('refresh_token')
    if (user.value) localStorage.setItem('user', JSON.stringify(user.value))
    else localStorage.removeItem('user')
  }

  async function login(phone: string, smsCode: string) {
    const res = await authApi.post<{
      access_token: string
      refresh_token: string
      user_id: number
      nickname: string
      role: string
    }>('/admin/auth/login', { phone, sms_code: smsCode })
    token.value = res.access_token
    refreshToken.value = res.refresh_token
    user.value = { user_id: res.user_id, nickname: res.nickname, role: res.role }
    persist()
  }

  async function refresh() {
    const access = token.value
    const refreshTok = refreshToken.value
    if (!access || !refreshTok) {
      throw new Error('access_token and refresh_token required')
    }

    const maxAttempts = 3
    let lastErr: unknown
    for (let attempt = 0; attempt < maxAttempts; attempt++) {
      try {
        const res = await authApi.post<{
          access_token: string
          refresh_token: string
          user_id: number
          nickname: string
          role: string
        }>('/admin/auth/refresh', {
          access_token: access,
          refresh_token: refreshTok,
        })
        // 成功后才改内存并落盘；失败保留旧票以便宽限内重试
        token.value = res.access_token
        refreshToken.value = res.refresh_token
        user.value = { user_id: res.user_id, nickname: res.nickname, role: res.role }
        persist()
        return
      } catch (err) {
        lastErr = err
        if (!isRetryableRefreshError(err) || attempt === maxAttempts - 1) {
          throw err
        }
        await new Promise((r) => setTimeout(r, 200 * (attempt + 1)))
      }
    }
    throw lastErr
  }

  async function logout() {
    try {
      if (token.value) {
        await authApi.post('/admin/auth/logout', { access_token: token.value })
      }
    } catch {
      /* ignore */
    }
    token.value = ''
    refreshToken.value = ''
    user.value = null
    persist()
  }

  return { token, refreshToken, user, isLoggedIn, login, refresh, logout, api }
})

function isRetryableRefreshError(err: unknown): boolean {
  if (!err || typeof err !== 'object') return true
  const code = (err as { code?: number }).code
  if (typeof code !== 'number') return true
  if (code === 1001 || code === 401 || code === 403 || code === 400) return false
  return code >= 500 || code === 0
}
