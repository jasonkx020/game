import { defineConfig, type Plugin } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'
import fs from 'node:fs'
import path from 'node:path'
import type { IncomingMessage, ServerResponse } from 'node:http'

const adminRoot = fileURLToPath(new URL('.', import.meta.url))
const repoRoot = path.resolve(adminRoot, '../../..')
const docsDir = path.resolve(repoRoot, 'docs')
const authOpenapiDir = path.resolve(repoRoot, 'auth-service/openapi')
const platformOpenapiDir = path.resolve(docsDir, 'tech/openapi')

function serveDir(root: string) {
  return (req: IncomingMessage, res: ServerResponse, next: () => void) => {
    try {
      const urlPath = decodeURIComponent((req.url || '/').split('?')[0])
      const filePath = path.normalize(path.join(root, urlPath))
      if (!filePath.startsWith(path.normalize(root + path.sep)) && filePath !== path.normalize(root)) {
        res.statusCode = 403
        res.end('forbidden')
        return
      }
      if (!fs.existsSync(filePath) || fs.statSync(filePath).isDirectory()) {
        next()
        return
      }
      const ext = path.extname(filePath).toLowerCase()
      const types: Record<string, string> = {
        '.yaml': 'text/yaml; charset=utf-8',
        '.yml': 'text/yaml; charset=utf-8',
        '.md': 'text/markdown; charset=utf-8',
        '.json': 'application/json; charset=utf-8',
        '.txt': 'text/plain; charset=utf-8',
      }
      res.setHeader('Content-Type', types[ext] || 'application/octet-stream')
      res.setHeader('Cache-Control', 'no-cache')
      fs.createReadStream(filePath).pipe(res)
    } catch {
      next()
    }
  }
}

function copyDir(src: string, dest: string) {
  if (!fs.existsSync(src)) return
  fs.mkdirSync(dest, { recursive: true })
  for (const name of fs.readdirSync(src)) {
    if (name === 'node_modules' || name === '.git') continue
    const from = path.join(src, name)
    const to = path.join(dest, name)
    if (fs.statSync(from).isDirectory()) copyDir(from, to)
    else fs.copyFileSync(from, to)
  }
}

/** Dev 挂载 docs + OpenAPI；Build 复制到 dist */
function projectDocsPlugin(): Plugin {
  return {
    name: 'project-docs-static',
    configureServer(server) {
      server.middlewares.use('/project-docs', serveDir(docsDir))
      server.middlewares.use('/openapi/auth', serveDir(authOpenapiDir))
      server.middlewares.use('/openapi/platform', serveDir(platformOpenapiDir))
      server.middlewares.use('/auth-service/openapi', serveDir(authOpenapiDir))
    },
    closeBundle() {
      const outDir = path.resolve(adminRoot, 'dist')
      copyDir(docsDir, path.join(outDir, 'project-docs'))
      copyDir(authOpenapiDir, path.join(outDir, 'openapi/auth'))
      copyDir(platformOpenapiDir, path.join(outDir, 'openapi/platform'))
      copyDir(authOpenapiDir, path.join(outDir, 'auth-service/openapi'))
    },
  }
}

export default defineConfig({
  plugins: [vue(), projectDocsPlugin()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    port: 5173,
    fs: {
      allow: [adminRoot, repoRoot, docsDir, authOpenapiDir, platformOpenapiDir],
    },
    proxy: {
      '/v1': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
