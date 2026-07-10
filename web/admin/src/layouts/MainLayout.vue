<template>
  <el-container class="layout">
    <el-aside width="220px" class="aside">
      <div class="logo">运营后台</div>
      <el-menu :default-active="route.path" router>
        <el-menu-item index="/dashboard">仪表盘</el-menu-item>
        <el-menu-item index="/clubs">俱乐部</el-menu-item>
        <el-menu-item index="/recharge">Mock 充值</el-menu-item>
        <el-menu-item index="/games">游戏配置</el-menu-item>
        <el-menu-item index="/companion">伴侣人设</el-menu-item>
      </el-menu>
    </el-aside>
    <el-container>
      <el-header class="header">
        <span>{{ auth.user?.nickname }} ({{ auth.user?.role }})</span>
        <el-button link type="danger" @click="onLogout">退出</el-button>
      </el-header>
      <el-main>
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

function onLogout() {
  auth.logout()
  router.push('/login')
}
</script>

<style scoped>
.layout {
  min-height: 100vh;
}
.aside {
  background: #001529;
  color: #fff;
}
.logo {
  padding: 20px;
  font-size: 18px;
  font-weight: bold;
  color: #fff;
}
.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: #fff;
  border-bottom: 1px solid #eee;
}
</style>
