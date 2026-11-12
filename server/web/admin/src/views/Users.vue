<template>
  <div v-loading="loading">
    <div class="toolbar">
      <h2>管理员用户</h2>
    </div>

    <el-table :data="users" stripe>
      <el-table-column prop="user_id" label="ID" width="80" />
      <el-table-column prop="phone_masked" label="手机号" width="140" />
      <el-table-column prop="nickname" label="昵称" />
      <el-table-column prop="role" label="角色" width="140">
        <template #default="{ row }">
          <el-tag :type="roleTagType(row.role)" size="small">{{ roleLabel(row.role) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="status" label="状态" width="100" />
      <el-table-column label="注册时间" width="180">
        <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
      </el-table-column>
      <el-table-column label="最近登录" width="180">
        <template #default="{ row }">{{ row.last_login_at ? formatTime(row.last_login_at) : '-' }}</template>
      </el-table-column>
    </el-table>

    <div class="pager">
      <el-pagination
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :total="total"
        :page-sizes="[10, 20, 50, 100]"
        layout="total, sizes, prev, pager, next"
        @current-change="load"
        @size-change="onPageSizeChange"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '@/api/client'

interface AdminUser {
  user_id: number
  phone_masked: string
  nickname: string
  role: string
  status: string
  created_at: string
  last_login_at?: string
}

const loading = ref(false)
const users = ref<AdminUser[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)

function roleLabel(role: string): string {
  const map: Record<string, string> = {
    platform_admin: '平台管理员',
    club_admin: '俱乐部管理员',
    agent: '代理',
  }
  return map[role] || role
}

function roleTagType(role: string): '' | 'success' | 'warning' | 'danger' | 'info' {
  if (role === 'platform_admin') return 'danger'
  if (role === 'club_admin') return 'warning'
  if (role === 'agent') return 'success'
  return 'info'
}

function formatTime(value: string): string {
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return value
  return d.toLocaleString('zh-CN')
}

async function load() {
  loading.value = true
  try {
    const res = await api.get<{
      users: AdminUser[]
      total: number
      page: number
      page_size: number
    }>(`/admin/users?page=${page.value}&page_size=${pageSize.value}`)
    users.value = res.users
    total.value = res.total
    page.value = res.page
    pageSize.value = res.page_size
  } catch (e: unknown) {
    const err = e as { message?: string; code?: number }
    if (err.code === 403) {
      ElMessage.error('无权限：仅平台管理员可查看用户列表')
    } else {
      ElMessage.error(err.message || '加载失败')
    }
  } finally {
    loading.value = false
  }
}

function onPageSizeChange() {
  page.value = 1
  load()
}

onMounted(load)
</script>

<style scoped>
.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}
.toolbar h2 {
  margin: 0;
}
.pager {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
</style>
