import { ClientConfig, defaultConfig } from './config'
import { signRequest } from './signature'

export interface LoginResult {
  accessToken: string
  refreshToken: string
  userId: number
  nickname: string
  role: string
  expiresIn?: number
  /** true when this call created the account (phone/email sign-in or first WeChat bind) */
  isNewUser?: boolean
}

export interface UserProfile {
  userId: number
  phone: string
  phoneMasked?: string
  nickname: string
  role: string
  avatarUrl?: string
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
  lastPlayedAt?: string
  bundle?: GameBundleInfo
}

export interface LobbyGamePrefUpdate {
  gameId: string
  visible?: boolean
  pinned?: boolean
  sortOrder?: number
}

export interface RechargeOrder {
  productId: string
  amountCny: number
  cards: number
  auditSn: number
  createdAt: string
}

export interface MatchSummary {
  roundId: string
  roomId: string
  roundNo: number
  gameId: string
  status: string
  startedAt: string
  endedAt?: string
  myRuleScore: number
  myCoinDelta: number
  isWinner: boolean
  replayAvailable: boolean
}

export interface ApiError {
  code: number
  message: string
  request_id?: string
}

export class ApiClient {
  private token = ''
  private refresh = ''
  private cfg: ClientConfig
  /** Optional persist hook; called after tokens are successfully updated in memory. */
  private onTokensUpdated: ((pair: LoginResult) => void) | null = null
  private refreshInFlight: Promise<LoginResult> | null = null

  constructor(cfg: Partial<ClientConfig> = {}) {
    this.cfg = { ...defaultConfig, ...cfg }
  }

  get accessToken(): string {
    return this.token
  }

  get refreshToken(): string {
    return this.refresh
  }

  setOnTokensUpdated(cb: ((pair: LoginResult) => void) | null): void {
    this.onTokensUpdated = cb
  }

  setAccessToken(token: string): void {
    this.token = token
  }

  setTokens(access: string, refresh: string): void {
    this.token = access
    this.refresh = refresh
  }

  clearAccessToken(): void {
    this.token = ''
    this.refresh = ''
  }

  async sendCode(params: {
    channel: 'phone' | 'email'
    target: string
    /** prefer `signin`; `register`/`login` accepted as aliases */
    purpose: 'signin' | 'register' | 'login'
  }): Promise<void> {
    await this.authPost('/v1/auth/send-code', params)
  }

  /** Unified player phone/email entry: creates account if missing. */
  async signIn(params: {
    channel: 'phone' | 'email'
    target: string
    code: string
    nickname?: string
  }): Promise<LoginResult> {
    const body =
      params.channel === 'phone'
        ? {
            channel: 'phone' as const,
            target: params.target,
            phone: params.target,
            code: params.code,
            sms_code: params.code,
            ...(params.nickname ? { nickname: params.nickname } : {}),
          }
        : {
            channel: 'email' as const,
            target: params.target,
            email: params.target,
            code: params.code,
            ...(params.nickname ? { nickname: params.nickname } : {}),
          }
    const res = await this.authPost<{
      access_token: string
      refresh_token: string
      user_id: number
      nickname: string
      role: string
      expires_in?: number
      is_new_user?: boolean
    }>('/v1/auth/sign-in', body)
    return this.applyTokenPair(res)
  }

  /** @deprecated use signIn */
  async register(params: {
    channel: 'phone' | 'email'
    target: string
    code: string
    nickname?: string
  }): Promise<LoginResult> {
    return this.signIn(params)
  }

  async login(phone: string, smsCode: string): Promise<LoginResult> {
    return this.signIn({ channel: 'phone', target: phone, code: smsCode })
  }

  /** @deprecated use signIn */
  async loginWithCode(params: {
    channel: 'phone' | 'email'
    target: string
    code: string
  }): Promise<LoginResult> {
    return this.signIn(params)
  }

  /** platform: client-detected `web` | `miniprogram` */
  async loginWeChat(platform: 'web' | 'miniprogram', code: string): Promise<LoginResult> {
    const res = await this.authPost<{
      access_token: string
      refresh_token: string
      user_id: number
      nickname: string
      role: string
      expires_in?: number
      is_new_user?: boolean
    }>('/v1/auth/oauth/wechat', { platform, code })
    return this.applyTokenPair(res)
  }

