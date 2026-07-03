<template>
  <div v-loading="loading">
    <h2>运营仪表盘</h2>
    <el-row :gutter="16" class="cards">
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="kpi-label">今日 GMV（分）</div>
          <div class="kpi-value">{{ overview?.gmv_today_cents ?? '-' }}</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="kpi-label">今日房卡销售</div>
          <div class="kpi-value">{{ overview?.room_cards_sold_today ?? '-' }}</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="kpi-label">今日房卡消耗</div>
          <div class="kpi-value">{{ overview?.room_cards_used_today ?? '-' }}</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="kpi-label">活跃俱乐部</div>
          <div class="kpi-value">{{ overview?.active_clubs ?? '-' }}</div>
        </el-card>
      </el-col>
    </el-row>

    <el-card style="margin-top: 24px">
      <template #header>房卡 7 日趋势</template>
      <el-table :data="trend" stripe>
        <el-table-column prop="date" label="日期" />
        <el-table-column prop="sold" label="销售" />
        <el-table-column prop="used" label="消耗" />
        <el-table-column label="消耗率">
          <template #default="{ row }">
            {{ row.sold > 0 ? ((row.used / row.sold) * 100).toFixed(1) + '%' : '-' }}
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '@/api/client'

interface Overview {
  gmv_today_cents: number
  room_cards_sold_today: number
  room_cards_used_today: number
  active_clubs: number
  rooms_created_today: number
}

interface DayRow {
  date: string
  sold: number
  used: number
}

const loading = ref(false)
const overview = ref<Overview | null>(null)
const trend = ref<DayRow[]>([])

onMounted(async () => {
  loading.value = true
  try {
    overview.value = await api.get<Overview>('/admin/metrics/overview')
    const res = await api.get<{ days: DayRow[] }>('/admin/metrics/room-cards?days=7')
    trend.value = res.days
  } catch (e: unknown) {
    const err = e as { message?: string }
    ElMessage.error(err.message || '加载失败')
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.cards {
  margin-top: 16px;
}
.kpi-label {
  color: #909399;
  font-size: 14px;
}
.kpi-value {
  font-size: 28px;
  font-weight: bold;
  margin-top: 8px;
}
</style>
