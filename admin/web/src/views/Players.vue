<template>
  <div v-loading="loading">
    <div class="toolbar">
      <h2>游戏玩家</h2>
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
      </div>
    </div>

    <el-table :data="players" stripe>
      <el-table-column prop="player_id" label="ID" width="80" />
      <el-table-column prop="phone_masked" label="手机号" width="140" />
      <el-table-column prop="nickname" label="昵称" />
      <el-table-column label="状态" width="110">
        <template #default="{ row }">
          <el-tag :type="row.status === 'active' ? 'success' : 'danger'" size="small">
            {{ row.status === 'active' ? '正常' : '封禁' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="注册时间" width="180">
        <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
      </el-table-column>
      <el-table-column label="最近登录" width="180">
        <template #default="{ row }">{{ row.last_login_at ? formatTime(row.last_login_at) : '-' }}</template>
      </el-table-column>
      <el-table-column label="操作" width="260" fixed="right">
        <template #default="{ row }">
          <el-button link type="warning" @click="openAction(row, 'kick')">踢下线</el-button>
          <el-button
            v-if="row.status === 'active'"
            link
            type="danger"
            @click="openAction(row, 'ban')"
          >
            封禁
          </el-button>
          <el-button v-else link type="success" @click="openAction(row, 'unban')">解封</el-button>
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

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="420px">
      <el-input v-model="reason" type="textarea" :rows="3" placeholder="操作原因（可选）" />
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="acting" @click="confirmAction">确认</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '@/api/client'

interface GamePlayer {
  player_id: number
  phone_masked: string
  nickname: string
  status: string
  created_at: string
  last_login_at?: string
}

type ActionKind = 'ban' | 'unban' | 'kick'

const loading = ref(false)
const acting = ref(false)
const players = ref<GamePlayer[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const q = ref('')

const dialogVisible = ref(false)
const reason = ref('')
const current = ref<GamePlayer | null>(null)
const actionKind = ref<ActionKind>('kick')

const dialogTitle = computed(() => {
  if (actionKind.value === 'ban') return '封禁玩家'
  if (actionKind.value === 'unban') return '解封玩家'
  return '踢玩家下线'
})

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
      players: GamePlayer[]
      total: number
      page: number
      page_size: number
    }>(`/admin/players?${qs}`)
    players.value = res.players
    total.value = res.total
    page.value = res.page
    pageSize.value = res.page_size
  } catch (e: unknown) {
    const err = e as { message?: string; code?: number }
    if (err.code === 403) {
      ElMessage.error('无权限：仅平台管理员可查看游戏玩家')
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

function openAction(row: GamePlayer, kind: ActionKind) {
  current.value = row
  actionKind.value = kind
  reason.value = ''
  dialogVisible.value = true
}

async function confirmAction() {
  if (!current.value) return
  acting.value = true
  try {
    const id = current.value.player_id
    const body = { reason: reason.value.trim() }
    if (actionKind.value === 'ban') {
      await api.post(`/admin/players/${id}/ban`, body)
      ElMessage.success('已封禁')
    } else if (actionKind.value === 'unban') {
      await api.post(`/admin/players/${id}/unban`, body)
      ElMessage.success('已解封')
    } else {
      await api.post(`/admin/players/${id}/kick`, body)
      ElMessage.success('已发送踢下线')
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
</style>
