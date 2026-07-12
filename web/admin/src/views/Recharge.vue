<template>
  <div v-loading="loading">
    <h2>Mock 房卡充值</h2>
    <p class="hint">开发环境直接到账，写入 recharge_order 与 wallet_ledger（对指定游戏玩家操作）。</p>

    <el-form inline style="margin-bottom: 16px">
      <el-form-item label="玩家 ID">
        <el-input v-model="playerId" placeholder="测试玩家 ID" style="width: 160px" />
      </el-form-item>
      <el-button type="primary" @click="load">查询余额</el-button>
    </el-form>

    <el-row :gutter="16" class="products">
      <el-col v-for="p in products" :key="p.id" :span="8">
        <el-card shadow="hover">
          <h3>{{ p.label }}</h3>
          <p>{{ p.cards }} 张房卡 · ¥{{ p.price }}</p>
          <el-button type="primary" @click="recharge(p.id)">购买</el-button>
        </el-card>
      </el-col>
    </el-row>

    <el-card style="margin-top: 24px">
      <template #header>玩家 {{ playerId }} 当前余额：{{ balance }} 张</template>
      <el-table :data="history" stripe>
        <el-table-column prop="product_id" label="产品" />
        <el-table-column prop="amount_cny" label="金额(元)" />
        <el-table-column prop="cards" label="房卡" />
        <el-table-column prop="audit_sn" label="audit_sn" />
        <el-table-column prop="created_at" label="时间" />
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '@/api/client'

const products = [
  { id: 'rc_10', label: '小包', cards: 10, price: 6 },
  { id: 'rc_50', label: '中包', cards: 50, price: 28 },
  { id: 'rc_200', label: '大包', cards: 200, price: 98 },
]

interface Order {
  product_id: string
  amount_cny: number
  cards: number
  audit_sn: number
  created_at: string
}

const loading = ref(false)
const balance = ref(0)
const history = ref<Order[]>([])
const playerId = ref('')

function queryPlayerId(): number {
  return parseInt(playerId.value, 10)
}

async function load() {
  const pid = queryPlayerId()
  if (!pid) {
    ElMessage.warning('请填写玩家 ID')
    return
  }
  loading.value = true
  try {
    const bal = await api.get<{ balance: number }>(`/admin/wallet/room-card?player_id=${pid}`)
    balance.value = bal.balance
    const res = await api.get<{ orders: Order[] }>(`/admin/wallet/recharge/history?player_id=${pid}`)
    history.value = res.orders || []
  } catch (e: unknown) {
    const err = e as { message?: string }
    ElMessage.error(err.message || '加载失败')
  } finally {
    loading.value = false
  }
}

async function recharge(productId: string) {
  const pid = queryPlayerId()
  if (!pid) {
    ElMessage.warning('请填写玩家 ID')
    return
  }
  try {
    const res = await api.post<{ balance: number; audit_sn: number }>(
      '/admin/wallet/room-card/recharge',
      { product_id: productId, player_id: pid },
    )
    balance.value = res.balance
    ElMessage.success(`充值成功 audit_sn=${res.audit_sn}`)
    await load()
  } catch (e: unknown) {
    const err = e as { message?: string }
    ElMessage.error(err.message || '充值失败')
  }
}

onMounted(() => {})
</script>

<style scoped>
.hint {
  color: #909399;
}
.products {
  margin-top: 16px;
}
</style>
