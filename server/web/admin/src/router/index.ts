import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', component: () => import('@/views/Login.vue') },
    {
      path: '/',
      component: () => import('@/layouts/MainLayout.vue'),
      meta: { requiresAuth: true },
      children: [
        { path: '', redirect: '/dashboard' },
        { path: 'dashboard', component: () => import('@/views/Dashboard.vue') },
        { path: 'users', component: () => import('@/views/Users.vue') },
        { path: 'players', component: () => import('@/views/Players.vue') },
        { path: 'clubs', component: () => import('@/views/Clubs.vue') },
        { path: 'clubs/:id', component: () => import('@/views/ClubDetail.vue') },
        { path: 'recharge', component: () => import('@/views/Recharge.vue') },
        { path: 'games', component: () => import('@/views/GameConfig.vue') },
        { path: 'companion', component: () => import('@/views/CompanionPersonas.vue') },
      ],
    },
  ],
})

router.beforeEach((to) => {
  const auth = useAuthStore()
  if (to.meta.requiresAuth && !auth.isLoggedIn) {
    return '/login'
  }
  if (to.path === '/login' && auth.isLoggedIn) {
    return '/dashboard'
  }
})

export default router
