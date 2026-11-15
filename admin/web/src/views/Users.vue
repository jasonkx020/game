<template>
  <div v-loading="loading">
    <div class="toolbar">
      <h2>管理员用户</h2>
      <div class="toolbar-right">
        <el-input
          v-model="q"
          clearable
          placeholder="搜索 ID / 手机号 / 昵称"
          style="width: 240px"
          @keyup.enter="search"
          @clear="search"
        />
        <el-button type="primary" @click="search">搜索</el-button>
        <el-button type="success" @click="openCreate">新建管理员</el-button>
      </div>
    </div>

    <el-table :data="users" stripe @row-click="openDetail">
      <el-table-column prop="user_id" label="ID" width="80" />
      <el-table-column prop="phone_masked" label="手机号" width="140" />
      <el-table-column prop="nickname" label="昵称" />
      <el-table-column prop="role" label="角色" width="160">
        <template #default="{ row }">
          <el-select
            :model-value="row.role"
            size="small"
            style="width: 140px"
            :disabled="isSelf(row)"
            @click.stop
            @change="(v: string) => changeRole(row, v)"
          >
            <el-option label="平台管理员" value="platform_admin" />
            <el-option label="俱乐部管理员" value="club_admin" />
            <el-option label="代理" value="agent" />
          </el-select>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="110">
        <template #default="{ row }">
          <el-tag :type="row.status === 'active' ? 'success' : 'danger'" size="small">
            {{ row.status === 'active' ? '正常' : '停用' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="注册时间" width="180">
        <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
      </el-table-column>
      <el-table-column label="最近登录" width="180">
        <template #default="{ row }">{{ row.last_login_at ? formatTime(row.last_login_at) : '-' }}</template>
      </el-table-column>
      <el-table-column label="操作" width="280" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click.stop="openDetail(row)">详情</el-button>
          <el-button link type="primary" @click.stop="openNickname(row)">改昵称</el-button>
          <el-button
            v-if="!isSelf(row)"
            link
            type="warning"
            @click.stop="forceLogout(row)"
          >
            强制下线
          </el-button>
          <el-button
            v-if="row.status === 'active' && !isSelf(row)"
            link
            type="danger"
            @click.stop="openStatus(row, 'disable')"
          >
            停用
          </el-button>
          <el-button
            v-else-if="row.status !== 'active'"
            link
            type="success"
            @click.stop="openStatus(row, 'enable')"
          >
            启用
          </el-button>
        </template>
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

    <el-dialog v-model="createVisible" title="新建管理员" width="440px">
      <el-form label-width="80px">
        <el-form-item label="手机号" required>
          <el-input v-model="createForm.phone" placeholder="登录用手机号" maxlength="15" />
        </el-form-item>
        <el-form-item label="昵称">
          <el-input v-model="createForm.nickname" placeholder="可选，默认管理员" maxlength="32" />
        </el-form-item>
        <el-form-item label="角色" required>
          <el-select v-model="createForm.role" style="width: 100%">
            <el-option label="平台管理员" value="platform_admin" />
            <el-option label="俱乐部管理员" value="club_admin" />
            <el-option label="代理" value="agent" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :loading="acting" @click="confirmCreate">创建</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="nickVisible" title="修改昵称" width="420px">
      <el-input v-model="nickValue" maxlength="32" placeholder="新昵称" />
      <template #footer>
        <el-button @click="nickVisible = false">取消</el-button>
        <el-button type="primary" :loading="acting" @click="confirmNickname">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="dialogVisible" :title="statusKind === 'disable' ? '停用管理员' : '启用管理员'" width="420px">
      <el-input v-model="reason" type="textarea" :rows="3" placeholder="操作原因（可选）" />
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="acting" @click="confirmStatus">确认</el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="detailVisible" title="管理员详情" size="480px">
      <template v-if="detailUser">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="ID">{{ detailUser.user_id }}</el-descriptions-item>
          <el-descriptions-item label="手机号">{{ detailUser.phone_masked }}</el-descriptions-item>
          <el-descriptions-item label="昵称">{{ detailUser.nickname }}</el-descriptions-item>
          <el-descriptions-item label="角色">{{ roleLabel(detailUser.role) }}</el-descriptions-item>
          <el-descriptions-item label="状态">
            {{ detailUser.status === 'active' ? '正常' : '停用' }}
          </el-descriptions-item>
          <el-descriptions-item label="Auth账号">
            {{ detailUser.auth_account_id ?? '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="最近登录">
            {{ detailUser.last_login_at ? formatTime(detailUser.last_login_at) : '-' }}
          </el-descriptions-item>
        </el-descriptions>
        <h3 class="audit-title">近期操作记录</h3>
        <el-table :data="detailActions" size="small" empty-text="暂无记录">
          <el-table-column prop="action" label="动作" width="120">
            <template #default="{ row }">{{ actionLabel(row.action) }}</template>
          </el-table-column>
          <el-table-column prop="actor_admin_id" label="操作人" width="80" />
          <el-table-column prop="reason" label="原因" show-overflow-tooltip />
          <el-table-column label="时间" width="160">
            <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
          </el-table-column>
        </el-table>
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api } from '@/api/client'
import { useAuthStore } from '@/stores/auth'

interface AdminUser {
  user_id: number
  auth_account_id?: number
  phone_masked: string
  nickname: string
  role: string
  status: string
  created_at: string
  last_login_at?: string
}

interface AuditAction {
  id: number
  actor_admin_id: number
  action: string
  reason: string
  created_at: string
}

const auth = useAuthStore()
const loading = ref(false)
const acting = ref(false)
const users = ref<AdminUser[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const q = ref('')

const dialogVisible = ref(false)
const reason = ref('')
const current = ref<AdminUser | null>(null)
const statusKind = ref<'disable' | 'enable'>('disable')

const createVisible = ref(false)
const createForm = ref({ phone: '', nickname: '', role: 'club_admin' })

const nickVisible = ref(false)
const nickValue = ref('')

const detailVisible = ref(false)
const detailUser = ref<(AdminUser & { auth_account_id?: number }) | null>(null)
const detailActions = ref<AuditAction[]>([])

function isSelf(row: AdminUser): boolean {
  return !!auth.user && auth.user.user_id === row.user_id
}

function roleLabel(role: string): string {
  switch (role) {
    case 'platform_admin':
      return '平台管理员'
    case 'club_admin':
      return '俱乐部管理员'
    case 'agent':
      return '代理'
    default:
      return role
  }
}

function actionLabel(action: string): string {
  const map: Record<string, string> = {
    create: '创建',
    update_nickname: '改昵称',
    set_role: '改角色',
    disable: '停用',
    enable: '启用',
    force_logout: '强制下线',
  }
  return map[action] || action
}

function formatTime(value: string): string {
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return value
  return d.toLocaleString('zh-CN')
}

async function load() {
  loading.value = true
  try {
    const qs = new URLSearchParams({
      page: String(page.value),
      page_size: String(pageSize.value),
    })
    if (q.value.trim()) qs.set('q', q.value.trim())
    const res = await api.get<{
      users: AdminUser[]
      total: number
      page: number
      page_size: number
    }>(`/admin/users?${qs}`)
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

function search() {
  page.value = 1
  load()
}

function onPageSizeChange() {
  page.value = 1
  load()
}

function openCreate() {
  createForm.value = { phone: '', nickname: '', role: 'club_admin' }
  createVisible.value = true
}

async function confirmCreate() {
  const phone = createForm.value.phone.replace(/\D/g, '')
  if (phone.length < 8) {
    ElMessage.warning('请输入有效手机号')
    return
  }
  acting.value = true
  try {
    await api.post('/admin/users', {
      phone,
      nickname: createForm.value.nickname.trim(),
      role: createForm.value.role,
    })
    ElMessage.success('已创建')
    createVisible.value = false
    await load()
  } catch (e: unknown) {
    const err = e as { message?: string }
    ElMessage.error(err.message || '创建失败')
  } finally {
    acting.value = false
  }
}

function openNickname(row: AdminUser) {
  current.value = row
  nickValue.value = row.nickname
  nickVisible.value = true
}

async function confirmNickname() {
  if (!current.value) return
  const nickname = nickValue.value.trim()
  if (!nickname) {
    ElMessage.warning('昵称不能为空')
    return
  }
  acting.value = true
  try {
    await api.patch(`/admin/users/${current.value.user_id}`, { nickname })
    ElMessage.success('昵称已更新')
    nickVisible.value = false
    await load()
  } catch (e: unknown) {
    const err = e as { message?: string }
    ElMessage.error(err.message || '修改失败')
  } finally {
    acting.value = false
  }
}

async function openDetail(row: AdminUser) {
  detailVisible.value = true
  detailUser.value = null
  detailActions.value = []
  try {
    const res = await api.get<{
      user: AdminUser
      actions: AuditAction[]
    }>(`/admin/users/${row.user_id}`)
    detailUser.value = res.user
    detailActions.value = res.actions || []
  } catch (e: unknown) {
    const err = e as { message?: string }
    ElMessage.error(err.message || '加载详情失败')
  }
}

async function forceLogout(row: AdminUser) {
  try {
    await ElMessageBox.confirm(`强制下线「${row.nickname || row.user_id}」？将吊销其 refresh token。`, '强制下线')
    await api.post(`/admin/users/${row.user_id}/logout`, { reason: '' })
    ElMessage.success('已强制下线')
  } catch (e: unknown) {
    if (e === 'cancel') return
    const err = e as { message?: string }
    ElMessage.error(err.message || '操作失败')
  }
}

function openStatus(row: AdminUser, kind: 'disable' | 'enable') {
  current.value = row
  statusKind.value = kind
  reason.value = ''
  dialogVisible.value = true
}

async function confirmStatus() {
  if (!current.value) return
  acting.value = true
  try {
    const id = current.value.user_id
    const body = { reason: reason.value.trim() }
    if (statusKind.value === 'disable') {
      await api.post(`/admin/users/${id}/disable`, body)
      ElMessage.success('已停用')
    } else {
      await api.post(`/admin/users/${id}/enable`, body)
      ElMessage.success('已启用')
    }
    dialogVisible.value = false
    await load()
  } catch (e: unknown) {
    const err = e as { message?: string }
    ElMessage.error(err.message || '操作失败')
  } finally {
    acting.value = false
  }
}

async function changeRole(row: AdminUser, role: string) {
  if (role === row.role) return
  try {
    await ElMessageBox.confirm(`将「${row.nickname || row.user_id}」角色改为 ${roleLabel(role)}？`, '修改角色')
    await api.patch(`/admin/users/${row.user_id}/role`, { role })
    ElMessage.success('角色已更新（已同步 Auth）')
    await load()
  } catch (e: unknown) {
    if (e === 'cancel') return
    const err = e as { message?: string }
    ElMessage.error(err.message || '修改失败')
    await load()
  }
}

onMounted(load)
</script>

<style scoped>
.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}
.toolbar h2 {
  margin: 0;
}
.toolbar-right {
  display: flex;
  gap: 8px;
  align-items: center;
}
.pager {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
.audit-title {
  margin: 20px 0 12px;
  font-size: 15px;
}
</style>
