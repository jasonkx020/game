import { ApiClient, CreateRoomResult, LoginResult, UserProfile } from '../platform/sdk/ApiClient'
import type { GameSession } from '../platform/sdk/GameSession'

/** 跨场景共享会话状态（Cocos 场景切换时保留） */
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
    this.resetRoom()
  }
}

export const SessionStore = new SessionStoreImpl()
