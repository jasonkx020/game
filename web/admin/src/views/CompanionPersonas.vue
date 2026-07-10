<template>
  <div>
    <h2>伴侣人设（只读）</h2>
    <p class="hint">运营可后续扩展编辑；当前展示 companion_persona 表数据。</p>
    <table v-if="personas.length">
      <thead>
        <tr>
          <th>ID</th>
          <th>名称</th>
          <th>风格</th>
          <th>头像</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="p in personas" :key="p.persona_id">
          <td>{{ p.persona_id }}</td>
          <td>{{ p.name }}</td>
          <td>{{ p.voice_style || '-' }}</td>
          <td>{{ p.avatar_url || '-' }}</td>
        </tr>
      </tbody>
    </table>
    <p v-else>加载中或暂无人设…</p>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '@/api/client'

const personas = ref<Array<Record<string, string>>>([])

onMounted(async () => {
  try {
    const res = await api.get<{ personas: Array<Record<string, string>> }>('/v1/companion/personas')
    personas.value = res.personas ?? []
  } catch (e) {
    console.error(e)
  }
})
</script>

<style scoped>
.hint { color: #666; font-size: 14px; }
table { width: 100%; border-collapse: collapse; margin-top: 12px; }
th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
</style>
