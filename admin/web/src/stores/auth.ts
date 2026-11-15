import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api, authApi, refreshSession } from '@/api/client'
import { bindSessionListener, clearSession, readSession, writeSession } from '@/api/session'

export interface AuthUser {
  user_id: number
  nickname: string
  role: string
}

export const useAuthStore = defineStore('auth', () => {
  const initial = readSession()
  const token = ref(initial?.accessToken || '')
  const refreshToken = ref(initial?.refreshToken || '')
  const user = ref<AuthUser | null>(initial?.user ?? null)

  const isLoggedIn = computed(() => !!token.value)

  // Keep Pinia in sync when the base HTTP layer refreshes/clears tokens.
  bindSessionListener((session) => {
    if (!session) {
      token.value = ''
      refreshToken.value = ''
      user.value = null
      return
    }
    token.value = session.accessToken
    refreshToken.value = session.refreshToken
    user.value = session.user
  })

  async function login(phone: string, smsCode: string) {
    const res = await authApi.post<{
      access_token: string
      refresh_token: string
      user_id: number
      nickname: string
      role: string
    }>('/admin/auth/login', { phone, sms_code: smsCode })
    writeSession({
      accessToken: res.access_token,
      refreshToken: res.refresh_token,
      user: {
        user_id: res.user_id,
        nickname: res.nickname,
        role: res.role,
      },
    })
  }

  async function refresh() {
    await refreshSession()
  }

  async function logout() {
    try {
      if (token.value) {
        await authApi.post('/admin/auth/logout', { access_token: token.value })
      }
    } catch {
      /* ignore */
    }
    clearSession()
  }

  return { token, refreshToken, user, isLoggedIn, login, refresh, logout, api }
})
