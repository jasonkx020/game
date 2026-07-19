import { assetManager, sys } from 'cc'
import { DEV, EDITOR, PREVIEW } from 'cc/env'

export interface BundleMeta {
  version: string
  url: string
  sizeBytes: number
  sha256?: string
  entryScene?: string
}

interface InstalledRecord {
  gameId: string
  version: string
  installedAt: number
}

const STORAGE_KEY = 'games_installed_bundles'

function readInstalled(): Record<string, InstalledRecord> {
  try {
    const raw = sys.localStorage.getItem(STORAGE_KEY)
    if (!raw) return {}
    return JSON.parse(raw) as Record<string, InstalledRecord>
  } catch {
    return {}
  }
}

function writeInstalled(data: Record<string, InstalledRecord>): void {
  sys.localStorage.setItem(STORAGE_KEY, JSON.stringify(data))
}

/** Creator 预览 / 编辑器：禁止跨域 HTTP loadBundle */
function isCreatorPreviewOrDev(): boolean {
  if (PREVIEW || EDITOR || DEV) return true
  if (typeof window === 'undefined' || !window.location) return false
  const port = window.location.port
  return port === '7456' || port === '7457' || port === '7458'
}

function isCrossOriginBundleUrl(url: string): boolean {
  if (!url || typeof window === 'undefined' || !window.location?.href) return false
  try {
    const page = new URL(window.location.href)
    const bundle = new URL(url, page.href)
    return page.origin !== bundle.origin
  } catch {
    return true
  }
}

export type BundleProgress = (loaded: number, total: number) => void

export class GameBundleManager {
  private loaded = new Map<string, unknown>()

  isInstalled(gameId: string, version: string): boolean {
    const rec = readInstalled()[gameId]
    return rec?.version === version
  }

  markInstalled(gameId: string, version: string): void {
    const all = readInstalled()
    all[gameId] = { gameId, version, installedAt: Date.now() }
    writeInstalled(all)
  }

  async ensureLoaded(
    gameId: string,
    meta: BundleMeta,
    onProgress?: BundleProgress,
  ): Promise<void> {
    onProgress?.(0, meta.sizeBytes || 1)

    // 已加载过本地/远程包：仍要确保 Module 已注册（场景切换后 Map 可能还在）
    if (this.loaded.has(gameId) && this.isInstalled(gameId, meta.version)) {
      await this.registerBuiltinModule(gameId)
      onProgress?.(meta.sizeBytes || 1, meta.sizeBytes || 1)
      return
    }

    const skipHttpRemote =
      !meta.url ||
      meta.version === 'builtin' ||
      isCreatorPreviewOrDev() ||
      isCrossOriginBundleUrl(meta.url)

    if (skipHttpRemote) {
      if (meta.url && (isCreatorPreviewOrDev() || isCrossOriginBundleUrl(meta.url))) {
        console.info(
          `[Bundle] skip HTTP remote for ${gameId}; load local AssetBundle + builtin module`,
        )
      }
      // 关键：按包名加载本地 Asset Bundle，否则 Liuzichong.scene 不在内存，进不了游戏
      const localOk = await this.loadLocalNamedBundle(gameId)
      await this.registerBuiltinModule(gameId)
      if (!localOk) {
        console.warn(
          `[Bundle] local AssetBundle "${gameId}" not loaded; loadScene may fail. ` +
            `Check folder is Bundle and not only remote-HTTP.`,
        )
      }
      this.markInstalled(gameId, meta.version === 'builtin' ? 'builtin' : meta.version)
      onProgress?.(meta.sizeBytes || 1, meta.sizeBytes || 1)
      return
    }

    // 同域 HTTP Remote Bundle（正式 Web 包）
    const remoteOk = await this.isRemoteHttpAvailable(meta)
    if (remoteOk) {
      try {
        await this.loadRemoteHttpBundle(gameId, meta, onProgress)
        await this.registerBuiltinModule(gameId)
        this.markInstalled(gameId, meta.version)
        return
      } catch (e) {
        console.warn(`[Bundle] HTTP remote failed for ${gameId}, fallback local`, e)
      }
    }

    const localOk = await this.loadLocalNamedBundle(gameId)
    await this.registerBuiltinModule(gameId)
    if (!localOk) {
      throw new Error(`failed to load game bundle: ${gameId}`)
    }
    this.markInstalled(gameId, meta.version === 'builtin' ? 'builtin' : meta.version)
    onProgress?.(meta.sizeBytes || 1, meta.sizeBytes || 1)
  }

  /** Creator / 主包内：按 Bundle 名加载（含场景） */
  private loadLocalNamedBundle(gameId: string): Promise<boolean> {
    return new Promise((resolve) => {
      const existing = assetManager.getBundle(gameId)
      if (existing) {
        this.loaded.set(gameId, existing)
        resolve(true)
        return
      }
      assetManager.loadBundle(gameId, (err, bundle) => {
        if (err || !bundle) {
          console.warn(`[Bundle] loadBundle("${gameId}") failed`, err)
          resolve(false)
          return
        }
        this.loaded.set(gameId, bundle)
        console.info(`[Bundle] local AssetBundle ready: ${gameId}`)
        resolve(true)
      })
    })
  }

  private async isRemoteHttpAvailable(meta: BundleMeta): Promise<boolean> {
    const base = meta.url.replace(/\/$/, '')
    const indexCandidates = [`${base}/index.${meta.version}.js`, `${base}/index.js`]
    for (const url of indexCandidates) {
      try {
        const res = await fetch(url, { method: 'HEAD' })
        if (res.ok) return true
        if (res.status === 405 || res.status === 501) {
          const getRes = await fetch(url, { method: 'GET' })
          if (getRes.ok) return true
        }
      } catch {
        /* ignore */
      }
    }
    return false
  }

  private loadRemoteHttpBundle(
    gameId: string,
    meta: BundleMeta,
    onProgress?: BundleProgress,
  ): Promise<void> {
    return new Promise((resolve, reject) => {
      assetManager.loadBundle(meta.url, { version: meta.version }, (err, bundle) => {
        if (err || !bundle) {
          reject(err ?? new Error(`loadBundle failed: ${gameId}`))
          return
        }
        this.loaded.set(gameId, bundle)
        bundle.loadDir(
          '/',
          (finished, total) => onProgress?.(finished, total),
          (loadErr) => {
            if (loadErr) reject(loadErr)
            else resolve()
          },
        )
      })
    })
  }

  /** 注册 GameModule（幂等）；不依赖 GameEntry 是否自动执行 */
  private async registerBuiltinModule(gameId: string): Promise<void> {
    switch (gameId) {
      case 'dawugui': {
        const mod = await import('../../games/dawugui/DawuguiModule')
        mod.registerDawuguiModule()
        return
      }
      case 'liuzichong': {
        const mod = await import('../../games/liuzichong/LiuzichongModule')
        mod.registerLiuzichongModule()
        return
      }
      default:
        throw new Error(`no builtin module for game: ${gameId}`)
    }
  }
}

export const gameBundleManager = new GameBundleManager()
