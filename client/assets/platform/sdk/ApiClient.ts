import { ClientConfig, defaultConfig } from './config'
import { signRequest } from './signature'

export interface LoginResult {
  accessToken: string
  userId: number
  nickname: string
  role: string
}

export interface CreateRoomParams {
  gameId?: string
  roomMode?: string
  playerCount?: number
  clubId?: number
  fillBots?: boolean
}

export interface CreateRoomResult {
  roomId: string
  wsUrl: string
  gameId: string
  auditSn?: number
  cost?: number
  entryScene?: string
}

export interface GameBundleInfo {
  version: string
  url: string
  sizeBytes: number
  sha256?: string
  entryScene?: string
  minHostVersion?: string
}

export interface LobbyGameItem {
  gameId: string
  name: string
  iconUrl?: string
  description?: string
  minPlayers: number
  maxPlayers: number
  visible: boolean
  pinned: boolean
  sortOrder: number
  bundle?: GameBundleInfo
}

export interface LobbyGamePrefUpdate {
  gameId: string
  visible?: boolean
  pinned?: boolean
  sortOrder?: number
}

export interface ApiError {
  code: number
  message: string
  request_id?: string
}

export class ApiClient {
  private token = ''
  private cfg: ClientConfig

  constructor(cfg: Partial<ClientConfig> = {}) {
    this.cfg = { ...defaultConfig, ...cfg }
  }

  get accessToken(): string {
    return this.token
  }

  setAccessToken(token: string): void {
    this.token = token
  }

  async login(phone: string, smsCode: string): Promise<LoginResult> {
    const res = await this.post<{
      access_token: string
      user_id: number
      nickname: string
      role: string
    }>('/v1/auth/login', { phone, sms_code: smsCode })
    this.token = res.access_token
    return {
      accessToken: res.access_token,
      userId: res.user_id,
      nickname: res.nickname,
      role: res.role,
    }
  }

  async getProfile(): Promise<{
    userId: number
    phone: string
    phoneMasked?: string
    nickname: string
    role: string
    avatarUrl?: string
  }> {
    const res = await this.get<{
      user_id: number
      phone: string
      phone_masked?: string
      nickname: string
      role: string
      avatar_url?: string
    }>('/v1/user/profile')
    return {
      userId: res.user_id,
      phone: res.phone,
      phoneMasked: res.phone_masked,
      nickname: res.nickname,
      role: res.role,
      avatarUrl: res.avatar_url,
    }
  }

  async listLobbyGames(): Promise<LobbyGameItem[]> {
    const res = await this.get<{ games: Array<Record<string, unknown>> }>('/v1/lobby/games')
    return (res.games ?? []).map((g) => this.mapLobbyGame(g))
  }

  async updateLobbyGames(games: LobbyGamePrefUpdate[]): Promise<LobbyGameItem[]> {
    const res = await this.put<{ games: Array<Record<string, unknown>> }>('/v1/lobby/games', {
      games: games.map((g) => ({
        game_id: g.gameId,
        ...(g.visible !== undefined ? { visible: g.visible } : {}),
        ...(g.pinned !== undefined ? { pinned: g.pinned } : {}),
        ...(g.sortOrder !== undefined ? { sort_order: g.sortOrder } : {}),
      })),
    })
    return (res.games ?? []).map((g) => this.mapLobbyGame(g))
  }

  async listGames(): Promise<Array<{ gameId: string; name: string; minPlayers: number; maxPlayers: number }>> {
    const res = await this.get<{ games: Array<Record<string, unknown>> }>('/v1/games')
    return (res.games ?? []).map((g) => ({
      gameId: String(g.game_id),
      name: String(g.name),
      minPlayers: Number(g.min_players),
      maxPlayers: Number(g.max_players),
    }))
  }

  async listLobbyRecommendations(): Promise<Array<{ gameId: string; name: string; reason?: string }>> {
    const res = await this.get<{ recommendations: Array<Record<string, unknown>> }>('/v1/lobby/recommendations')
    return (res.recommendations ?? []).map((r) => ({
      gameId: String(r.game_id),
      name: String(r.name),
      reason: r.reason ? String(r.reason) : undefined,
    }))
  }

  async getUserSettings(): Promise<Record<string, unknown>> {
    const res = await this.get<{ settings: Record<string, unknown> }>('/v1/user/settings')
    return res.settings ?? {}
  }

  async updateUserSettings(settings: Record<string, unknown>): Promise<Record<string, unknown>> {
    const res = await this.put<{ settings: Record<string, unknown> }>('/v1/user/settings', { settings })
    return res.settings ?? {}
  }

  async getRoomCardBalance(): Promise<number> {
    const res = await this.get<{ balance: number }>('/v1/wallet/room-card')
    return res.balance
  }

  async createRoom(params: CreateRoomParams = {}): Promise<CreateRoomResult> {
    const res = await this.post<{
      room_id: string
      ws_url: string
      game_id: string
      audit_sn?: number
      cost?: number
    }>('/v1/rooms', {
      game_id: params.gameId ?? 'dawugui',
      room_mode: params.roomMode ?? 'room_card',
      player_count: params.playerCount ?? 4,
      fill_bots: params.fillBots ?? false,
      ...(params.clubId ? { club_id: params.clubId } : {}),
    })
    return {
      roomId: res.room_id,
      wsUrl: res.ws_url,
      gameId: res.game_id,
      auditSn: res.audit_sn,
      cost: res.cost,
    }
  }

  private mapLobbyGame(g: Record<string, unknown>): LobbyGameItem {
    const bundle = g.bundle as Record<string, unknown> | undefined
    return {
      gameId: String(g.game_id),
      name: String(g.name),
      iconUrl: g.icon_url ? String(g.icon_url) : undefined,
      description: g.description ? String(g.description) : undefined,
      minPlayers: Number(g.min_players),
      maxPlayers: Number(g.max_players),
      visible: Boolean(g.visible),
      pinned: Boolean(g.pinned),
      sortOrder: Number(g.sort_order ?? 0),
      bundle: bundle
        ? {
            version: String(bundle.version ?? '1.0.0'),
            url: String(bundle.url ?? ''),
            sizeBytes: Number(bundle.size_bytes ?? 0),
            sha256: bundle.sha256 ? String(bundle.sha256) : undefined,
            entryScene: bundle.entry_scene ? String(bundle.entry_scene) : undefined,
            minHostVersion: bundle.min_host_version ? String(bundle.min_host_version) : undefined,
          }
        : undefined,
    }
  }

  private async put<T>(path: string, body: unknown): Promise<T> {
    return this.request<T>('PUT', path, body)
  }

  private async get<T>(path: string): Promise<T> {
    return this.request<T>('GET', path)
  }

  private async post<T>(path: string, body: unknown): Promise<T> {
    return this.request<T>('POST', path, body)
  }

  private async request<T>(method: string, path: string, body?: unknown): Promise<T> {
    const bodyStr = body !== undefined ? JSON.stringify(body) : ''
    const headers = await signRequest({
      method,
      path,
      body: bodyStr || undefined,
      token: this.token || undefined,
      appId: this.cfg.appId,
      appSecret: this.cfg.appSecret,
    })
    const url = `${this.cfg.apiBaseUrl}${path}`
    const res = await fetch(url, {
      method,
      headers: {
        ...headers,
        ...(bodyStr ? { 'Content-Type': 'application/json' } : {}),
      },
      body: bodyStr || undefined,
    })
    const data = await res.json()
    if (!res.ok) {
      throw data as ApiError
    }
    return data as T
  }
}
