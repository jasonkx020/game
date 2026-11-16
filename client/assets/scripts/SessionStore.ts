import { sys } from 'cc'
import {
  ApiClient,
  CreateRoomResult,
  LoginResult,
  UserProfile,
  isRefreshHardReject,
} from '../platform/sdk/ApiClient'
import type { GameSession } from '../platform/sdk/GameSession'

const TOKEN_STORAGE_KEY = 'games_auth_tokens'
/** Refresh a bit early to avoid racing access expiry on the next request. */
const ACCESS_REFRESH_SKEW_MS = 60_000

type StoredTokens = {
  accessToken: string
  refreshToken: string
  userId: number
  nickname: string
  role: string
  /** Absolute ms when access should be treated as expired (best-effort). */
  accessExpiresAt?: number
}

/** Decode JWT `exp` (seconds) → ms; null if not a readable JWT. */
function readJwtExpMs(token: string): number | null {
  try {
    const parts = token.split('.')
    if (parts.length < 2) return null
    let b64 = parts[1].replace(/-/g, '+').replace(/_/g, '/')
    const pad = b64.length % 4
    if (pad) b64 += '='.repeat(4 - pad)
    if (typeof atob !== 'function') return null
    const payload = JSON.parse(atob(b64)) as { exp?: number }
    return typeof payload.exp === 'number' ? payload.exp * 1000 : null
  } catch {
    return null
  }
}

function resolveAccessExpiresAt(accessToken: string, expiresInSec?: number, storedAt?: number): number | null {
  const fromJwt = readJwtExpMs(accessToken)
  if (fromJwt != null) return fromJwt
  if (typeof expiresInSec === 'number' && expiresInSec > 0 && typeof storedAt === 'number') {
    return storedAt + expiresInSec * 1000
  }
  return null
}

function accessNeedsRefresh(accessToken: string, accessExpiresAt?: number): boolean {
  const exp =
    typeof accessExpiresAt === 'number' && accessExpiresAt > 0
      ? accessExpiresAt
      : readJwtExpMs(accessToken)
  if (exp == null) {
    // Unknown expiry → refresh once to be safe.
    return true
  }
  return Date.now() >= exp - ACCESS_REFRESH_SKEW_MS
}

/** 跨场景共享会话状态；token 持久化到 localStorage，refresh 未过期可自动恢复登录 */
class SessionStoreImpl {
  api = new ApiClient()
  login: LoginResult | null = null
  profile: UserProfile | null = null
  room: CreateRoomResult | null = null
  session: GameSession | null = null
  companionSessionId: number | null = null
  /** Pitaya join 返回的座位；-1 表示未知 */
  mySeat = -1
  pushLogs: string[] = []
  companionHints: string[] = []

  constructor() {
    this.api.setOnTokensUpdated((pair) => {
      this.login = pair
      this.persistTokens(pair)
    })
    // Only when /auth/refresh hard-rejects — never for business 403.
    this.api.setOnSessionInvalid(() => {
      this.login = null
      this.profile = null
      this.clearPersistedTokens()
    })
  }

  private persistTokens(pair: LoginResult): void {
    const accessExpiresAt =
      resolveAccessExpiresAt(pair.accessToken, pair.expiresIn, Date.now()) ?? undefined
    const payload: StoredTokens = {
      accessToken: pair.accessToken,
      refreshToken: pair.refreshToken,
      userId: pair.userId,
      nickname: pair.nickname,
      role: pair.role,
      accessExpiresAt,
    }
    try {
      sys.localStorage.setItem(TOKEN_STORAGE_KEY, JSON.stringify(payload))
    } catch {
      /* ignore quota / private mode */
    }
  }

  private clearPersistedTokens(): void {
    try {
      sys.localStorage.removeItem(TOKEN_STORAGE_KEY)
    } catch {
      /* ignore */
    }
  }

  private readPersistedTokens(): StoredTokens | null {
    try {
      const raw = sys.localStorage.getItem(TOKEN_STORAGE_KEY)
      if (!raw) return null
      const parsed = JSON.parse(raw) as StoredTokens
      if (!parsed?.accessToken || !parsed?.refreshToken) return null
      return parsed
    } catch {
      return null
    }
  }

  private async loadProfileQuietly(): Promise<void> {
    try {
      this.profile = await this.api.getProfile()
      if (this.login && this.profile.nickname) {
        this.login.nickname = this.profile.nickname
      }
    } catch {
      /* profile optional for entering lobby */
    }
  }

  hasPersistedTokens(): boolean {
    return this.readPersistedTokens() != null
  }

  /**
   * Restore session after app/page reload while refresh token is still valid.
   * Reuses access when still valid; only calls /auth/refresh when access is near/past expiry.
   * - ok: session ready
   * - none: no stored tokens
   * - auth: refresh rejected → tokens cleared, need SMS/oauth login
   * - network: transient failure → tokens kept, caller should let user retry
   */
  async tryRestoreSession(): Promise<'ok' | 'none' | 'auth' | 'network'> {
    if (this.login?.accessToken && this.login?.refreshToken) {
      return 'ok'
    }
    const saved = this.readPersistedTokens()
    if (!saved) return 'none'

    this.api.setTokens(saved.accessToken, saved.refreshToken)
    this.login = {
      accessToken: saved.accessToken,
      refreshToken: saved.refreshToken,
      userId: saved.userId,
      nickname: saved.nickname,
      role: saved.role,
    }
    try {
      if (accessNeedsRefresh(saved.accessToken, saved.accessExpiresAt)) {
        const pair = await this.api.refreshTokens()
        this.login = pair
      }
      await this.loadProfileQuietly()
      return 'ok'
    } catch (e) {
      // Drop in-memory session; wipe disk only on refresh hard-reject (not business 403).
      this.api.clearAccessToken()
      this.login = null
      this.profile = null
      if (isRefreshHardReject(e)) {
        console.warn('[SessionStore] restore refresh rejected, need re-login', e)
        this.clearPersistedTokens()
        return 'auth'
      }
      console.warn('[SessionStore] restore network/transient failure, tokens kept', e)
      return 'network'
    }
  }

  appendLog(line: string): void {
    this.pushLogs.push(line)
    if (this.pushLogs.length > 100) this.pushLogs.shift()
  }

  resetRoom(): void {
    this.room = null
    this.mySeat = -1
    this.pushLogs = []
    this.companionHints = []
  }

  appendCompanionHint(hint: string): void {
    this.companionHints.push(hint)
    if (this.companionHints.length > 20) this.companionHints.shift()
  }

  logout(): void {
    this.session?.leave()
    this.session = null
    this.api.clearAccessToken()
    this.login = null
    this.profile = null
    this.companionSessionId = null
    this.clearPersistedTokens()
    this.resetRoom()
  }
}

export const SessionStore = new SessionStoreImpl()
