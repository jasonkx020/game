<template>
  <div class="login-page">
    <el-card class="login-card">
      <h2>游戏运营后台</h2>
      <el-form @submit.prevent="onSubmit">
        <el-form-item label="手机号">
          <el-input v-model="phone" placeholder="13800000000" />
        </el-form-item>
        <el-form-item label="验证码">
          <el-input v-model="smsCode" placeholder="dev: 123456" />
        </el-form-item>
        <el-button type="primary" :loading="loading" native-type="submit" style="width: 100%">
          登录
        </el-button>
      </el-form>
      <p class="hint">开发环境验证码：123456；平台管理员：13800000000</p>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth'

const phone = ref('13800000000')
const smsCode = ref('123456')
const loading = ref(false)
const router = useRouter()
const auth = useAuthStore()

async function onSubmit() {
  loading.value = true
  try {
    await auth.login(phone.value, smsCode.value)
    ElMessage.success('登录成功')
    router.push('/dashboard')
  } catch (e: unknown) {
    const err = e as { message?: string }
    ElMessage.error(err.message || '登录失败')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
}
.login-card {
  width: 400px;
}
.hint {
  margin-top: 16px;
  color: #909399;
  font-size: 12px;
}
</style>
