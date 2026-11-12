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
    if (!token.value || !refreshToken.value) {
      throw new Error('access_token and refresh_token required')
    }
    const res = await authApi.post<{
      access_token: string
      refresh_token: string
      user_id: number
      nickname: string
      role: string
    }>('/admin/auth/refresh', {
      access_token: token.value,
      refresh_token: refreshToken.value,
    })
    token.value = res.access_token
    refreshToken.value = res.refresh_token
    user.value = { user_id: res.user_id, nickname: res.nickname, role: res.role }
    persist()
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
