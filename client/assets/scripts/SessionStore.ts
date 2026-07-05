import { ApiClient, CreateRoomResult, LoginResult } from '../platform/sdk/ApiClient'
import type { GameSession } from '../platform/sdk/GameSession'

/** 跨场景共享会话状态（Cocos 场景切换时保留） */
class SessionStoreImpl {
  api = new ApiClient()
  login: LoginResult | null = null
  room: CreateRoomResult | null = null
  session: GameSession | null = null
  pushLogs: string[] = []

  appendLog(line: string): void {
    this.pushLogs.push(line)
    if (this.pushLogs.length > 100) this.pushLogs.shift()
  }

  resetRoom(): void {
    this.room = null
    this.pushLogs = []
  }
}

export const SessionStore = new SessionStoreImpl()
