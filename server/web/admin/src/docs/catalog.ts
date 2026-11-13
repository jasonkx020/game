/** 后台文档中心目录：协议 + 接入（路径相对仓库 docs/ 或 OpenAPI 静态挂载点） */

export type DocKind = 'markdown' | 'openapi'

export interface DocItem {
  id: string
  title: string
  kind: DocKind
  /** markdown: /project-docs/... ; openapi: /openapi/.../openapi.yaml */
  src: string
  description?: string
}

export interface DocSection {
  id: string
  title: string
  items: DocItem[]
}

export const docCatalog: DocSection[] = [
  {
    id: 'integrate',
    title: '快速接入',
    items: [
      {
        id: 'auth-guide',
        title: 'Auth 服务接入',
        kind: 'markdown',
        src: '/project-docs/tech/auth-service.md',
        description: '玩家 sign-in / 微信 / 实名 / 双 Token（Admin 仅登录）',
      },
      {
        id: 'auth-openapi',
        title: 'Auth OpenAPI',
        kind: 'openapi',
        src: '/openapi/auth/openapi.yaml',
        description: '登录体系权威契约（sign-in + is_new_user）',
      },
      {
        id: 'auth-openapi-readme',
        title: 'Auth OpenAPI 说明',
        kind: 'markdown',
        src: '/openapi/auth/README.md',
      },
      {
        id: 'http-signature-auth',
        title: 'HTTP 签名（Auth）',
        kind: 'markdown',
        src: '/openapi/auth/http-signature.md',
      },
      {
        id: 'platform-openapi',
        title: '平台 OpenAPI',
        kind: 'openapi',
        src: '/openapi/platform/openapi.yaml',
        description: '业务 HTTP API（不含 Auth；登录见 Auth OpenAPI）',
      },
      {
        id: 'openapi-readme',
        title: '平台 OpenAPI 说明',
        kind: 'markdown',
        src: '/project-docs/tech/openapi/README.md',
      },
      {
        id: 'http-signature-platform',
        title: 'HTTP 签名（平台）',
        kind: 'markdown',
        src: '/project-docs/tech/openapi/http-signature.md',
      },
      {
        id: 'pitaya-client',
        title: 'Pitaya 客户端接入',
        kind: 'markdown',
        src: '/project-docs/tech/pitaya-client.md',
      },
      {
        id: 'game-host-sdk',
        title: 'Game Host SDK',
        kind: 'markdown',
        src: '/project-docs/tech/game-host-sdk.md',
      },
      {
        id: 'games-readme',
        title: '游戏接入总览',
        kind: 'markdown',
        src: '/project-docs/games/README.md',
      },
    ],
  },
  {
    id: 'protocol',
    title: '协议文档',
    items: [
      {
        id: 'protocol-overview',
        title: '通信协议总览',
        kind: 'markdown',
        src: '/project-docs/tech/protocol.md',
        description: 'HTTP + Pitaya 双通道',
      },
      {
        id: 'proto-readme',
        title: 'Proto / Route 说明',
        kind: 'markdown',
        src: '/project-docs/tech/proto/README.md',
      },
      {
        id: 'proto-pitaya',
        title: 'Pitaya Proto',
        kind: 'markdown',
        src: '/project-docs/tech/proto/pitaya/README.md',
      },
      {
        id: 'proto-ws',
        title: 'Game WebSocket 协议',
        kind: 'markdown',
        src: '/project-docs/tech/proto/game_ws.md',
      },
      {
        id: 'proto-dawugui',
        title: '打乌龟 Proto',
        kind: 'markdown',
        src: '/project-docs/tech/proto/dawugui.md',
      },
      {
        id: 'replay-api',
        title: '战绩回放协议',
        kind: 'markdown',
        src: '/project-docs/tech/replay.md',
      },
      {
        id: 'audit-log',
        title: '有序 Action Log',
        kind: 'markdown',
        src: '/project-docs/tech/audit-action-log.md',
      },
    ],
  },
  {
    id: 'architecture',
    title: '架构与指南',
    items: [
      {
        id: 'tech-index',
        title: '技术文档索引',
        kind: 'markdown',
        src: '/project-docs/tech/README.md',
      },
      {
        id: 'platform-arch',
        title: '平台架构',
        kind: 'markdown',
        src: '/project-docs/tech/platform-architecture.md',
      },
      {
        id: 'client-arch',
        title: '客户端架构',
        kind: 'markdown',
        src: '/project-docs/tech/client-architecture.md',
      },
      {
        id: 'game-framework',
        title: '游戏框架',
        kind: 'markdown',
        src: '/project-docs/tech/game-framework.md',
      },
      {
        id: 'lobby-migration',
        title: '大厅迁出指南',
        kind: 'markdown',
        src: '/project-docs/tech/games-lobby-migration.md',
      },
      {
        id: 'adr-index',
        title: 'ADR 索引',
        kind: 'markdown',
        src: '/project-docs/tech/adr/README.md',
      },
      {
        id: 'adr-001',
        title: 'ADR-001 服务端技术栈',
        kind: 'markdown',
        src: '/project-docs/tech/adr/001-server-stack.md',
      },
      {
        id: 'adr-002',
        title: 'ADR-002 可插拔游戏约束',
        kind: 'markdown',
        src: '/project-docs/tech/adr/002-pluggable-game-constraints.md',
      },
      {
        id: 'adr-004',
        title: 'ADR-004 Pitaya 游戏框架',
        kind: 'markdown',
        src: '/project-docs/tech/adr/004-pitaya-game-framework.md',
      },
      {
        id: 'adr-006',
        title: 'ADR-006 大厅与动态 Bundle',
        kind: 'markdown',
        src: '/project-docs/tech/adr/006-game-lobby-dynamic-bundle.md',
      },
      {
        id: 'adr-008',
        title: 'ADR-008 Game Host SDK',
        kind: 'markdown',
        src: '/project-docs/tech/adr/008-game-host-sdk.md',
      },
    ],
  },
]

export function findDoc(id: string): DocItem | undefined {
  for (const s of docCatalog) {
    const hit = s.items.find((i) => i.id === id)
    if (hit) return hit
  }
  return undefined
}

export function defaultDocId(): string {
  return 'auth-guide'
}

/** 将 markdown 相对链接解析为 catalog 中的文档 id（若可匹配） */
export function resolveMarkdownLink(currentSrc: string, href: string): string | null {
  if (!href || href.startsWith('http') || href.startsWith('mailto:') || href.startsWith('#')) {
    return null
  }
  const baseDir = currentSrc.replace(/\/[^/]*$/, '/')
  let joined = href
  if (href.startsWith('/')) {
    joined = href
  } else {
    const stack = baseDir.replace(/^\/project-docs\//, '').split('/').filter(Boolean)
    for (const part of href.split('/')) {
      if (part === '.' || part === '') continue
      if (part === '..') stack.pop()
      else stack.push(part)
    }
    joined = '/project-docs/' + stack.join('/')
  }
  // strip hash
  joined = joined.split('#')[0]
  for (const s of docCatalog) {
    for (const item of s.items) {
      if (item.kind === 'markdown' && item.src === joined) return item.id
      // also match without /project-docs prefix variants for openapi mounts
      if (item.src === joined) return item.id
    }
  }
  return null
}
