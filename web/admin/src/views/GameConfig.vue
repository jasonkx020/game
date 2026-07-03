<template>
  <div v-loading="loading">
    <h2>游戏配置（只读）</h2>
    <el-table :data="games" stripe style="margin-top: 16px">
      <el-table-column prop="game_id" label="游戏 ID" />
      <el-table-column prop="name" label="名称" />
      <el-table-column prop="min_players" label="最少人数" />
      <el-table-column prop="max_players" label="最多人数" />
      <el-table-column label="操作" width="120">
        <template #default="{ row }">
          <el-button link type="primary" @click="loadConfig(row.game_id)">查看 ops-hooks</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-card v-if="config" style="margin-top: 24px">
      <template #header>{{ config.game_id }} ops-hooks</template>
      <pre>{{ JSON.stringify(config, null, 2) }}</pre>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '@/api/client'

interface Game {
  game_id: string
  name: string
  min_players: number
  max_players: number
}

const loading = ref(false)
const games = ref<Game[]>([])
const config = ref<Record<string, unknown> | null>(null)

async function loadGames() {
  loading.value = true
  try {
    const res = await api.get<{ games: Game[] }>('/games')
    games.value = res.games
    if (games.value.length > 0) {
      await loadConfig(games.value[0].game_id)
    }
  } catch (e: unknown) {
    const err = e as { message?: string }
    ElMessage.error(err.message || '加载失败')
  } finally {
    loading.value = false
  }
}

async function loadConfig(gameId: string) {
  try {
    config.value = await api.get<Record<string, unknown>>(`/games/${gameId}/config`)
  } catch (e: unknown) {
    const err = e as { message?: string }
    ElMessage.error(err.message || '加载配置失败')
  }
}

onMounted(loadGames)
</script>

<style scoped>
pre {
  background: #f5f7fa;
  padding: 16px;
  border-radius: 4px;
  overflow: auto;
}
</style>
