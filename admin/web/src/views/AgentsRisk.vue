<template>
  <div v-loading="loading">
    <div class="toolbar">
      <h2>代理保证金 / 提现</h2>
      <div class="toolbar-right">
        <el-input
          v-model="agentId"
          placeholder="代理 admin_users.id"
          style="width: 200px"
          @keyup.enter="loadAll"
        />
        <el-button type="primary" @click="loadAll">查询</el-button>
      </div>
    </div>

    <el-card v-if="deposit" class="block">
      <template #header>
        <span>保证金：余额 {{ deposit.balance_cny }} / 应缴 {{ deposit.required_cny }}</span>
      </template>
      <div class="row">
        <el-select v-model="depKind" style="width: 120px">
          <el-option label="缴纳" value="deposit" />
          <el-option label="扣罚" value="penalty" />
        </el-select>
        <el-input-number v-model="depAmount" :min="1" :step="100" />
        <el-input v-model="depReason" placeholder="原因" style="width: 220px" />
        <el-button type="primary" :loading="acting" @click="postDeposit">提交</el-button>
      </div>
      <el-table :data="deposit.ledger || []" size="small" style="margin-top: 12px">
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="kind" label="类型" width="90" />
        <el-table-column prop="amount_cny" label="金额" width="100" />
        <el-table-column prop="balance_after" label="余额后" width="100" />
        <el-table-column prop="reason" label="原因" />
        <el-table-column label="时间" width="180">
          <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card v-if="agentId" class="block">
      <template #header>
        <div class="card-head">
          <span>提现申请</span>
          <div class="row">
            <el-input-number v-model="wdAmount" :min="1" :step="100" />
            <el-button @click="createWithdrawal">新建申请</el-button>
          </div>
        </div>
      </template>
      <el-table :data="withdrawals" size="small">
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="amount_cny" label="金额" width="100" />
        <el-table-column prop="status" label="状态" width="100" />
        <el-table-column label="可审时间" width="180">
          <template #default="{ row }">{{ formatTime(row.eligible_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="220">
          <template #default="{ row }">
            <template v-if="row.status === 'pending'">
              <el-button link type="success" @click="reviewWd(row, true, false)">通过</el-button>
              <el-button link type="danger" @click="reviewWd(row, false, false)">拒绝</el-button>
              <el-button link type="warning" @click="reviewWd(row, true, true)">强制通过</el-button>
            </template>
            <span v-else>{{ row.review_note || '-' }}</span>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api } from '@/api/client'

interface DepositInfo {
  agent_admin_id: number
  required_cny: number
  balance_cny: number
  ledger: Array<Record<string, unknown>>
}

interface Withdrawal {
  id: number
  amount_cny: number
  status: string
  eligible_at: string
  review_note?: string
}

const loading = ref(false)
const acting = ref(false)
const agentId = ref('')
const deposit = ref<DepositInfo | null>(null)
const withdrawals = ref<Withdrawal[]>([])

const depKind = ref('deposit')
const depAmount = ref(5000)
const depReason = ref('')
const wdAmount = ref(100)

function formatTime(v: string) {
  return v ? new Date(v).toLocaleString() : '-'
}

async function loadAll() {
  const id = Number(agentId.value)
  if (!id) {
    ElMessage.warning('请输入代理 ID')
    return
  }
  loading.value = true
  try {
    deposit.value = await api.get<DepositInfo>(`/admin/risk/agents/${id}/deposit`)
    const res = await api.get<{ withdrawals: Withdrawal[] }>(`/admin/risk/agents/${id}/withdrawals`)
    withdrawals.value = res.withdrawals || []
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

async function postDeposit() {
  const id = Number(agentId.value)
  if (!id) return
  acting.value = true
  try {
    await api.post(`/admin/risk/agents/${id}/deposit`, {
      kind: depKind.value,
      amount_cny: depAmount.value,
      reason: depReason.value,
    })
    ElMessage.success('已记账')
    await loadAll()
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '提交失败')
  } finally {
    acting.value = false
  }
}

async function createWithdrawal() {
  const id = Number(agentId.value)
  if (!id) return
  try {
    await api.post(`/admin/risk/agents/${id}/withdrawals`, { amount_cny: wdAmount.value })
    ElMessage.success('已申请')
    await loadAll()
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '申请失败')
  }
}

async function reviewWd(row: Withdrawal, approve: boolean, force: boolean) {
  const id = Number(agentId.value)
  let note = ''
  if (force) {
    try {
      const { value } = await ElMessageBox.prompt('强制通过须填写原因', '强制审核', {
        confirmButtonText: '确认',
        cancelButtonText: '取消',
        inputPattern: /.+/,
        inputErrorMessage: '必填',
      })
      note = value
    } catch {
      return
    }
  }
  try {
    await api.post(`/admin/risk/agents/${id}/withdrawals/${row.id}/review`, {
      approve,
      force,
      note,
    })
    ElMessage.success('已审核')
    await loadAll()
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '审核失败')
  }
}
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
}
.block {
  margin-bottom: 16px;
}
.row {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
}
.card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
</style>
