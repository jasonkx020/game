<template>
  <div v-loading="loading">
    <el-page-header @back="$router.push('/clubs')">
      <template #content>{{ club?.name || '俱乐部详情' }}</template>
    </el-page-header>

    <el-row :gutter="16" style="margin-top: 16px">
      <el-col :span="8">
        <el-card>
          <div class="kpi-label">房卡池余额</div>
          <div class="kpi-value">{{ club?.pool_balance ?? 0 }}</div>
          <el-form inline style="margin-top: 12px">
            <el-input-number v-model="transferAmount" :min="1" />
            <el-button type="primary" @click="transfer">划拨房卡</el-button>
          </el-form>
        </el-card>
      </el-col>
    </el-row>

    <el-card style="margin-top: 16px">
      <template #header>
        <div class="member-header">
          <span>成员列表</span>
          <el-form inline>
            <el-input v-model="newUserId" placeholder="用户 ID" style="width: 120px" />
            <el-button type="primary" @click="addMember">添加成员</el-button>
          </el-form>
        </div>
      </template>
      <el-table :data="members" stripe>
        <el-table-column prop="user_id" label="用户 ID" />
        <el-table-column prop="nickname" label="昵称" />
        <el-table-column prop="phone" label="手机号" />
        <el-table-column prop="role" label="角色" />
        <el-table-column label="操作" width="100">
          <template #default="{ row }">
            <el-button link type="danger" @click="removeMember(row.user_id)">踢出</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { api } from '@/api/client'

interface ClubDetail {
  id: number
  name: string
  owner_user_id: number
  status: string
  pool_balance: number
}

interface Member {
  user_id: number
  nickname: string
  phone: string
  role: string
}

const route = useRoute()
const clubId = route.params.id as string
const loading = ref(false)
const club = ref<ClubDetail | null>(null)
const members = ref<Member[]>([])
const transferAmount = ref(10)
const newUserId = ref('')

async function load() {
  loading.value = true
  try {
    club.value = await api.get<ClubDetail>(`/clubs/${clubId}`)
    const res = await api.get<{ members: Member[] }>(`/clubs/${clubId}/members`)
    members.value = res.members
  } catch (e: unknown) {
    const err = e as { message?: string }
    ElMessage.error(err.message || '加载失败')
  } finally {
    loading.value = false
  }
}

async function transfer() {
  try {
    const res = await api.post<{ pool_balance: number }>(`/clubs/${clubId}/room-card/transfer`, {
      amount: transferAmount.value,
    })
    if (club.value) club.value.pool_balance = res.pool_balance
    ElMessage.success('划拨成功')
  } catch (e: unknown) {
    const err = e as { message?: string }
    ElMessage.error(err.message || '划拨失败')
  }
}

async function addMember() {
  const uid = parseInt(newUserId.value, 10)
  if (!uid) return
  try {
    await api.post(`/clubs/${clubId}/members`, { user_id: uid })
    ElMessage.success('已添加')
    newUserId.value = ''
    await load()
  } catch (e: unknown) {
    const err = e as { message?: string }
    ElMessage.error(err.message || '添加失败')
  }
}

async function removeMember(userId: number) {
  try {
    await api.delete(`/clubs/${clubId}/members/${userId}`)
    ElMessage.success('已踢出')
    await load()
  } catch (e: unknown) {
    const err = e as { message?: string }
    ElMessage.error(err.message || '操作失败')
  }
}

onMounted(load)
</script>

<style scoped>
.kpi-label {
  color: #909399;
}
.kpi-value {
  font-size: 32px;
  font-weight: bold;
}
.member-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
