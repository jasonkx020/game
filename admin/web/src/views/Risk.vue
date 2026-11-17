<template>
  <div v-loading="loading">
    <div class="toolbar">
      <h2>风控告警</h2>
      <div class="toolbar-right">
        <el-select v-model="typ" clearable placeholder="类型" style="width: 140px" @change="reload">
          <el-option label="同 IP" value="same_ip" />
          <el-option label="GPS 邻近" value="gps_near" />
          <el-option label="同设备" value="device" />
          <el-option label="异常输赢" value="abnormal_pnl" />
          <el-option label="未成年" value="minor" />
        </el-select>
        <el-select v-model="status" clearable placeholder="状态" style="width: 120px" @change="reload">
          <el-option label="待复核" value="open" />
          <el-option label="已确认" value="reviewed" />
          <el-option label="已忽略" value="dismissed" />
        </el-select>
        <el-button type="primary" @click="reload">刷新</el-button>
      </div>
    </div>

    <el-table :data="alerts" stripe>
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="type" label="类型" width="120" />
      <el-table-column prop="status" label="状态" width="100" />
      <el-table-column prop="player_id" label="玩家" width="100" />
      <el-table-column label="房间" width="120">
        <template #default="{ row }">
          <span v-if="row.room_id">{{ shortId(row.room_id) }}</span>
          <span v-else>-</span>
        </template>
      </el-table-column>
      <el-table-column label="局" width="120">
        <template #default="{ row }">
          <el-button
            v-if="row.round_id"
            link
            type="primary"
            @click="openReplay(row.round_id)"
          >
            {{ shortId(row.round_id) }}
          </el-button>
          <span v-else>-</span>
        </template>
      </el-table-column>
      <el-table-column label="时间" width="180">
        <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="200" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openReview(row)">复核</el-button>
          <el-button
            v-if="row.room_id"
            link
            @click="loadSnapshot(row.room_id)"
          >
            同桌
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="pager">
      <el-pagination
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :total="total"
        :page-sizes="[10, 20, 50]"
        layout="total, sizes, prev, pager, next"
        @current-change="load"
        @size-change="onPageSizeChange"
      />
    </div>

    <el-dialog v-model="reviewVisible" title="复核告警" width="480px">
      <p v-if="current">告警 #{{ current.id }} · {{ current.type }}</p>
      <el-input v-model="note" type="textarea" :rows="3" placeholder="备注" />
      <el-checkbox v-model="freeze" style="margin-top: 12px">确认后冻结玩家</el-checkbox>
      <template #footer>
        <el-button @click="submitReview('dismissed')">忽略</el-button>
        <el-button type="danger" :loading="acting" @click="submitReview('confirmed')">确认</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="snapVisible" title="同桌摘要" width="640px">
      <el-table :data="snapshot" size="small">
        <el-table-column prop="player_id" label="玩家" width="100" />
        <el-table-column prop="ip" label="IP" />
        <el-table-column prop="device_id" label="设备" />
        <el-table-column label="GPS">
          <template #default="{ row }">
            <span v-if="row.lat != null">{{ row.lat }}, {{ row.lng }}</span>
            <span v-else>-</span>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '@/api/client'

interface Alert {
  id: number
  type: string
  status: string
  player_id?: number
  room_id?: string
  round_id?: string
  created_at: string
  meta?: unknown
}

const loading = ref(false)
const acting = ref(false)
const alerts = ref<Alert[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const typ = ref('')
const status = ref('open')

const reviewVisible = ref(false)
const current = ref<Alert | null>(null)
const note = ref('')
const freeze = ref(false)

const snapVisible = ref(false)
const snapshot = ref<Array<Record<string, unknown>>>([])

function formatTime(v: string) {
  return v ? new Date(v).toLocaleString() : '-'
}

function shortId(id: string) {
  return id.slice(0, 8)
}

function openReplay(roundId: string) {
  window.open(`/docs?q=replay&round_id=${roundId}`, '_blank')
}

async function load() {
  loading.value = true
  try {
    const qs = new URLSearchParams({
      page: String(page.value),
      page_size: String(pageSize.value),
    })
    if (typ.value) qs.set('type', typ.value)
    if (status.value) qs.set('status', status.value)
    const res = await api.get<{ alerts: Alert[]; total: number }>(`/admin/risk/alerts?${qs}`)
    alerts.value = res.alerts || []
    total.value = res.total || 0
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

function reload() {
  page.value = 1
  load()
}

function onPageSizeChange() {
  page.value = 1
  load()
}

function openReview(row: Alert) {
  current.value = row
  note.value = ''
  freeze.value = false
  reviewVisible.value = true
}

async function submitReview(decision: 'dismissed' | 'confirmed') {
  if (!current.value) return
  acting.value = true
  try {
    await api.post(`/admin/risk/alerts/${current.value.id}/review`, {
      decision,
      note: note.value,
      freeze: freeze.value && decision === 'confirmed',
    })
    ElMessage.success('已提交')
    reviewVisible.value = false
    await load()
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '复核失败')
  } finally {
    acting.value = false
  }
}

async function loadSnapshot(roomId: string) {
  try {
    const res = await api.get<{ seats: Array<Record<string, unknown>> }>(
      `/admin/risk/table-snapshot?room_id=${roomId}`,
    )
    snapshot.value = res.seats || []
    snapVisible.value = true
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载同桌失败')
  }
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
.toolbar-right {
  display: flex;
  gap: 8px;
  align-items: center;
}
.pager {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}
</style>
