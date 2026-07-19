import { GameModuleRegistry } from '../../platform/lobby/GameModuleRegistry'
import { registerLiuzichongPushHandlers, liuzichongState } from './LiuzichongPushHandler'
import type { GameModule } from '../../platform/lobby/GameModuleRegistry'
import { SessionStore } from '../../scripts/SessionStore'

export function registerLiuzichongModule(): void {
  const mod: GameModule = {
    gameId: 'liuzichong',
    entryScene: 'Liuzichong',
    keepsSessionOnLeave: true,
    prepareRoom() {
      liuzichongState.reset()
      if (SessionStore.mySeat >= 0) {
        liuzichongState.mySeat = SessionStore.mySeat
      }
    },
    registerPush(ctx) {
      if (SessionStore.mySeat >= 0) {
        liuzichongState.mySeat = SessionStore.mySeat
      }
      registerLiuzichongPushHandlers(ctx.router, {
        log: ctx.log,
        userId: ctx.userId,
      })
    },
    companionHooks: {
      getRulesSummary: () =>
        '六子冲：4×4 棋盘，正交走一步。形成「己-己-敌」或「敌-己-己」且敌方外侧为空（边界算空）则吃子。对方≤1 子或无合法步即胜。',
      onPublicEvent(event) {
        if (event === 'onBoardInit') return [{ text: '棋盘摆好啦，黑方先手～注意冲吃外侧要空。' }]
        return []
      },
    },
  }
  GameModuleRegistry.register(mod)
}
