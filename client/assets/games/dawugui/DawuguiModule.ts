import { GameModuleRegistry } from '../../platform/lobby/GameModuleRegistry'
import { registerDawuguiPushHandlers } from './DawuguiPushHandler'
import { encodeProto, decodeProto } from '../../platform/sdk/protoHelpers'
import { PassReq, PassRsp, PlayCardsReq, PlayCardsRsp } from '../../platform/generated/pitaya/pitaya/dawugui'
import type { GameModule, GameModuleContext } from '../../platform/lobby/GameModuleRegistry'
import { SessionStore } from '../../scripts/SessionStore'

export function registerDawuguiModule(): void {
  const mod: GameModule = {
    gameId: 'dawugui',
    registerPush(ctx) {
      const log = (line: string) => {
        ctx.log(line)
        handleCompanionEvent(ctx, 'push', line)
      }
      registerDawuguiPushHandlers(ctx.router, log)
    },
    companionHooks: {
      getRulesSummary: () => '打乌龟是 3-5 人扑克跑牌游戏，先出完手牌获胜。注意报单与包牌。',
      getStrategyTips: (phase) =>
        phase === 'playing' ? ['观察对手出牌习惯', '报单后要谨慎'] : ['熟悉牌型大小优先'],
      onPublicEvent(event, payload) {
        if (event === 'onAlert') return [{ text: '有人报单了，注意防守！' }]
        if (event === 'onSettlement') return [{ text: '本局结束啦，要不要再来一局？' }]
        return []
      },
    },
    async onPassClick(ctx) {
      const body = encodeProto(PassReq, { roomId: ctx.roomId })
      const rsp = await ctx.session.pitaya.request('game.dawugui.pass', body)
      decodeProto(PassRsp, rsp)
      ctx.log('[action] pass sent')
    },
    async onPlayClick(ctx) {
      const body = encodeProto(PlayCardsReq, { roomId: ctx.roomId, cards: [1] })
      const rsp = await ctx.session.pitaya.request('game.dawugui.playcards', body)
      decodeProto(PlayCardsRsp, rsp)
      ctx.log('[action] playcards sent')
    },
  }
  GameModuleRegistry.register(mod)
}

function handleCompanionEvent(_ctx: GameModuleContext, _event: string, payload: unknown): void {
  const line = String(payload)
  const mod = GameModuleRegistry.get('dawugui')
  let event = 'push'
  if (line.includes('onAlert')) event = 'onAlert'
  if (line.includes('onSettlement')) event = 'onSettlement'
  const hints = mod?.companionHooks?.onPublicEvent?.(event, payload) ?? []
  for (const h of hints) SessionStore.appendCompanionHint(h.text)
}
