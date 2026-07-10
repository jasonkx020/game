import type { PushRouter } from '../sdk/PushRouter'
import type { GameSession } from '../sdk/GameSession'

export interface GameModuleContext {
  router: PushRouter
  session: GameSession
  roomId: string
  userId: number
  log: (line: string) => void
}

export interface CompanionHint {
  text: string
}

export interface GameModule {
  gameId: string
  entryScene?: string
  prepareRoom?(ctx: GameModuleContext): void
  registerPush(ctx: GameModuleContext): void
  onPassClick?(ctx: GameModuleContext): Promise<void>
  onPlayClick?(ctx: GameModuleContext): Promise<void>
  keepsSessionOnLeave?: boolean
  companionHooks?: {
    onPublicEvent?(event: string, payload: unknown): CompanionHint[]
    getRulesSummary?(): string
    getStrategyTips?(phase: string): string[]
  }
}

const modules = new Map<string, GameModule>()

export const GameModuleRegistry = {
  register(mod: GameModule): void {
    modules.set(mod.gameId, mod)
  },

  has(gameId: string): boolean {
    return modules.has(gameId)
  },

  get(gameId: string): GameModule | undefined {
    return modules.get(gameId)
  },

  clear(): void {
    modules.clear()
  },
}
