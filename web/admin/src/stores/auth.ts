import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api } from '@/api/client'

export interface AuthUser {
  user_id: number
  nickname: string
  role: string
}

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('access_token') || '')
  const user = ref<AuthUser | null>(
    localStorage.getItem('user') ? JSON.parse(localStorage.getItem('user')!) : null,
  )

  const isLoggedIn = computed(() => !!token.value)

  async function login(phone: string, smsCode: string) {
    const res = await api.post<{
      access_token: string
      user_id: number
      nickname: string
      role: string
    }>('/admin/auth/login', { phone, sms_code: smsCode })
    token.value = res.access_token
    user.value = { user_id: res.user_id, nickname: res.nickname, role: res.role }
    localStorage.setItem('access_token', res.access_token)
    localStorage.setItem('user', JSON.stringify(user.value))
  }

  function logout() {
    token.value = ''
    user.value = null
    localStorage.removeItem('access_token')
    localStorage.removeItem('user')
  }

  return { token, user, isLoggedIn, login, logout }
})
