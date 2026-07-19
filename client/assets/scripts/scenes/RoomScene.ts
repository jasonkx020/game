import { _decorator, Component, director, Label } from 'cc'
import { companionHintsFromEvent } from '../../platform/companion/ProactiveTriggers'
import { GameModuleRegistry } from '../../platform/lobby/GameModuleRegistry'
import type { GameModuleContext } from '../../platform/lobby/GameModuleRegistry'
import { GameSession } from '../../platform/sdk/GameSession'
import { SessionStore } from '../SessionStore'

const { ccclass, property } = _decorator

@ccclass('RoomScene')
export class RoomScene extends Component {
  @property(Label)
  logLabel: Label | null = null

  @property(Label)
  companionHintLabel: Label | null = null

  private moduleCtx: GameModuleContext | null = null

  async start(): Promise<void> {
    if (!SessionStore.login || !SessionStore.room) {
      director.loadScene('Lobby')
      return
    }
    const room = SessionStore.room
    const login = SessionStore.login
    const gameId = room.gameId || 'dawugui'

    const mod = GameModuleRegistry.get(gameId)
    if (!mod) {
      SessionStore.appendLog(`[error] game module not loaded: ${gameId}`)
      this.refreshLog()
      director.loadScene('Lobby')
      return
    }

    SessionStore.session?.leave()
    const session = new GameSession({
      api: SessionStore.api,
      onLog: (line) => {
        SessionStore.appendLog(line)
        this.refreshLog()
      },
    })
    SessionStore.session = session

    const ctx: GameModuleContext = {
      router: session.router,
      session,
      roomId: room.roomId,
      userId: login.userId,
      log: (line) => {
        SessionStore.appendLog(line)
        this.refreshLog()
        this.refreshCompanionFromLog(line)
      },
    }
    this.moduleCtx = ctx

    mod.prepareRoom?.(ctx)
    mod.registerPush(ctx)

    try {
      const entered = await session.enterRoom({
        wsUrl: room.wsUrl,
        roomId: room.roomId,
        accessToken: login.accessToken,
      })
      SessionStore.mySeat = entered.seat
      SessionStore.appendLog(`[room] entered ${room.roomId} game=${gameId} seat=${entered.seat}`)
      this.refreshLog()
      const entryScene = mod.entryScene || room.entryScene
      if (entryScene) {
        try {
          await new Promise<void>((resolve, reject) => {
            director.loadScene(entryScene, (err) => {
              if (err) reject(err instanceof Error ? err : new Error(String(err)))
              else resolve()
            })
          })
        } catch (sceneErr) {
          console.error('[Room] loadScene failed', entryScene, sceneErr)
          SessionStore.appendLog(`[error] loadScene ${entryScene}: ${String(sceneErr)}`)
          this.refreshLog()
        }
      }
    } catch (e) {
      console.error('[Room] enter failed', e)
      SessionStore.appendLog(`[error] ${String(e)}`)
      this.refreshLog()
    }
  }

  onDestroy(): void {
    const gameId = SessionStore.room?.gameId
    const mod = gameId ? GameModuleRegistry.get(gameId) : undefined
    if (!mod?.keepsSessionOnLeave) {
      SessionStore.session?.leave()
      SessionStore.session = null
    }
  }

  async onPassClick(): Promise<void> {
    const mod = this.moduleCtx ? GameModuleRegistry.get(SessionStore.room?.gameId ?? '') : undefined
    if (!mod?.onPassClick || !this.moduleCtx) return
    await mod.onPassClick(this.moduleCtx)
    this.refreshLog()
  }

  async onPlayClick(): Promise<void> {
    const mod = this.moduleCtx ? GameModuleRegistry.get(SessionStore.room?.gameId ?? '') : undefined
    if (!mod?.onPlayClick || !this.moduleCtx) return
    await mod.onPlayClick(this.moduleCtx)
    this.refreshLog()
  }

  private refreshLog(): void {
    if (this.logLabel) {
      this.logLabel.string = SessionStore.pushLogs.slice(-8).join('\n')
    }
    if (this.companionHintLabel) {
      const hints = SessionStore.companionHints.slice(-2)
      this.companionHintLabel.string = hints.length ? `陪玩: ${hints.join(' | ')}` : ''
    }
  }

  private refreshCompanionFromLog(line: string): void {
    const gameId = SessionStore.room?.gameId ?? ''
    let event = 'push'
    if (line.includes('onAlert')) event = 'onAlert'
    if (line.includes('onSettlement')) event = 'onSettlement'
    companionHintsFromEvent(gameId, event, line)
    this.refreshLog()
  }
}
