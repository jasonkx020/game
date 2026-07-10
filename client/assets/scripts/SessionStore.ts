import { ApiClient, CreateRoomResult, LoginResult } from '../platform/sdk/ApiClient'
import type { GameSession } from '../platform/sdk/GameSession'

/** 跨场景共享会话状态（Cocos 场景切换时保留） */
class SessionStoreImpl {
  api = new ApiClient()
  login: LoginResult | null = null
  room: CreateRoomResult | null = null
  session: GameSession | null = null
  companionSessionId: number | null = null
  pushLogs: string[] = []
  companionHints: string[] = []

  appendLog(line: string): void {
    this.pushLogs.push(line)
    if (this.pushLogs.length > 100) this.pushLogs.shift()
  }

  resetRoom(): void {
    this.room = null
    this.pushLogs = []
    this.companionHints = []
  }

  appendCompanionHint(hint: string): void {
    this.companionHints.push(hint)
    if (this.companionHints.length > 20) this.companionHints.shift()
  }
}

export const SessionStore = new SessionStoreImpl()
