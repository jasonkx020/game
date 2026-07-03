<template>
  <div v-loading="loading">
    <div class="toolbar">
      <h2>俱乐部管理</h2>
      <el-button type="primary" @click="showCreate = true">创建俱乐部</el-button>
    </div>

    <el-table :data="clubs" stripe>
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="name" label="名称" />
      <el-table-column prop="owner_user_id" label="群主 ID" />
      <el-table-column prop="status" label="状态" />
      <el-table-column label="操作" width="120">
        <template #default="{ row }">
          <el-button link type="primary" @click="$router.push(`/clubs/${row.id}`)">详情</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="showCreate" title="创建俱乐部" width="400px">
      <el-input v-model="newName" placeholder="俱乐部名称" />
      <template #footer>
        <el-button @click="showCreate = false">取消</el-button>
        <el-button type="primary" :loading="creating" @click="createClub">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '@/api/client'

interface Club {
  id: number
  name: string
  owner_user_id: number
  status: string
}

const loading = ref(false)
const creating = ref(false)
const clubs = ref<Club[]>([])
const showCreate = ref(false)
const newName = ref('')

async function load() {
  loading.value = true
  try {
    const res = await api.get<{ clubs: Club[] }>('/admin/clubs')
    clubs.value = res.clubs
  } catch (e: unknown) {
    const err = e as { message?: string }
    ElMessage.error(err.message || '加载失败')
  } finally {
    loading.value = false
  }
}

async function createClub() {
  if (!newName.value) return
  creating.value = true
  try {
    await api.post('/clubs', { name: newName.value })
    ElMessage.success('创建成功')
    showCreate.value = false
    newName.value = ''
    await load()
  } catch (e: unknown) {
    const err = e as { message?: string }
    ElMessage.error(err.message || '创建失败')
  } finally {
    creating.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
