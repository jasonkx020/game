import { GameModuleRegistry } from '../../platform/lobby/GameModuleRegistry'
import { registerLiuzichongPushHandlers, liuzichongState } from './LiuzichongPushHandler'
import type { GameModule } from '../../platform/lobby/GameModuleRegistry'

export function registerLiuzichongModule(): void {
  const mod: GameModule = {
    gameId: 'liuzichong',
    entryScene: 'Liuzichong',
    keepsSessionOnLeave: true,
    prepareRoom() {
      liuzichongState.reset()
    },
    registerPush(ctx) {
      registerLiuzichongPushHandlers(ctx.router, {
        log: ctx.log,
        userId: ctx.userId,
      })
    },
    companionHooks: {
      getRulesSummary: () => '六子冲是二人棋类对弈，先连成六子者胜。',
      onPublicEvent(event) {
        if (event === 'onBoardInit') return [{ text: '棋盘摆好啦，先占中间往往更有利～' }]
        return []
      },
    },
  }
  GameModuleRegistry.register(mod)
}
