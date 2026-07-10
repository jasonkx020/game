import { assetManager, sys } from 'cc'

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
    if (this.loaded.has(gameId) && this.isInstalled(gameId, meta.version)) {
      return
    }

    onProgress?.(0, meta.sizeBytes || 1)

    if (meta.url) {
      try {
        await this.loadRemoteBundle(gameId, meta, onProgress)
        this.markInstalled(gameId, meta.version)
        return
      } catch (e) {
        console.warn(`[Bundle] remote load failed for ${gameId}, fallback builtin`, e)
      }
    }

    await this.loadBuiltinModule(gameId)
    this.markInstalled(gameId, meta.version)
    onProgress?.(meta.sizeBytes || 1, meta.sizeBytes || 1)
  }

  private loadRemoteBundle(
    gameId: string,
    meta: BundleMeta,
    onProgress?: BundleProgress,
  ): Promise<void> {
    return new Promise((resolve, reject) => {
      assetManager.loadBundle(
        meta.url,
        { version: meta.version },
        (err, bundle) => {
          if (err || !bundle) {
            reject(err ?? new Error(`loadBundle failed: ${gameId}`))
            return
          }
          this.loaded.set(gameId, bundle)
          bundle.loadDir('/', (finished, total) => {
            onProgress?.(finished, total)
          }, (loadErr) => {
            if (loadErr) {
              reject(loadErr)
              return
            }
            resolve()
          })
        },
      )
    })
  }

  private async loadBuiltinModule(gameId: string): Promise<void> {
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
