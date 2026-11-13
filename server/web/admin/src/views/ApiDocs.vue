<template>
  <div class="docs-center" :class="{ standalone: isStandalone }">
    <div v-if="isStandalone" class="docs-topbar">
      <span>GameS 协议与接入文档</span>
      <router-link to="/login">管理员登录</router-link>
    </div>

    <el-container class="docs-body">
      <el-aside width="280px" class="docs-aside">
        <div class="aside-head">
          <h2>文档中心</h2>
          <p>协议 · OpenAPI · 接入指南</p>
          <el-input v-model="filter" placeholder="筛选文档…" clearable size="small" />
        </div>
        <el-menu
          :default-active="activeId"
          :default-openeds="['integrate', 'protocol', 'architecture']"
          class="docs-menu"
          @select="onSelect"
        >
          <el-sub-menu v-for="sec in filteredSections" :key="sec.id" :index="sec.id">
            <template #title>{{ sec.title }}</template>
            <el-menu-item v-for="item in sec.items" :key="item.id" :index="item.id">
              <span class="menu-title">{{ item.title }}</span>
              <el-tag v-if="item.kind === 'openapi'" size="small" type="success" effect="plain">API</el-tag>
            </el-menu-item>
          </el-sub-menu>
        </el-menu>
      </el-aside>

      <el-main class="docs-main">
        <div class="main-toolbar" v-if="current">
          <div>
            <h3>{{ current.title }}</h3>
            <p v-if="current.description" class="desc">{{ current.description }}</p>
          </div>
          <div class="toolbar-actions">
            <el-button size="small" @click="openRaw">打开源文件</el-button>
            <el-button v-if="current.kind === 'openapi'" size="small" @click="reloadSwagger">刷新</el-button>
          </div>
        </div>

        <el-alert
          v-if="current?.id === 'auth-openapi' || current?.id === 'auth-guide'"
          type="info"
          :closable="false"
          show-icon
          class="hint"
          title="Auth 直连 :8081；业务 API 用共享 JWT_SECRET 验未过期 access。Refresh 须同时上传 access_token（可过期）与 refresh_token；失败时用同一对旧票重试（服务端有短宽限）。"
        />

        <div v-if="error" class="docs-error">{{ error }}</div>

        <div v-show="current?.kind === 'openapi'" ref="swaggerHost" class="swagger-host" />
        <article
          v-show="current?.kind === 'markdown'"
          class="md-host markdown-body"
          v-html="html"
          @click="onMarkdownClick"
        />
      </el-main>
    </el-container>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { marked } from 'marked'
import SwaggerUI from 'swagger-ui-dist/swagger-ui-es-bundle.js'
import 'swagger-ui-dist/swagger-ui.css'
import {
  defaultDocId,
  docCatalog,
  findDoc,
  resolveMarkdownLink,
  type DocItem,
  type DocSection,
} from '@/docs/catalog'
import { bundleOpenApi } from '@/docs/bundleOpenApi'

const route = useRoute()
const router = useRouter()
const isStandalone = computed(() => route.path === '/docs' || route.path.startsWith('/docs/'))

const filter = ref('')
const activeId = ref(defaultDocId())
const current = ref<DocItem | undefined>(findDoc(activeId.value))
const html = ref('')
const error = ref('')
const swaggerHost = ref<HTMLElement | null>(null)
let ui: { destroy?: () => void } | null = null

const filteredSections = computed<DocSection[]>(() => {
  const q = filter.value.trim().toLowerCase()
  if (!q) return docCatalog
  return docCatalog
    .map((s) => ({
      ...s,
      items: s.items.filter(
        (i) =>
          i.title.toLowerCase().includes(q) ||
          i.id.toLowerCase().includes(q) ||
          (i.description || '').toLowerCase().includes(q),
      ),
    }))
    .filter((s) => s.items.length > 0)
})

function destroySwagger() {
  if (ui && typeof ui.destroy === 'function') {
    try {
      ui.destroy()
    } catch {
      /* ignore */
    }
  }
  ui = null
  if (swaggerHost.value) swaggerHost.value.innerHTML = ''
}

async function loadMarkdown(src: string) {
  error.value = ''
  html.value = '<p>加载中…</p>'
  try {
    const res = await fetch(src)
    if (!res.ok) throw new Error(`无法加载 ${src}（${res.status}）`)
    const text = await res.text()
    html.value = await marked.parse(text, { gfm: true, breaks: false })
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
    html.value = ''
  }
}

async function loadSwagger(src: string) {
  error.value = ''
  destroySwagger()
  if (!swaggerHost.value) return
  try {
    // 先本地展开多文件 $ref，避免 swagger-ui 对 path 外置片段的相对引用解析失败
    const spec = await bundleOpenApi(src)
    if (!swaggerHost.value) return
    ui = SwaggerUI({
      domNode: swaggerHost.value,
      spec,
      docExpansion: 'list',
      defaultModelsExpandDepth: 1,
      deepLinking: true,
      tryItOutEnabled: false,
      persistAuthorization: false,
      filter: true,
      syntaxHighlight: { activate: true, theme: 'agate' },
    }) as { destroy?: () => void }
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  }
}

