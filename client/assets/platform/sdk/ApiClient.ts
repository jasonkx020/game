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
}

export interface CreateRoomResult {
  roomId: string
  wsUrl: string
  gameId: string
  auditSn?: number
  cost?: number
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

  async getProfile(): Promise<{ userId: number; phone: string; nickname: string; role: string }> {
    const res = await this.get<{ user_id: number; phone: string; nickname: string; role: string }>(
      '/v1/user/profile',
    )
    return { userId: res.user_id, phone: res.phone, nickname: res.nickname, role: res.role }
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