  async getAuthMe(): Promise<{
    accountId: number
    nickname: string
    phone?: string
    email?: string
    avatarUrl?: string
    realnameStatus: string
    realNameMasked?: string
    idCardMasked?: string
    isMinor: boolean
  }> {
    const res = await this.authGet<{
      account_id: number
      nickname: string
      phone?: string
      email?: string
      avatar_url?: string
      realname_status: string
      real_name_masked?: string
      id_card_masked?: string
      is_minor: boolean
    }>('/v1/auth/me')
    return {
      accountId: res.account_id,
      nickname: res.nickname,
      phone: res.phone,
      email: res.email,
      avatarUrl: res.avatar_url,
      realnameStatus: res.realname_status,
      realNameMasked: res.real_name_masked,
      idCardMasked: res.id_card_masked,
      isMinor: res.is_minor,
    }
  }

  async verifyRealName(realName: string, idCardNo: string): Promise<{
    realnameStatus: string
    realNameMasked?: string
    idCardMasked?: string
    isMinor: boolean
  }> {
    const res = await this.authPost<{
      realname_status: string
      real_name_masked?: string
      id_card_masked?: string
      is_minor: boolean
    }>('/v1/auth/realname/verify', { real_name: realName, id_card_no: idCardNo })
    return {
      realnameStatus: res.realname_status,
      realNameMasked: res.real_name_masked,
      idCardMasked: res.id_card_masked,
      isMinor: res.is_minor,
    }
  }

  private applyTokenPair(res: {
    access_token: string
    refresh_token: string
    user_id: number
    nickname: string
    role: string
    expires_in?: number
    is_new_user?: boolean
  }): LoginResult {
    this.token = res.access_token
    this.refresh = res.refresh_token
    const pair: LoginResult = {
      accessToken: res.access_token,
      refreshToken: res.refresh_token,
      userId: res.user_id,
      nickname: res.nickname,
      role: res.role,
      expiresIn: res.expires_in,
      isNewUser: res.is_new_user,
    }
    this.onTokensUpdated?.(pair)
    return pair
  }

  /**
   * Dual-token refresh: must send current access (even if expired) + refresh.
   * Rotates both tokens on success. Retries with the same old pair on network/5xx
   * (server keeps previous refresh valid during grace). Single-flight for concurrency.
   */
  async refreshTokens(): Promise<LoginResult> {
    if (this.refreshInFlight) {
      return this.refreshInFlight
    }
    this.refreshInFlight = this.doRefreshTokens().finally(() => {
      this.refreshInFlight = null
    })
    return this.refreshInFlight
  }

  private async doRefreshTokens(): Promise<LoginResult> {
    const access = this.token
    const refresh = this.refresh
    if (!access || !refresh) {
      throw { code: 1001, message: 'access_token and refresh_token required' } as ApiError
    }

    const maxAttempts = 3
    let lastErr: unknown
    for (let attempt = 0; attempt < maxAttempts; attempt++) {
      try {
        const res = await this.authPost<{
          access_token: string
          refresh_token: string
          user_id: number
          nickname: string
          role: string
          expires_in?: number
        }>('/v1/auth/refresh', {
          access_token: access,
          refresh_token: refresh,
        })
        // Only mutate memory after a full successful response
        return this.applyTokenPair(res)
      } catch (err) {
        lastErr = err
        if (!isRetryableRefreshError(err) || attempt === maxAttempts - 1) {
          throw err
        }
        await sleepMs(200 * (attempt + 1))
      }
    }
    throw lastErr
  }

  async logout(): Promise<void> {
    if (!this.token) return
    try {
      await this.authPost('/v1/auth/logout', { access_token: this.token })
    } finally {
      this.clearAccessToken()
    }
  }

  async getProfile(): Promise<UserProfile> {
    const res = await this.get<{
      user_id: number
      phone: string
      phone_masked?: string
      nickname: string
      role: string
      avatar_url?: string
    }>('/v1/user/profile')
    return this.mapProfile(res)
  }