async function showDoc(id: string) {
  const doc = findDoc(id)
  if (!doc) {
    error.value = `未找到文档：${id}`
    return
  }
  activeId.value = id
  current.value = doc
  destroySwagger()
  html.value = ''
  error.value = ''

  const query = { ...route.query, doc: id }
  if (route.query.doc !== id) {
    await router.replace({ path: route.path, query })
  }

  if (doc.kind === 'markdown') {
    await loadMarkdown(doc.src)
  } else {
    await nextTick()
    await loadSwagger(doc.src)
  }
}

function onSelect(id: string) {
  void showDoc(id)
}

function reloadSwagger() {
  if (current.value?.kind === 'openapi') loadSwagger(current.value.src)
}

function openRaw() {
  if (current.value) window.open(current.value.src, '_blank')
}

function onMarkdownClick(ev: MouseEvent) {
  const t = ev.target as HTMLElement | null
  const a = t?.closest?.('a') as HTMLAnchorElement | null
  if (!a || !current.value) return
  const href = a.getAttribute('href') || ''
  if (href.startsWith('http') || href.startsWith('mailto:') || href.startsWith('#')) return
  const docId = resolveMarkdownLink(current.value.src, href)
  if (docId) {
    ev.preventDefault()
    void showDoc(docId)
    return
  }
  // rewrite relative project-docs links to fetchable absolute path under /project-docs
  if (href.endsWith('.md') || href.includes('.md#')) {
    ev.preventDefault()
    const base = current.value.src.replace(/\/[^/]*$/, '/')
    try {
      const abs = new URL(href, window.location.origin + base).pathname
      void loadMarkdown(abs)
      // temporary view outside catalog
      current.value = {
        ...current.value,
        title: abs.split('/').pop() || current.value.title,
        src: abs,
        id: current.value.id,
      }
    } catch {
      /* ignore */
    }
  }
}

onMounted(() => {
  const fromParam = typeof route.params.docId === 'string' ? route.params.docId : ''
  const fromQuery = typeof route.query.doc === 'string' ? route.query.doc : ''
  const id = fromParam || fromQuery || defaultDocId()
  void showDoc(findDoc(id) ? id : defaultDocId())
})

watch(
  () => route.query.doc,
  (id) => {
    if (typeof id === 'string' && id !== activeId.value && findDoc(id)) {
      void showDoc(id)
    }
  },
)

onBeforeUnmount(() => destroySwagger())
</script>

<style scoped>
.docs-center {
  display: flex;
  flex-direction: column;
  min-height: calc(100vh - 120px);
}
.docs-center.standalone {
  min-height: 100vh;
  background: #f5f7fa;
}
.docs-topbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 16px;
  background: #001529;
  color: #fff;
  font-size: 14px;
}
.docs-topbar a {
  color: #69b1ff;
  text-decoration: none;
}
.docs-body {
  flex: 1;
  min-height: 0;
  background: #fff;
}
.docs-aside {
  border-right: 1px solid #ebeef5;
  background: #fafafa;
  overflow: auto;
}
.aside-head {
  padding: 16px 14px 8px;
}
.aside-head h2 {
  margin: 0 0 4px;
  font-size: 16px;
}
.aside-head p {
  margin: 0 0 10px;
  color: #909399;
  font-size: 12px;
}
.docs-menu {
  border-right: none;
  background: transparent;
}
.menu-title {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
}
.docs-main {
  padding: 16px 20px 24px;
  overflow: auto;
}
.main-toolbar {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: flex-start;
  margin-bottom: 12px;
  flex-wrap: wrap;
}
.main-toolbar h3 {
  margin: 0 0 4px;
  font-size: 18px;
}
.desc {
  margin: 0;
  color: #909399;
  font-size: 13px;
}
.toolbar-actions {
  display: flex;
  gap: 8px;
}
.hint {
  margin-bottom: 12px;
}
.docs-error {
  color: #c45656;
  padding: 8px 12px;
  background: #fff1f0;
  border-radius: 4px;
  margin-bottom: 12px;
}
.swagger-host {
  border: 1px solid #ebeef5;
  border-radius: 8px;
  min-height: 560px;
  background: #fff;
  overflow: auto;
}
.swagger-host :deep(.swagger-ui .topbar) {
  display: none;
}
.md-host {
  border: 1px solid #ebeef5;
  border-radius: 8px;
  padding: 20px 24px;
  background: #fff;
  line-height: 1.65;
  max-width: 960px;
}
.md-host :deep(h1),
.md-host :deep(h2),
.md-host :deep(h3) {
  margin-top: 1.2em;
  margin-bottom: 0.5em;
}
.md-host :deep(h1) {
  font-size: 1.6em;
  border-bottom: 1px solid #ebeef5;
  padding-bottom: 0.3em;
}
.md-host :deep(h2) {
  font-size: 1.3em;
}
.md-host :deep(pre) {
  background: #f6f8fa;
  padding: 12px;
  border-radius: 6px;
  overflow: auto;
}
.md-host :deep(code) {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 0.92em;
}
.md-host :deep(table) {
  border-collapse: collapse;
  width: 100%;
  margin: 12px 0;
}
.md-host :deep(th),
.md-host :deep(td) {
  border: 1px solid #ebeef5;
  padding: 6px 10px;
  text-align: left;
}
.md-host :deep(a) {
  color: #409eff;
}
.md-host :deep(blockquote) {
  margin: 12px 0;
  padding: 4px 12px;
  color: #606266;
  border-left: 4px solid #dcdfe6;
  background: #fafafa;
}
</style>