  async updateProfile(patch: { nickname?: string; avatarUrl?: string }): Promise<UserProfile> {
    const res = await this.put<{
      user_id: number
      phone: string
      phone_masked?: string
      nickname: string
      role: string
      avatar_url?: string
    }>('/v1/user/profile', {
      ...(patch.nickname !== undefined ? { nickname: patch.nickname } : {}),
      ...(patch.avatarUrl !== undefined ? { avatar_url: patch.avatarUrl } : {}),
    })
    return this.mapProfile(res)
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

  async getGameCoinBalance(gameId: string): Promise<number> {
    const res = await this.get<{ balance: number }>(`/v1/wallet/game-coin/${encodeURIComponent(gameId)}`)
    return res.balance
  }

  async getRechargeHistory(): Promise<RechargeOrder[]> {
    const res = await this.get<{ orders: Array<Record<string, unknown>> }>('/v1/wallet/recharge/history')
    return (res.orders ?? []).map((o) => ({
      productId: String(o.product_id ?? ''),
      amountCny: Number(o.amount_cny ?? 0),
      cards: Number(o.cards ?? 0),
      auditSn: Number(o.audit_sn ?? 0),
      createdAt: String(o.created_at ?? ''),
    }))
  }

  async listMyMatches(params: { gameId?: string; page?: number; pageSize?: number } = {}): Promise<{
    items: MatchSummary[]
    total: number
    page: number
    pageSize: number
  }> {
    const q = new URLSearchParams()
    if (params.gameId) q.set('game_id', params.gameId)
    if (params.page) q.set('page', String(params.page))
    if (params.pageSize) q.set('page_size', String(params.pageSize))
    const qs = q.toString()
    const res = await this.get<{
      items: Array<Record<string, unknown>>
      total: number
      page: number
      page_size: number
    }>(`/v1/users/me/matches${qs ? `?${qs}` : ''}`)
    return {
      items: (res.items ?? []).map((m) => ({
        roundId: String(m.round_id),
        roomId: String(m.room_id),
        roundNo: Number(m.round_no),
        gameId: String(m.game_id),
        status: String(m.status),
        startedAt: String(m.started_at),
        endedAt: m.ended_at ? String(m.ended_at) : undefined,
        myRuleScore: Number(m.my_rule_score ?? 0),
        myCoinDelta: Number(m.my_coin_delta ?? 0),
        isWinner: Boolean(m.is_winner),
        replayAvailable: Boolean(m.replay_available),
      })),
      total: Number(res.total ?? 0),
      page: Number(res.page ?? 1),
      pageSize: Number(res.page_size ?? 20),
    }
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

  async joinRoom(roomId: string): Promise<{ wsUrl: string }> {
    const res = await this.post<{ ws_url: string }>(`/v1/rooms/${roomId}/join`, {})
    return { wsUrl: res.ws_url }
  }

  private mapProfile(res: {
    user_id: number
    phone: string
    phone_masked?: string
    nickname: string
    role: string
    avatar_url?: string
  }): UserProfile {
    return {
      userId: res.user_id,
      phone: res.phone,
      phoneMasked: res.phone_masked,
      nickname: res.nickname,
      role: res.role,
      avatarUrl: res.avatar_url,
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
      lastPlayedAt: g.last_played_at ? String(g.last_played_at) : undefined,
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
    return this.request<T>('PUT', path, body, this.cfg.apiBaseUrl)
  }

  private async get<T>(path: string): Promise<T> {
    return this.request<T>('GET', path, undefined, this.cfg.apiBaseUrl)
  }

  private async post<T>(path: string, body: unknown): Promise<T> {
    return this.request<T>('POST', path, body, this.cfg.apiBaseUrl)
  }

  private async authPost<T>(path: string, body: unknown): Promise<T> {
    return this.request<T>('POST', path, body, this.cfg.authBaseUrl)
  }

  private async authGet<T>(path: string): Promise<T> {
    return this.request<T>('GET', path, undefined, this.cfg.authBaseUrl)
  }

  private async request<T>(
    method: string,
    path: string,
    body?: unknown,
    baseUrl: string = this.cfg.apiBaseUrl,
  ): Promise<T> {
    const bodyStr = body !== undefined ? JSON.stringify(body) : ''
    const headers = await signRequest({
      method,
      path,
      body: bodyStr || undefined,
      token: this.token || undefined,
      appId: this.cfg.appId,
      appSecret: this.cfg.appSecret,
    })
    const url = `${baseUrl}${path}`
    const res = await fetch(url, {
      method,
      headers: {
        ...headers,
        ...(bodyStr ? { 'Content-Type': 'application/json' } : {}),
      },
      body: bodyStr || undefined,
    })
    let data: unknown
    try {
      data = await res.json()
    } catch {
      if (!res.ok) {
        throw { code: res.status, message: `HTTP ${res.status}` } as ApiError
      }
      throw { code: 500, message: 'invalid JSON response' } as ApiError
    }
    if (!res.ok) {
      const err = (data && typeof data === 'object' ? data : {}) as ApiError
      if (typeof err.code !== 'number') {
        err.code = res.status
      }
      if (!err.message) {
        err.message = `HTTP ${res.status}`
      }
      throw err
    }
    return data as T
  }
}

function isRetryableRefreshError(err: unknown): boolean {
  if (!err || typeof err !== 'object') {
    return true
  }
  const code = (err as ApiError).code
  if (typeof code !== 'number') {
    return true
  }
  // Auth / client errors: do not retry (invalid or expired refresh outside grace)
  if (code === 1001 || code === 401 || code === 403 || code === 400) {
    return false
  }
  // HTTP 5xx or transient
  return code >= 500 || code === 0
}

function sleepMs(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}
